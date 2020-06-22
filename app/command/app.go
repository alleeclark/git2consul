package command

import (
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

//New cli application for git2consul commands
func New() *cli.App {
	app := cli.NewApp()
	app.Name = "git2consul"
	app.Version = "0.0.4"
	app.Usage = "a syncing service from git to consul"

	app.Flags = []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-user", Value: "git2consul", Usage: "git username", Required: false}),
		&cli.StringFlag{Name: "git-password", Value: "", Usage: "git password", Required: false},
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-url", Usage: "git url to clone", Required: false}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-branch", Value: "master", Usage: "git branch to run syncing on"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-dir", Value: "/var/git2consul/data", Usage: "directory to pull to"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-ssh-publickey-path", Usage: "public key for ssh agent"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-ssh-privatekey-path", Usage: "private key for ssh agent"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-ssh-passphrase-path", Usage: "passpharse for sshkey"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-remote", Value: "origin"}),
		altsrc.NewBoolFlag(&cli.BoolFlag{Name: "git-ssh-file", Usage: "read public, private, and passpharse for ssh agent", Hidden: true}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "git-fingerprint-path", FilePath: "/var/git2consul/.ssh/fingerprint", Usage: "git RSA finerprint id", Required: false}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "consul-addr", Value: "localhost:8500", EnvVars: []string{"CONSUL_ADDR"}, Usage: "consul address to write to. Will use agent unless an env is set of CONSUL_ADDR"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "consul-path", Value: "", Usage: "consul path to sync "}),
		&cli.StringFlag{Name: "consul-token", Value: "somestillytoken", EnvVars: []string{"CONSUL_TOKEN"}, Usage: "consul address to write to. Will use agent unless an env is set of CONSUL_TOKEN"},
		altsrc.NewBoolFlag(&cli.BoolFlag{Name: "metrics", Usage: "send metrics to pushgateway", EnvVars: []string{"GIT2CONSUL_METRICS"}, Hidden: true}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "metrics-port", Value: "2112", EnvVars: []string{"GIT2CONSUL_METRICS_PORT"}}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "pushgateway-addr", Value: "localhost:9091", Usage: "push gateway address for metrics", Hidden: true}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "log-level,l", Usage: "set the logging level [trace, debug, info, warn, error, fatal, panic]", Value: "debug"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "log-file", Usage: "logfile path", Value: "/var/git2consul/logs/git2consul.log"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "log-format", Usage: "json", Value: "text"}),
		&cli.StringFlag{Name: "config-file,c", Usage: "configuration file to read non sensative variables from must be in toml format"},
	}
	app.Before = func(c *cli.Context) error {

		if c.String("config-file") != "" {
			logrus.Debug("found a config file and attempting to apply input source values")
			inputSource := altsrc.NewTomlSourceFromFlagFunc("config-file")
			inputSourceCtx, err := inputSource(c)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			return altsrc.ApplyInputSourceValues(c, inputSourceCtx, c.App.Flags)
		}
		return nil
	}
	app.Commands = []*cli.Command{&operatorCommand, &syncCommand, &resyncCommand}
	return app
}
