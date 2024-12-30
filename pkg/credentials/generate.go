package credentials

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

var GenerateCommand = cli.Command{
	Name:   "credentials",
	Usage:  "generate credentials link",
	Action: generateLink,
	Flags: []cli.Flag{
		// Add the OVH_CREATE_TOKEN_ENDPOINT flag here
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
	{Method: "GET", Endpoint: "/cloud/project/*/flavor/*"},
	{Method: "GET", Endpoint: "/cloud/project/*/instance"},
	{Method: "GET", Endpoint: "/cloud/project/*/instance/*"},
	{Method: "GET", Endpoint: "/dedicated/server"},
	{Method: "GET", Endpoint: "/dedicated/server/*"},
	{Method: "GET", Endpoint: "/dedicated/server/*/serviceInfos"},
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
		generatedURL.WriteString(fmt.Sprintf("%s=%s", right.Method, right.Endpoint))
	}

	return generatedURL.String()
}

func generateLink(clicontext *cli.Context) error {
	baseURL := clicontext.String("endpoint")
	link := generateURL(baseURL, apiKeyRights)

	logger.Info().Msgf("%s", link)
	return nil
}
