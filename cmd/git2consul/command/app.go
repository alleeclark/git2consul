package command

import (
	"github.com/urfave/cli"
)

func New() *cli.App {
	app := cli.NewApp()
	app.Name = "git2consul"
	app.Version = "0.0.1"
	app.Usage = "a syncing service from git to consul"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "git-user", Value: "git2consul", Usage: "git user for ssh", Required: false},
		&cli.StringFlag{Name: "git-url", Usage: "Git url to clone", Required: false},
		&cli.StringFlag{Name: "git-branch", Value: "master", Usage: "git branch to run syncing on"},
		&cli.StringFlag{Name: "git-dir", FilePath: "/var/git2consul/data", Usage: "directory to pull to"},
		&cli.StringFlag{Name: "git-fingerprint-path", FilePath: "/var/git2consul/.ssh/fingerprint", Usage: "git RSA finerprint id", Required: false},
		&cli.StringFlag{Name: "consul-addr", Value: "localhost:8300", EnvVars: []string{"CONSUL_ADDR"}, Usage: "consul address to write to. Will use agent unless an env is set of CONSUL_ADDR"},
		&cli.StringFlag{Name: "consul-token", Value: "somestillytoken", EnvVars: []string{"CONSUL_TOKEN"}, Usage: "consul address to write to. Will use agent unless an env is set of CONSUL_TOKEN"},
		&cli.Int64Flag{Name: "interval", Value: 5, Usage: "sync interval to consul in minutes"},
		&cli.BoolFlag{Name: "force", Value: false, Usage: "force git2consul to sync the entire repository"},
		&cli.BoolFlag{Name: "metrics", Value: false, Usage: "send metrics to pushgateway", EnvVars: []string{"GIT2CONSUL_METRICS"}},
		&cli.StringFlag{Name: "pushgateway-addr", Value: "localhost:9091", Usage: "push Gateway Address for metrics"},
		&cli.StringFlag{Name: "consul-path", Value: "", Usage: "consul path to sync "},
	}
	/* 	app.Before = func(context *cli.Context) error {
		if context.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	} */
	app.Commands = []*cli.Command{&syncCommand, &resyncCommand}
	return app
}
