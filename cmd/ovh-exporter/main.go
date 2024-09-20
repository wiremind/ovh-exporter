package main

import (
	"log/slog"
	"os"

	"github.com/urfave/cli"
	"github.com/wiremind/ovh-exporter/pkg/cmd"
	"github.com/wiremind/ovh-exporter/pkg/credentials"
	"github.com/wiremind/ovh-exporter/pkg/network"
)

func main() {
	var exitValue int

	app := cli.NewApp()
	app.Name = "ovh-exporter"
	app.Usage = "Prometheus exporter for the OVH API"
	app.Version = cmd.Version

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "exit, e",
			Value:       1,
			Usage:       "value returned on error",
			Destination: &exitValue,
		},
	}

	app.Commands = []cli.Command{
		credentials.GenerateCommand,
		network.ServeCommand,
	}

	err := app.Run(os.Args)
	if err != nil {
		slog.Error(
			"main() error",
			slog.String("error", err.Error()),
		)
		os.Exit(exitValue)
	}
}
