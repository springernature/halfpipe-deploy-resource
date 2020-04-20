package plan

import (
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"path"

	"github.com/spf13/afero"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type ResourcePlan interface {
	Plan(request Request, concourseRoot string) (plan Plan, err error)
}

type planner struct {
	manifestReaderWrite manifest.ReaderWriter
	fs                  afero.Afero
}

func NewPlanner(manifestReaderWrite manifest.ReaderWriter, fs afero.Afero, appsSummary []cfclient.AppSummary) ResourcePlan {
	return planner{
		manifestReaderWrite: manifestReaderWrite,
		fs:                  fs,
	}
}

func (p planner) setFullPathInRequest(request Request, concourseRoot string) Request {
	updatedRequest := request

	updatedRequest.Params.ManifestPath = path.Join(concourseRoot, updatedRequest.Params.ManifestPath)

	if updatedRequest.Params.AppPath != "" {
		updatedRequest.Params.AppPath = path.Join(concourseRoot, updatedRequest.Params.AppPath)
	}

	if updatedRequest.Params.DockerTag != "" {
		updatedRequest.Params.DockerTag = path.Join(concourseRoot, updatedRequest.Params.DockerTag)
	}

	if request.Params.GitRefPath != "" {
		updatedRequest.Params.GitRefPath = path.Join(concourseRoot, request.Params.GitRefPath)
	}

	if request.Params.BuildVersionPath != "" {
		updatedRequest.Params.BuildVersionPath = path.Join(concourseRoot, request.Params.BuildVersionPath)
	}

	return updatedRequest
}

func (p planner) Plan(request Request, concourseRoot string) (pl Plan, err error) {
	// Here we assume that the request is complete.
	// It has already been verified in out.go with the help of requests.VerifyRequest.

	// Here we update the paths to take into account concourse root
	request = p.setFullPathInRequest(request, concourseRoot)

	readManifest, err := p.readManifest(request.Params.ManifestPath)
	if err != nil {
		// todo: test this
		return
	}

	// We lint that there is only one app.
	appUnderDeployment := readManifest.Applications[0]

	if request.Params.Command == config.PUSH {
		if err = p.updateManifestWithVars(request); err != nil {
			return
		}
	}

	pl = append(pl, NewCfCommand("login",
		"-a", request.Source.API,
		"-u", request.Source.Username,
		"-p", request.Source.Password,
		"-o", request.Source.Org,
		"-s", request.Source.Space))

	switch request.Params.Command {
	case config.PUSH:
		var dockerTag string
		if request.Params.DockerTag != "" {
			content, e := p.fs.ReadFile(request.Params.DockerTag)
			if e != nil {
				err = e
				return
			}
			dockerTag = string(content)
		}

		pushCommands, e := NewPushPlan().Plan(appUnderDeployment, request, dockerTag)
		if e != nil {
			// todo: test this
			err = e
			return
		}

		pl = append(pl, pushCommands...)
		//
		//
		//candidateAppName, e := p.getCandidateName(fullManifestPath)
		//if e != nil {
		//	err = e
		//	return
		//}
		//
		//pushCommand := NewCfCommand(
		//	request.Params.Command,
		//	"-manifestPath", fullManifestPath,
		//	"-testDomain", request.Params.TestDomain,
		//)
		//
		//

		//
		//} else {
		//	pushCommand = pushCommand.AddToArgs("-appPath", path.Join(concourseRoot, request.Params.AppPath))
		//}
		//
		//if request.Params.PreStartCommand != "" {
		//	quotedCommand := fmt.Sprintf(`"%s"`, strings.ReplaceAll(request.Params.PreStartCommand, `"`, `\"`))
		//	pushCommand = pushCommand.AddToArgs("-preStartCommand", quotedCommand)
		//}
		//
		//halfpipeCommand = NewCompoundCommand(
		//	pushCommand,
		//	NewCfCommand("logs",
		//		candidateAppName,
		//		"--recent",
		//	),
		//	func(log []byte) bool {
		//		return strings.Contains(string(log), `TIP: use 'cf logs`)
		//	})

		//case config.PROMOTE:
		//	halfpipeCommand = NewCfCommand(request.Params.Command,
		//		"-manifestPath", fullManifestPath,
		//		"-testDomain", request.Params.TestDomain,
		//	)
		//case config.CHECK, config.CLEANUP, config.DELETE:
		//	halfpipeCommand = NewCfCommand(request.Params.Command,
		//		"-manifestPath", fullManifestPath,
		//	)
	}

	//if request.Params.Timeout != "" {
	//	halfpipeCommand = halfpipeCommand.AddToArgs("-timeout", request.Params.Timeout)
	//}
	//
	//pl = append(pl, halfpipeCommand)

	return
}

func (p planner) getCandidateName(manifestPath string) (candidateName string, err error) {
	apps, err := p.readManifest(manifestPath)
	if err != nil {
		return
	}

	// We just assume the first app in the manifest is the app under deployment.
	// We lint that this is the case in the halfpipe linter.
	app := apps.Applications[0]
	candidateName = fmt.Sprintf("%s-CANDIDATE", app.Name)
	return
}

func (p planner) readManifest(manifestPath string) (manifest.Manifest, error) {
	return p.manifestReaderWrite.ReadManifest(manifestPath)
}

func (p planner) updateManifestWithVars(request Request) (err error) {
	if len(request.Params.Vars) > 0 || request.Params.GitRefPath != "" {
		apps, e := p.readManifest(request.Params.ManifestPath)
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

		for key, value := range request.Params.Vars {
			app.EnvironmentVariables[key] = value
		}

		if request.Params.GitRefPath != "" {
			ref, errRead := p.readFile(request.Params.GitRefPath)
			if errRead != nil {
				err = errRead
				return
			}
			app.EnvironmentVariables["GIT_REVISION"] = ref
		}

		if request.Params.BuildVersionPath != "" {
			version, errRead := p.readFile(request.Params.BuildVersionPath)
			if errRead != nil {
				err = errRead
				return
			}
			app.EnvironmentVariables["BUILD_VERSION"] = version
		}

		if err = p.manifestReaderWrite.WriteManifest(request.Params.ManifestPath, app); err != nil {
			return
		}
	}
	return
}

func (p planner) readFile(gitRefPath string) (ref string, err error) {
	bytes, err := p.fs.ReadFile(gitRefPath)
	if err != nil {
		return
	}
	ref = strings.TrimSpace(string(bytes))
	return
}
