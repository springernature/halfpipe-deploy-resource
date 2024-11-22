package config

import (
	"errors"
	"fmt"
	"strings"
)

type Request struct {
	Source   Source
	Params   Params
	Metadata Metadata
}

type Metadata struct {
	GitRef    string
	GitRepo   string
	Version   string
	DockerTag string
	AppName   string
	IsActions bool
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
	GitUri           string
	BuildVersionPath string
	Timeout          string
	PreStartCommand  string
	DockerUsername   string
	DockerPassword   string
	DockerTag        string
	CliVersion       string
	Instances        int
	Team             string
	SSOHost          string
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

func (r Request) Verify(isActions bool) error {
	if err := r.Source.Verify(); err != nil {
		return err
	}

	if err := r.Params.Verify(isActions); err != nil {
		return err
	}

	return nil
}

func (source Source) Verify() error {
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

func (params Params) Verify(isActions bool) error {
	if params.Command == "" {
		return ParamsMissingError("command")
	}

	if params.ManifestPath == "" {
		return ParamsMissingError("manifestPath")
	}

	if params.CliVersion != "cf6" && params.CliVersion != "cf7" && params.CliVersion != "cf8" {
		return ParamsInvalidError("cliVersion", "must be either 'cf6', 'cf7' or 'cf8'")
	}

	switch params.Command {
	case PUSH:
		if params.TestDomain == "" {
			return ParamsMissingError("testDomain")
		}

		if params.AppPath == "" {
			if params.DockerPassword == "" && params.DockerUsername == "" {
				return ParamsMissingError("appPath")
			}
		}

		if params.GitRefPath == "" && !isActions {
			return ParamsMissingError("gitRefPath")
		}

		if len(params.PreStartCommand) > 0 && !strings.HasPrefix(params.PreStartCommand, "cf ") {
			return PreStartCommandError(params.PreStartCommand)
		}
	case PROMOTE:
		if params.TestDomain == "" {
			return ParamsMissingError("testDomain")
		}
	case SSO:
		if params.SSOHost == "" {
			return ParamsMissingError("ssoHost")
		}
	}

	return nil
}
