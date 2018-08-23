package plan

import (
	"path"

	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/spf13/afero"
	"strings"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
)

type ResourcePlan interface {
	Plan(request Request, concourseRoot string) (plan Plan, err error)
}

type planner struct {
	manifestReaderWrite manifest.ReaderWriter
	fs                  afero.Afero
}

func NewPlanner(manifestReaderWrite manifest.ReaderWriter, fs afero.Afero) ResourcePlan {
	return planner{
		manifestReaderWrite: manifestReaderWrite,
		fs:                  fs,
	}
}

func (p planner) Plan(request Request, concourseRoot string) (pl Plan, err error) {
	// Here we assume that the request is complete.
	// It has already been verified in out.go with the help of requests.VerifyRequest.

	fullManifestPath := path.Join(concourseRoot, request.Params.ManifestPath)

	if request.Params.Command == config.PUSH {
		fullGitRefPath := ""
		if request.Params.GitRefPath != "" {
			fullGitRefPath = path.Join(concourseRoot, request.Params.GitRefPath)
		}

		if err = p.updateManifestWithVars(fullManifestPath, fullGitRefPath, request.Params.Vars); err != nil {
			return
		}
	}

	pl = append(pl, NewCfCommand("login",
			"-a", request.Source.API,
			"-u", request.Source.Username,
			"-p", request.Source.Password,
			"-o", request.Source.Org,
			"-s", request.Source.Space))

	var halfpipeCommand Command
	switch request.Params.Command {
	case config.PUSH:
		halfpipeCommand = NewCfCommand(request.Params.Command,
			"-manifestPath", fullManifestPath,
			"-appPath", path.Join(concourseRoot, request.Params.AppPath),
			"-testDomain", request.Params.TestDomain,
		)
	case config.PROMOTE:
		halfpipeCommand = NewCfCommand(request.Params.Command,
			"-manifestPath", fullManifestPath,
			"-testDomain", request.Params.TestDomain,
		)
	case config.CLEANUP, config.DELETE:
		halfpipeCommand = NewCfCommand(request.Params.Command,
			"-manifestPath", fullManifestPath,
		)
	}

	if request.Params.Timeout != "" {
		halfpipeCommand = halfpipeCommand.AddToArgs("-timeout", request.Params.Timeout)
	}

	pl = append(pl, halfpipeCommand)

	return
}

func (p planner) updateManifestWithVars(manifestPath string, gitRefPath string, vars map[string]string) (err error) {
	if len(vars) > 0 || gitRefPath != "" {
		apps, e := p.manifestReaderWrite.ReadManifest(manifestPath)
		if e != nil {
			err = e
			return
		}

		// We just assume the first app in the manifest is the app under deployment.
		// We lint that this is the case in the halfpipe linter.
		app := apps.Applications[0]
		if len(app.EnvironmentVariables) == 0 {
			app.EnvironmentVariables = map[string]string{}
		}

		for key, value := range vars {
			app.EnvironmentVariables[key] = value
		}

		if gitRefPath != "" {
			ref, errRead := p.readGitRef(gitRefPath)
			if errRead != nil {
				err = errRead
				return
			}
			app.EnvironmentVariables["GIT_REVISION"] = ref
		}

		if err = p.manifestReaderWrite.WriteManifest(manifestPath, app); err != nil {
			return
		}
	}
	return
}

func (p planner) readGitRef(gitRefPath string) (ref string, err error) {
	bytes, err := p.fs.ReadFile(gitRefPath)
	if err != nil {
		return
	}
	ref = strings.TrimSpace(string(bytes))
	return
}