package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"git2consul/consul"
	"git2consul/git"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
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
		pusher := push.New("git2consul", c.String("pushgateway-addr"))
		git.NewRepository(git.Username(c.String("git-user")),
			git.URL(c.String("git-url")),
			git.PullDir(c.String("git-dir")),
		)
		logrus.Infoln("Cloned git repository")
		consulGitReads.Inc()
		if err := pusher.Collector(consulGitReads).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		err := filepath.Walk(c.String("git-dir"), func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					logrus.Warningf("Error reading file %s: %v", path, err)
				}
				consulInteractor, err := consul.NewConsulHandler(consul.ConsulConfig(c.String("consul-addr"), c.String("consul-token")))
				consulGitConnectionFailed.Inc()
				if err := pusher.Collector(consulGitConnectionFailed).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
				if err != nil {
					logrus.Warningf("Error could not connect to consul %v", err)
				}
				if ok, err := consulInteractor.Put(c.String("consul-path")+strings.SplitAfterN(c.String("git-dir"), "/", 3)[2], bytes.TrimSpace(contents)); err != nil || !ok {
					logrus.Warningf("Error adding contents %s %v ", path, err)
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
			return nil
		})
		if err != nil {
			logrus.Warningf("Failed to read repository's path %s and sync to consul", c.String("git-dir"))
			return err
		}
		return nil
	},
}
