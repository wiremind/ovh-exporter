package credentials

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
)

var GenerateCommand = &cli.Command{
	Name:   "credentials",
	Usage:  "generate credentials link",
	Action: generateLink,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "endpoint",
			Usage: "Specify the OVH create token endpoint",
			Value: "https://eu.api.ovh.com/createToken",
		},
	},
}

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

type APIKeyRight struct {
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
}

var apiKeyRights = []APIKeyRight{
	{Method: "GET", Endpoint: "/cloud/project/*"},
	{Method: "GET", Endpoint: "/cloud/project/*/flavor/*"},
	{Method: "GET", Endpoint: "/cloud/project/*/instance"},
	{Method: "GET", Endpoint: "/cloud/project/*/instance/*"},
	{Method: "GET", Endpoint: "/cloud/project/*/region"},
	{Method: "GET", Endpoint: "/cloud/project/*/region/*/floatingip"},
	{Method: "GET", Endpoint: "/cloud/project/*/region/*/loadbalancing/loadbalancer"},
	{Method: "GET", Endpoint: "/cloud/project/*/volume"},
	{Method: "GET", Endpoint: "/dedicated/server"},
	{Method: "GET", Endpoint: "/dedicated/server/*"},
	{Method: "GET", Endpoint: "/dedicated/server/*/serviceInfos"},
	{Method: "GET", Endpoint: "/services"},
	{Method: "GET", Endpoint: "/services/*/savingsPlans/subscribed"},
}

func generateURL(baseURL string, apiKeyRights []APIKeyRight) string {
	var generatedURL strings.Builder

	generatedURL.WriteString(baseURL)

	for index, right := range apiKeyRights {
		if index == 0 {
			generatedURL.WriteString("/?")
		} else {
			generatedURL.WriteString("&")
		}
		_, _ = fmt.Fprintf(&generatedURL, "%s=%s", right.Method, right.Endpoint)
	}

	return generatedURL.String()
}

func generateLink(ctx context.Context, cmd *cli.Command) error {
	baseURL := cmd.String("endpoint")
	link := generateURL(baseURL, apiKeyRights)

	logger.Info().Msgf("%s", link)
	return nil
}
