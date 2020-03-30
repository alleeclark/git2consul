package command

import (
	cli "github.com/urfave/cli/v2"
)

//New cli application for git2consul commands
func New() *cli.App {
	app := cli.NewApp()
	app.Name = "git2consul"
	app.Version = "0.0.3"
	app.Usage = "a syncing service from git to consul"

	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "git-user", Value: "git2consul", Usage: "git username", Required: false},
		&cli.StringFlag{Name: "git-password", Value: "", Usage: "git password", Required: false},
		&cli.StringFlag{Name: "git-url", Usage: "git url to clone", Required: false},
		&cli.StringFlag{Name: "git-branch", Value: "master", Usage: "git branch to run syncing on"},
		&cli.StringFlag{Name: "git-dir", Value: "/var/git2consul/data", Usage: "directory to pull to"},
		&cli.StringFlag{Name: "git-ssh-publickey-path", Usage: "public key for ssh agent"},
		&cli.StringFlag{Name: "git-ssh-privatekey-path", Usage: "private key for ssh agent"},
		&cli.StringFlag{Name: "git-ssh-passphrase-path", Usage: "passpharse for sshkey"},
		&cli.StringFlag{Name: "git-remote", Value: "origin"},
		&cli.BoolFlag{Name: "git-ssh-file", Usage: "read public, private, and passpharse for ssh agent", Hidden: true},
		&cli.StringFlag{Name: "git-fingerprint-path", FilePath: "/var/git2consul/.ssh/fingerprint", Usage: "git RSA finerprint id", Required: false},
		&cli.StringFlag{Name: "consul-addr", Value: "localhost:8500", EnvVars: []string{"CONSUL_ADDR"}, Usage: "consul address to write to. Will use agent unless an env is set of CONSUL_ADDR"},
		&cli.StringFlag{Name: "consul-path", Value: "", Usage: "consul path to sync "},
		&cli.StringFlag{Name: "consul-token", Value: "somestillytoken", EnvVars: []string{"CONSUL_TOKEN"}, Usage: "consul address to write to. Will use agent unless an env is set of CONSUL_TOKEN"},
		&cli.BoolFlag{Name: "metrics", Usage: "send metrics to pushgateway", EnvVars: []string{"GIT2CONSUL_METRICS"}, Hidden: true},
		&cli.StringFlag{Name: "pushgateway-addr", Value: "localhost:9091", Usage: "push gateway address for metrics", Hidden: true},
		&cli.StringFlag{Name: "log-level,l", Usage: "set the logging level [trace, debug, info, warn, error, fatal, panic]", Value: "debug"},
		&cli.StringFlag{Name: "log-file", Usage: "logfile path", Value: "/var/git2consul/logs/git2consul.log"},
		&cli.StringFlag{Name: "log-format", Usage: "json", Value: "text"},
	}
	app.Commands = []*cli.Command{&operatorCommand, &syncCommand, &resyncCommand}
	return app
}
