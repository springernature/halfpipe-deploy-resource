package plan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/springernature/halfpipe-deploy-resource/config"
)

type Request struct {
	Source Source
	Params Params
}

type Source struct {
	API                  string
	Org                  string
	Space                string
	Username             string
	Password             string
	PrometheusGatewayURL string
}

type Params struct {
	Command          string
	ManifestPath     string
	AppPath          string
	TestDomain       string
	Vars             map[string]string
	GitRefPath       string
	BuildVersionPath string
	Timeout          string
	PreStartCommand  string
	DockerUsername   string
	DockerPassword   string
	DockerTag        string
	CliVersion       string
}

func SourceMissingError(field string) error {
	return errors.New(fmt.Sprintf("Source config must contain %s", field))
}

func ParamsMissingError(field string) error {
	return errors.New(fmt.Sprintf("Params config must contain %s", field))
}

func ParamsInvalidError(field string, reason string) error {
	return errors.New(fmt.Sprintf("Params '%s': %s", field, reason))
}

func PreStartCommandError(preStartCommand string) error {
	return errors.New(fmt.Sprintf("invalid preStartCommand - only cf commands are allowed: '%s'", preStartCommand))
}

func VerifyRequest(request Request) error {
	if err := VerifyRequestSource(request.Source); err != nil {
		return err
	}

	if err := VerifyRequestParams(request.Params); err != nil {
		return err
	}

	return nil
}

func VerifyRequestSource(source Source) error {
	if source.API == "" {
		return SourceMissingError("api")
	}

	if source.Space == "" {
		return SourceMissingError("space")
	}

	if source.Org == "" {
		return SourceMissingError("org")
	}

	if source.Password == "" {
		return SourceMissingError("password")
	}

	if source.Username == "" {
		return SourceMissingError("username")
	}

	return nil
}

func VerifyRequestParams(params Params) error {
	if params.Command == "" {
		return ParamsMissingError("command")
	}

	if params.ManifestPath == "" {
		return ParamsMissingError("manifestPath")
	}

	if params.CliVersion != "cf6" && params.CliVersion != "cf7" {
		return ParamsInvalidError("cliVersion", "must be either 'cf6' or 'cf7'")
	}

	switch params.Command {
	case config.PUSH:
		if params.TestDomain == "" {
			return ParamsMissingError("testDomain")
		}

		if params.AppPath == "" {
			if params.DockerPassword == "" && params.DockerUsername == "" {
				return ParamsMissingError("appPath")
			}
		}

		if params.GitRefPath == "" {
			return ParamsMissingError("gitRefPath")
		}

		if len(params.PreStartCommand) > 0 && !strings.HasPrefix(params.PreStartCommand, "cf ") {
			return PreStartCommandError(params.PreStartCommand)
		}
	case config.PROMOTE:
		if params.TestDomain == "" {
			return ParamsMissingError("testDomain")
		}
	}

	return nil
}
