package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/wiremind/ovh-exporter/pkg/cmd"
	"github.com/wiremind/ovh-exporter/pkg/credentials"
	"github.com/wiremind/ovh-exporter/pkg/network"
)

func main() {
	var exitValue int64 = 1

	app := &cli.Command{
		Name:    "ovh-exporter",
		Usage:   "Prometheus exporter for the OVH API",
		Version: cmd.Version,
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:        "exit",
				Aliases:     []string{"e"},
				Value:       1,
				Usage:       "value returned on error",
				Destination: &exitValue,
			},
		},
		Commands: []*cli.Command{
			credentials.GenerateCommand,
			network.ServeCommand,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		slog.Error(
			"main() error",
			slog.String("error", err.Error()),
		)
		os.Exit(int(exitValue))
	}
}
