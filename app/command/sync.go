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
		&cli.StringFlag{Name: "commit-id", Value: "", Usage: "git commit id to filter by", EnvVars: []string{"GIT2CONSUL_COMMITID"}, Hidden: true},
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
		if c.Bool("metrics") {
			metricsInit(c.String("metrics-port"))
		}
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
		tip, err := gitCollection.Repository.Head()
		if err != nil {
			return err
		}
		startCommit := tip.Target().String()
		for {
			time.Sleep(time.Second * time.Duration(c.Int64("since")))
			logrus.Debug("running sync")
			gitCollection = gitCollection.Pull(
				git.CloneOptions(c.String("git-user"),
					c.String("git-password"),
					c.String("git-ssh-publickey-path"),
					c.String("git-ssh-privatekey-path"),
					c.String("git-ssh-passpharse-path"),
					[]byte(c.String("git-fingerprint-path"))),
				c.String("git-remote"), c.String("git-branch"))
			consulGitReads.Inc()
			diffDetlas := gitCollection.DifftoHead(startCommit)
			consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
			if err != nil {
				logrus.WithError(err).Error("failed connecting to consul")
				consulGitConnectionFailed.Inc()
				continue
			}
			for _, diff := range diffDetlas {
				consulPath := filepath.Join(c.String("consul-path"), diff.NewFile)
				consulPath = strings.TrimLeft(consulPath, "/")
				switch diff.Status {
				case "Deleted":
					if ok, err := consulInteractor.Delete(filepath.Join(c.String("consul-path"), diff.OldFile)); err != nil || !ok {
						logrus.WithError(err).WithFields(
							logrus.Fields{
								"old-file":    diff.OldFile,
								"consul-path": c.String("consul-path"),
							}).Error("failed adding content")
						consulGitSyncedFailed.Inc()
						continue
					}
					consulGitSynced.Inc()
				default:
					if ok, err := consulInteractor.Put(consulPath, bytes.TrimSpace(gitCollection.ReadFile(c.String("git-dir"), diff.NewFile))); err != nil || !ok {
						logrus.WithError(err).WithFields(
							logrus.Fields{
								"new-file":    diff.NewFile,
								"consul-path": c.String("consul-path"),
							}).Error("failed adding content")
						consulGitSyncedFailed.Inc()
						continue
					}
					consulGitSynced.Inc()
				}
				logrus.WithFields(logrus.Fields{
					"delta-status": diff.Status,
					"old-file":     diff.OldFile,
					"new-file":     diff.NewFile,
				}).Info("processed delta")
			}
			head, err := gitCollection.Head()
			defer head.Free()
			if err != nil {
				logrus.WithError(err).Error("failed to get head after run")
			}
			startCommit = head.Target().String()
		}
	},
	After: func(c *cli.Context) error {
		if c.String("post-shell") != "" {
			return exec.Command(c.String("post-shell")).Run()
		}
		return nil
	},
}
