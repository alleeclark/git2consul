package command

import (
	"bytes"
	"git2consul/consul"
	"git2consul/git"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var syncCommand = cli.Command{
	Name:        "sync",
	Usage:       "start a syncing frequency",
	ArgsUsage:   "[flags] <ref>",
	Description: "fetch contents changes and sync to consul",
	Flags:       []cli.Flag{&cli.Int64Flag{Name: "interval", Value: 20, Usage: "sync interval to consul in seconds"}},
	Action: func(c *cli.Context) error {
		defer func() {
			if c.GlobalBool("metrics") {
				pushMetrics(c.GlobalString("pushgateway-addr"))
			}
		}()
		gitCollection := git.NewRepository(git.Username(c.GlobalString("git-user")), git.URL(c.GlobalString("git-url")), git.PullDir(c.GlobalString("git-dir")))
		logrus.Infoln("Cloned git repository")
		consulGitReads.Inc()
		ticker := time.NewTicker(time.Duration(c.Int64("interval")) * time.Second)
		currentTime := time.Now()
		quit := make(chan struct{})
		for {
			select {
			case <-ticker.C:
				logrus.Infoln("git2consul sync running running")
				if len(gitCollection.Commits) > 0 {
					gitCollection.Commits = nil
					gitCollection.Ref = nil
				}
				gitCollection = gitCollection.Fetch(git.CloneOptions(c.GlobalString("git-user"), []byte(c.GlobalString("git-fingerprint-path"))), c.GlobalString("git-branch")).Filter(git.ByBranch(c.GlobalString("git-branch"))).Filter(git.ByDate(currentTime))
				consulGitReads.Inc()
				fileChanges := gitCollection.ListFileChanges(c.GlobalString("git-dir"))
				consulInteractor, err := consul.NewConsulHandler(consul.ConsulConfig(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				if err != nil {
					logrus.Warningf("Failed connecting to consul %v", err)
					consulGitConnectionFailed.Inc()
				}
				for key, val := range fileChanges {
					consulPath := c.GlobalString("consul-path") + key
					if consulPath[0:1] == "/" {
						consulPath = consulPath[1:len(consulPath)]
					}
					if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(val)); err != nil || !ok {
						logrus.Warningf("Failed adding content %s %v ", key, err)
						consulGitSyncedFailed.Inc()
					} else {
						consulGitSynced.Inc()
					}
				}
			case <-quit:
				ticker.Stop()
				break
			}
			break
		}
		return nil
	},
}
