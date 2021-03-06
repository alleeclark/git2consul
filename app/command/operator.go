package command

import (
	"git2consul/consul"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var operatorCommand = cli.Command{
	Name:        "operator",
	Usage:       "deregister or register the git2consul service in consul",
	ArgsUsage:   "[flags] <ref>",
	Description: "management commands",
	Flags:       []cli.Flag{&cli.StringFlag{Name: "service-id", Value: "git2consul", Usage: "name of the service to register in consul"}},
	Subcommands: []*cli.Command{
		{
			Name: "register",
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
				if err != nil {
					return err
				}
				return consulInteractor.ServiceRegistration(c.String("service-id"))

			},
		},
		{
			Name: "deregister",
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
				if err != nil {
					return err
				}
				return consulInteractor.ServiceDeregistation(c.String("service-id"))
			},
		},
		{
			Name:   "force-unlock",
			Usage:  "force a consul unlock",
			Hidden: true,
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
				if err != nil {
					return err
				}
				consulInteractor.Unlock(c.String("service-id"))
				return nil
			},
		},
		{
			Name:      "force-lock",
			Usage:     "force a lock on consul",
			Hidden:    true,
			UsageText: "force a lock on consul will force a lock for this service you will have to trigger an unlock for git2consul to run again",
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.String("consul-addr"), c.String("consul-token")))
				if err != nil {
					return err
				}
				consulInteractor.Lock(c.String("service-id"))
				return nil
			},
		},
	},
}

func setLog(context *cli.Context) error {
	l := context.String("log-level")
	lvl, err := logrus.ParseLevel(l)
	if err != nil {
		return err
	}
	if context.String("log-format") == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}
	logrus.SetLevel(lvl)
	if err := os.MkdirAll(filepath.Dir(context.String("log-file")), 0666); err != nil {
		logrus.WithField("error", err).Error("unable to create log directory")
		logrus.SetOutput(os.Stdout)
		return nil
	}
	file, err := os.OpenFile(context.String("log-file"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.WithField("error", err).Error("unable to create log file")
		logrus.SetOutput(os.Stdout)
		return nil
	}
	logrus.SetOutput(
		io.MultiWriter(os.Stdout, file),
	)
	return nil
}
