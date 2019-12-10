package main

import (
	"fmt"
	"git2consul/app/command"
	"os"
)

func main() {
	app := command.New()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "git2consul:  %s\n", err)
		os.Exit(1)
	}
}
