package command

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func New() *cli.App {
	app := cli.NewApp()
	app.Name = "git2consul"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{}
	app.Before = func(context *cli.Context) error {
		if context.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	return app
}

var Command = cli.Command{
	Name:  "git2consul",
	Usage: "a syncing service from git to consul",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "git-user", Value: "git2consul", Usage: "Git user for ssh", Required: true},
		&cli.StringFlag{Name: "git-url", Usage: "Git url to clone", Required: true},
		&cli.StringFlag{Name: "git-branch", Value: "master", Usage: "Git branch to run syncing on"},
		&cli.StringFlag{Name: "git-dir", FilePath: "/var/git2consul/data", Usage: "Directory to pull to"},
		&cli.StringFlag{Name: "git-fingerprint-path", FilePath: "/var/git2consul/.ssh/fingerprint", Usage: "git RSA finerprint id", Required: true},
		&cli.StringFlag{Name: "consul-addr", Value: "localhost:8300", EnvVars: []string{"CONSUL_ADDR"}, Usage: "Consul address to write to. Will use agent unless an env is set of CONSUL_ADDR"},
		&cli.StringFlag{Name: "consul-token", Value: "somestillytoken", EnvVars: []string{"CONSUL_TOKEN"}, Usage: "Consul address to write to. Will use agent unless an env is set of CONSUL_TOKEN"},
		&cli.Int64Flag{Name: "interval", Value: 5, Usage: "Sync interval to consul in minutes"},
		&cli.BoolFlag{Name: "force", Value: false, Usage: "Force git2consul to sync the entire repository"},
		&cli.BoolFlag{Name: "metrics", Value: false, Usage: "Send metrics to pushgateway", EnvVars: []string{"GIT2CONSUL_METRICS"}},
		&cli.StringFlag{Name: "pushgateway-addr", Value: "localhost:9091", Usage: "Push Gateway Address for metrics"},
		&cli.StringFlag{Name: "consul-path", Value: "", Usage: "consul path to sync "},
	},
	Subcommands: []*cli.Command{
		&syncCommand,
		&resyncCommand,
	},
}
