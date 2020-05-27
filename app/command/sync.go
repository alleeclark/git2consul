package command

import (
	"bytes"
	"git2consul/consul"
	"git2consul/git"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var syncCommand = cli.Command{
	Name:        "sync",
	Usage:       "start a syncing frequency",
	ArgsUsage:   "[flags] <ref>",
	Description: "fetch contents changes and sync to consul",
	Flags: []cli.Flag{
		&cli.Int64Flag{Name: "since", Value: 30, Usage: "sync interval to consul in seconds"},
		&cli.StringFlag{Name: "commit-id", Value: "", Usage: "git commit id to filter by", EnvVars: []string{"GIT2CONSUL_COMMITID"}},
		&cli.StringFlag{Name: "pre-shell", Value: "", Usage: "shell command to execute before syncing", Hidden: true},
		&cli.StringFlag{Name: "post-shell", Value: "", Usage: "shell command to execute after syncing", Hidden: true}},
	Before: func(c *cli.Context) error {
		if c.String("pre-shell") != "" {
			return exec.Command(c.String("pre-shell")).Run()
		}
		return nil

	},
	Action: func(c *cli.Context) error {
		setLog(c)
		gitCollection := git.NewRepository(git.Username(c.String("git-user")),
			git.Password(c.String("git-password")),
			git.URL(c.String("git-url")),
			git.PullDir(c.String("git-dir")),
			git.PublicKeyPath(c.String("git-ssh-publickey-path")),
			git.PrivateKeyPath(c.String("git-ssh-privatekey-path")),
		)
		if gitCollection == nil {
			return cli.NewExitError("did not get git repository", 1)
		}
		consulGitReads.Inc()
		logrus.Debug("cloned the repository")
		for {

			time.Sleep(time.Second * time.Duration(c.Int64("since")))
			logrus.Debug("running sync")
			//since := time.Second * -time.Duration(c.Int64("since"))
			//past := time.Now().Add(since)
			//logrus.Infof("past time is %s", past.UTC().String())
			gitCollection = gitCollection.Pull(
				git.CloneOptions(c.String("git-user"),
					c.String("git-password"),
					c.String("git-ssh-publickey-path"),
					c.String("git-ssh-privatekey-path"),
					c.String("git-ssh-passpharse-path"),
					[]byte(c.String("git-fingerprint-path"))),
				c.String("git-remote"), c.String("git-branch")).Filter(git.ByBranch(c.String("git-branch"))).Filter(git.ByTopo()).Filter(git.ByHead())
			consulGitReads.Inc()
			fileChanges := gitCollection.ListFileChangesV2(c.String("git-dir"))
			if len(fileChanges) == 0 {
				logrus.Debugln("no File changes")
				continue
			}
			//need to clean up
			consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
			if err != nil {
				logrus.WithError(err).Warning("failed connecting to consul")
				consulGitConnectionFailed.Inc()
			}

			for key, val := range fileChanges {
				consulPath := filepath.Join(c.String("consul-path"), key)
				consulPath = strings.TrimLeft(consulPath, "/")
				if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(val)); err != nil || !ok {
					logrus.WithError(err).WithFields(
						logrus.Fields{
							"key": key,
						}).Warning("failed adding content")
					consulGitSyncedFailed.Inc()
				} else {
					consulGitSynced.Inc()
				}
			}
			logrus.WithField("fileschanged", len(fileChanges)).Info("synced")
		}
	},
	After: func(c *cli.Context) error {
		if c.String("post-shell") != "" {
			return exec.Command(c.String("post-shell")).Run()
		}
		return nil
	},
}
