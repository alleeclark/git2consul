package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"git2consul/consul"
	"git2consul/git"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var resyncCommand = cli.Command{
	Name:        "resync",
	Aliases:     []string{"force"},
	Usage:       "start a full resync",
	ArgsUsage:   "[flags] <ref>",
	Description: "fetch content changes from git and sync to consul",
	Action: func(c *cli.Context) error {
		defer func() {
			if c.GlobalBool("metrics") {
				pushMetrics(c.GlobalString("pushgateway-addr"))
			}
		}()
		git.NewRepository(git.Username(c.GlobalString("git-user")),
			git.URL(c.GlobalString("git-url")),
			git.PullDir(c.GlobalString("git-dir")),
		)
		logrus.WithField("git-url", c.GlobalString("git-url")).Infoln("Cloned git repository")
		consulGitReads.Inc()
		err := filepath.Walk(c.GlobalString("git-dir"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.WithField("git-dir", c.GlobalString("git-dir")).Fatalf("Failed to walk the directory %v", err)
				return err
			}
			if !info.IsDir() {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					logrus.Warningf("Failed reading file %s: %v", path, err)
				}
				consulInteractor, err := consul.NewConsulHandler(consul.ConsulConfig(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				consulGitConnectionFailed.Inc()
				if err != nil {
					logrus.Warningf("Failed connecting to consul %v", err)
				}
				consulPath := c.GlobalString("consul-path") + path
				if consulPath[0:1] == "/" {
					consulPath = consulPath[1:len(consulPath)]
				}
				if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(contents)); err != nil || !ok {
					logrus.Warningf("Failed adding contents %s %v ", path, err)
					consulGitSyncedFailed.Inc()
				} else {
					consulGitSynced.Inc()
				}
			}
			return nil
		})
		if err != nil {
			logrus.Warningf("Failed to read repository's path %s and sync to consul", c.GlobalString("git-dir"))
			return err
		}
		return nil
	},
}
