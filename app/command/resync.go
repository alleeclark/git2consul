package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		repo := git.NewRepository(git.Username(c.String("git-user")), git.Password(c.String("git-password")),
			git.URL(c.String("git-url")),
			git.PullDir(c.String("git-dir")),
		)
		if repo == nil {
			return cli.Exit("could not intialize the repo", 1)
		}
		consulGitReads.Inc()
		err := filepath.Walk(c.String("git-dir"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.WithError(err).WithField("git-dir", c.String("git-dir")).Error("failed to walk the directory")
				return err
			}
			if !info.IsDir() {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					logrus.WithFields(
						logrus.Fields{"path": path, "error": err},
					).Error("failed reading file")
				}
				consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
				consulGitConnectionFailed.Inc()
				if err != nil {
					logrus.WithError(err).Error("failed connecting to consul")
				}
				path = strings.TrimPrefix(path, c.String("git-dir"))
				consulPath := c.String("consul-path") + path
				consulPath = strings.TrimLeft(consulPath, "/")
				if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(contents)); err != nil || !ok {
					logrus.WithFields(logrus.Fields{
						"path":        path,
						"consul-path": consulPath,
						"error":       err,
					}).Error("failed adding contents")
					consulGitSyncedFailed.Inc()
					return nil
				}
				consulGitSynced.Inc()
			}
			return nil
		})
		if err != nil {
			logrus.WithField("directory", c.String("git-dir")).Error("failed to read repository's path %s and sync to consul")
			return cli.NewExitError(err.Error, 1)
		}
		return nil
	},
}
