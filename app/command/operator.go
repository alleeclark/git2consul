package command

import (
	"git2consul/consul"

	"github.com/urfave/cli"
)

var operatorCommand = cli.Command{
	Name:        "operator",
	Usage:       "start a syncing frequency",
	ArgsUsage:   "[flags] <ref>",
	Description: "fetch contents changes and sync to consul",
	Flags:       []cli.Flag{&cli.StringFlag{Name: "name", Value: "git2consul", Usage: "name of the service to register in consul"}},
	Subcommands: []cli.Command{
		cli.Command{
			Name: "register",
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				if err != nil {
					return err
				}
				return consulInteractor.ServiceRegistration(c.GlobalString("name"))

			},
		},
		cli.Command{
			Name: "deregister",
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				if err != nil {
					return err
				}
				return consulInteractor.ServiceDeregistation(c.GlobalString("name"))
			},
		},
		cli.Command{
			Name:   "force-unlock",
			Usage:  "force a consul unlock",
			Hidden: true,
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				if err != nil {
					return err
				}
				consulInteractor.Unlock(c.GlobalString("name"))
				return nil
			},
		},
		cli.Command{
			Name:      "force-lock",
			Usage:     "force a lock on consul",
			Hidden:    true,
			UsageText: "force a lock on consul will force a lock for this service you will have to trigger an unlock for git2consul to run again",
			Action: func(c *cli.Context) error {
				consulInteractor, err := consul.NewHandler(consul.Config(c.GlobalString("consul-addr"), c.GlobalString("consul-token")))
				if err != nil {
					return err
				}
				consulInteractor.Lock(c.GlobalString("name"))
				return nil
			},
		},
	},
}
