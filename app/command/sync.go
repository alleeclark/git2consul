package command

import (
	"bytes"
	"fmt"
	"git2consul/consul"
	"git2consul/git"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var syncCommand = cli.Command{
	Name:        "sync",
	Usage:       "start a syncing frequency",
	ArgsUsage:   "[flags] <ref>",
	Description: "fetch contents changes and sync to consul",
	Flags:       []cli.Flag{&cli.Int64Flag{Name: "interval", Value: 5, Usage: "sync interval to consul in minutes"}},
	Action: func(c *cli.Context) error {
		pusher := push.New("git2consul", c.String("push-gateway"))
		gitCollection := git.NewRepository(git.Username(c.GlobalString("git-user")), git.URL(c.GlobalString("git-url")), git.PullDir(c.GlobalString("git-dir")))
		logrus.Infoln("cloned git repository")
		consulGitReads.Inc()
		if err := pusher.Collector(consulGitReads).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		ticker := time.NewTicker(time.Duration(c.Int64("interval")) * time.Minute)
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
				if err := pusher.Collector(consulGitReads).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
				fileChanges := gitCollection.ListFileChanges(c.GlobalString("git-dir"))
				consulInteractor, err := consul.NewConsulHandler(consul.ConsulConfig(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				if err != nil {
					logrus.Warningf("failed connecting to consul %v", err)
					consulGitConnectionFailed.Inc()
					if err := pusher.Collector(consulGitConnectionFailed).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
						fmt.Fprintln(os.Stderr, err)
					}
				}
				for key, val := range fileChanges {
					consulPath := c.GlobalString("consul-path") + key
					if consulPath[0:1] == "/" {
						consulPath = consulPath[1:len(consulPath)]
					}
					if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(val)); err != nil || !ok {
						logrus.Warningf("frror adding content %s %v ", key, err)
						consulGitSyncedFailed.Inc()
						if err := pusher.Collector(consulGitSyncedFailed).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
							fmt.Fprintln(os.Stderr, err)
						}
					} else {
						consulGitSynced.Inc()
						if err := pusher.Collector(consulGitSynced).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
							fmt.Fprintln(os.Stderr, err)
						}
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
