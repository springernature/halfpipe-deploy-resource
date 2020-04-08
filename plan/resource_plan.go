package plan

import (
	"fmt"
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
		fullBuildVersionPath := ""
		if request.Params.BuildVersionPath != "" {
			fullBuildVersionPath = path.Join(concourseRoot, request.Params.BuildVersionPath)
		}

		if err = p.updateManifestWithVars(fullManifestPath, fullGitRefPath, request.Params.Vars, fullBuildVersionPath); err != nil {
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
		candidateAppName, e := p.getCandidateName(fullManifestPath)
		if e != nil {
			err = e
			return
		}

		pushCommand := NewCfCommand(
			request.Params.Command,
			"-manifestPath", fullManifestPath,
			"-testDomain", request.Params.TestDomain,
		)

		isDockerPush := request.Params.DockerPassword != ""
		if isDockerPush {
			fullDockerTagPath := ""
			if request.Params.DockerTag != "" {
				fullDockerTagPath = path.Join(concourseRoot, request.Params.DockerTag)
			}

			dockerImage, e := p.getDockerImage(fullManifestPath, fullDockerTagPath)
			if e != nil {
				err = e
				return
			}

			pushCommand = pushCommand.
				AddToArgs("-dockerImage", dockerImage).
				AddToArgs("-dockerUsername", request.Params.DockerUsername).
				AddToEnv(fmt.Sprintf("CF_DOCKER_PASSWORD=%s", request.Params.DockerPassword))

		} else {
			pushCommand = pushCommand.AddToArgs("-appPath", path.Join(concourseRoot, request.Params.AppPath))
		}

		if request.Params.PreStartCommand != "" {
			quotedCommand := fmt.Sprintf(`"%s"`, strings.ReplaceAll(request.Params.PreStartCommand, `"`, `\"`))
			pushCommand = pushCommand.AddToArgs("-preStartCommand", quotedCommand)
		}

		if request.Params.Instances != "" {
			pushCommand = pushCommand.AddToArgs("-instances", request.Params.Instances)
		}

		halfpipeCommand = NewCompoundCommand(
			pushCommand,
			NewCfCommand("logs",
				candidateAppName,
				"--recent",
			),
			func(log []byte) bool {
				return strings.Contains(string(log), `TIP: use 'cf logs`)
			})

	case config.PROMOTE:
		halfpipeCommand = NewCfCommand(request.Params.Command,
			"-manifestPath", fullManifestPath,
			"-testDomain", request.Params.TestDomain,
		)
	case config.CHECK, config.CLEANUP, config.DELETE:
		halfpipeCommand = NewCfCommand(request.Params.Command,
			"-manifestPath", fullManifestPath,
		)
	case config.DEPLOY_ROLLING:

		pushCommand := NewCfCommand(
			"push",
			"--manifest", fullManifestPath,
			"--strategy", "rolling",
		)

		isDockerPush := request.Params.DockerPassword != ""
		if isDockerPush {
			fullDockerTagPath := ""
			if request.Params.DockerTag != "" {
				fullDockerTagPath = path.Join(concourseRoot, request.Params.DockerTag)
			}

			dockerImage, e := p.getDockerImage(fullManifestPath, fullDockerTagPath)
			if e != nil {
				err = e
				return
			}

			halfpipeCommand = pushCommand.
				AddToArgs("--docker-image", dockerImage).
				AddToArgs("--docker-username", request.Params.DockerUsername).
				AddToEnv(fmt.Sprintf("CF_DOCKER_PASSWORD=%s", request.Params.DockerPassword))

		} else {
			halfpipeCommand = pushCommand.
				AddToArgs("--path", path.Join(concourseRoot, request.Params.AppPath))
		}

	case config.DELETE_TEST:
		candidateAppName, e := p.getCandidateName(fullManifestPath)
		if e != nil {
			err = e
			return
		}

		halfpipeCommand = NewCfCommand(
			"delete",
			"-f", candidateAppName)
	}

	if request.Params.Timeout != "" && request.Params.Command != config.DEPLOY_ROLLING {
		halfpipeCommand = halfpipeCommand.AddToArgs("-timeout", request.Params.Timeout)
	}

	pl = append(pl, halfpipeCommand)

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

func (p planner) updateManifestWithVars(manifestPath string, gitRefPath string, vars map[string]string, buildVersionPath string) (err error) {
	if len(vars) > 0 || gitRefPath != "" {
		apps, e := p.readManifest(manifestPath)
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
			ref, errRead := p.readFile(gitRefPath)
			if errRead != nil {
				err = errRead
				return
			}
			app.EnvironmentVariables["GIT_REVISION"] = ref
		}

		if buildVersionPath != "" {
			version, errRead := p.readFile(buildVersionPath)
			if errRead != nil {
				err = errRead
				return
			}
			app.EnvironmentVariables["BUILD_VERSION"] = version
		}

		if err = p.manifestReaderWrite.WriteManifest(manifestPath, app); err != nil {
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

func (p planner) getDockerImage(manifestPath string, tagPath string) (dockerImage string, err error) {
	apps, err := p.readManifest(manifestPath)
	if err != nil {
		return
	}

	dockerImage = apps.Applications[0].Docker.Image

	if tagPath != "" {
		content, e := p.fs.ReadFile(tagPath)
		if e != nil {
			err = e
			return
		}

		if strings.Contains(dockerImage, ":") {
			dockerImage = strings.Split(dockerImage, ":")[0]
		}

		dockerImage = fmt.Sprintf("%s:%s", dockerImage, strings.Trim(string(content), "\n"))

	}
	return
}
