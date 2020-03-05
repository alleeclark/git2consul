package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
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
	Flags: []cli.Flag{
		cli.StringFlag{Name: "pre-shell", Value: "", Usage: "shell command to execute before syncing", Hidden: true},
		cli.StringFlag{Name: "post-shell", Value: "", Usage: "shell command to execute after syncing", Hidden: true},
	},
	Before: func(c *cli.Context) error {
		if c.String("pre-shell") != "" {
			return exec.Command(c.String("shell")).Run()
		}
		return nil
	},
	After: func(c *cli.Context) error {
		if c.String("post-shell") != "" {
			return exec.Command(c.String("post-shell")).Run()
		}
		return nil
	},
	Action: func(c *cli.Context) error {
		defer func() {
			if c.GlobalBool("metrics") {
				pushMetrics(c.GlobalString("pushgateway-addr"))
			}
		}()
		git.NewRepository(git.Username(c.GlobalString("git-user")), git.Password(c.GlobalString("git-password")),
			git.URL(c.GlobalString("git-url")),
			git.PullDir(c.GlobalString("git-dir")),
		)
		logrus.WithField("git-url", c.GlobalString("git-url")).Infoln("Cloned git repository")
		consulGitReads.Inc()
		err := filepath.Walk(c.GlobalString("git-dir"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.WithField("git-dir", c.GlobalString("git-dir")).Fatalf("failed to walk the directory %v", err)
				return err
			}
			if !info.IsDir() {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					logrus.Warningf("failed reading file %s: %v", path, err)
				}
				consulInteractor, err := consul.NewHandler(consul.Config(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				consulGitConnectionFailed.Inc()
				if err != nil {
					logrus.Warningf("failed connecting to consul %v", err)
				}
				consulPath := c.GlobalString("consul-path") + path
				if consulPath[0:1] == "/" {
					consulPath = consulPath[1:len(consulPath)]
				}
				if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(contents)); err != nil || !ok {
					logrus.Warningf("failed adding contents %s %v ", path, err)
					consulGitSyncedFailed.Inc()
				} else {
					consulGitSynced.Inc()
				}
			}
			return nil
		})
		if err != nil {
			logrus.Warningf("failed to read repository's path %s and sync to consul", c.GlobalString("git-dir"))
			return err
		}
		return nil
	},
}
