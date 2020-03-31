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
	"github.com/urfave/cli/v2"
)

var resyncCommand = cli.Command{
	Name:        "resync",
	Aliases:     []string{"force"},
	Usage:       "start a full resync",
	ArgsUsage:   "[flags] <ref>",
	Description: "fetch content changes from git and sync to consul",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "pre-shell", Value: "", Usage: "shell command to execute before syncing", Hidden: true},
		&cli.StringFlag{Name: "post-shell", Value: "", Usage: "shell command to execute after syncing", Hidden: true},
	},
	Before: func(c *cli.Context) error {
		if c.String("pre-shell") != "" {
			return exec.Command(c.String("pre-shell")).Run()
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
		setLog(c)
		defer func() {
			if c.Bool("metrics") {
				pushMetrics(c.String("pushgateway-addr"))
			}
		}()
		git.NewRepository(git.Username(c.String("git-user")), git.Password(c.String("git-password")),
			git.URL(c.String("git-url")),
			git.PullDir(c.String("git-dir")),
		)
		logrus.WithField("git-url", c.String("git-url")).Infoln("cloned git repository")
		consulGitReads.Inc()
		err := filepath.Walk(c.String("git-dir"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.WithField("git-dir", c.String("git-dir")).Warningf("failed to walk the directory %v", err)
				return err
			}
			if !info.IsDir() {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					logrus.WithFields(
						logrus.Fields{"path": path, "error": err},
					).Warning("failed reading file")
				}
				consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
				consulGitConnectionFailed.Inc()
				if err != nil {
					logrus.WithField("error", err).Warning("failed connecting to consul")
				}
				consulPath := c.String("consul-path") + path
				if consulPath[0:1] == "/" {
					consulPath = consulPath[1:len(consulPath)]
				}
				if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(contents)); err != nil || !ok {
					logrus.WithFields(logrus.Fields{
						"path":        path,
						"consul-path": consulPath,
						"error":       err,
					}).Warning("failed adding contents")
					consulGitSyncedFailed.Inc()
				} else {
					consulGitSynced.Inc()
				}
			}
			return nil
		})
		if err != nil {
			logrus.WithField("directory", c.String("git-dir")).Fatal("failed to read repository's path %s and sync to consul")
			return cli.NewExitError(err.Error, 1)
		}
		return nil
	},
}
