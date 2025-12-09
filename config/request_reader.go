package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/spf13/afero"
)

type RequestReader struct {
	osArgs              []string
	environ             map[string]string
	stdin               io.Reader
	fs                  afero.Afero
	manifestReaderWrite manifest.ReaderWriter
}

func NewRequestReader(osArgs []string, environ map[string]string, stdin io.Reader, fs afero.Afero, manifestReaderWrite manifest.ReaderWriter) RequestReader {
	return RequestReader{
		osArgs:              osArgs,
		environ:             environ,
		stdin:               stdin,
		fs:                  fs,
		manifestReaderWrite: manifestReaderWrite,
	}
}

func (r RequestReader) isActions() bool {
	return r.environ["GITHUB_WORKSPACE"] != ""
}

func (r RequestReader) actionRequest() (request Request, err error) {
	request.Source = Source{
		API:      r.environ["INPUT_API"],
		Org:      r.environ["INPUT_ORG"],
		Space:    r.environ["INPUT_SPACE"],
		Username: r.environ["INPUT_USERNAME"],
		Password: r.environ["INPUT_PASSWORD"],
	}

	dockerPassword, err := base64.StdEncoding.DecodeString(r.environ["INPUT_DOCKERPASSWORD"])
	if err != nil {
		return
	}

	cliVersion := ""
	if cv, found := r.environ["INPUT_CLI_VERSION"]; found {
		cliVersion = cv
	}
	if cv, found := r.environ["INPUT_CLIVERSION"]; found {
		cliVersion = cv
	}

	request.Params = Params{
		Command:        r.environ["INPUT_COMMAND"],
		AppPath:        r.environ["INPUT_APPPATH"],
		ManifestPath:   r.environ["INPUT_MANIFESTPATH"],
		TestDomain:     r.environ["INPUT_TESTDOMAIN"],
		DockerUsername: r.environ["INPUT_DOCKERUSERNAME"],
		DockerPassword: string(dockerPassword),
		DockerTag:      r.environ["INPUT_DOCKERTAG"],
		CliVersion:     cliVersion,
		SSOHost:        r.environ["INPUT_SSOHOST"],
	}

	request.Metadata.IsActions = true

	return
}

func (r RequestReader) concourseRequest() (request Request, err error) {
	data, err := io.ReadAll(r.stdin)
	if err != nil {
		return
	}

	if e := r.fs.WriteFile("/tmp/request", data, 0777); e != nil {
		err = e
		return
	}

	err = json.Unmarshal(data, &request)
	return
}

func (r RequestReader) parseRequest() (request Request, err error) {
	if r.isActions() {
		return r.actionRequest()
	}
	return r.concourseRequest()
}

func (r RequestReader) baseDir() string {
	if r.isActions() {
		return r.environ["GITHUB_WORKSPACE"]
	}
	return r.osArgs[1]
}

func (r RequestReader) setFullPathInRequest(request Request) Request {
	updatedRequest := request

	updatedRequest.Params.ManifestPath = path.Join(r.baseDir(), updatedRequest.Params.ManifestPath)

	updatedRequest.Params.AppPath = path.Join(r.baseDir(), updatedRequest.Params.AppPath)

	if !updatedRequest.Metadata.IsActions && updatedRequest.Params.DockerTag != "" {
		updatedRequest.Params.DockerTag = path.Join(r.baseDir(), updatedRequest.Params.DockerTag)
	}

	if request.Params.GitRefPath != "" {
		updatedRequest.Params.GitRefPath = path.Join(r.baseDir(), request.Params.GitRefPath)
	}

	if request.Params.BuildVersionPath != "" {
		updatedRequest.Params.BuildVersionPath = path.Join(r.baseDir(), request.Params.BuildVersionPath)
	}

	return updatedRequest
}

func (r RequestReader) addGitRefAndVersion(request Request) (updated Request, err error) {
	updated = request
	if r.isActions() {
		updated.Metadata.GitRef = r.environ["GIT_REVISION"]
		updated.Metadata.Version = r.environ["BUILD_VERSION"]
		updated.Metadata.DockerTag = request.Params.DockerTag
		return
	}

	readFile := func(path string) (string, error) {
		file, e := r.fs.ReadFile(path)
		if e != nil {
			return "", e
		}
		return strings.TrimSpace(string(file)), nil
	}

	if request.Params.GitRefPath != "" {
		content, e := readFile(request.Params.GitRefPath)
		if e != nil {
			err = e
			return
		}
		updated.Metadata.GitRef = content
	}

	if request.Params.BuildVersionPath != "" {
		content, e := readFile(request.Params.BuildVersionPath)
		if e != nil {
			err = e
			return
		}
		updated.Metadata.Version = content
	}

	if request.Params.DockerTag != "" {
		content, e := readFile(request.Params.DockerTag)
		if e != nil {
			err = e
			return
		}
		updated.Metadata.DockerTag = content
	}

	return
}

func (r RequestReader) addVars(request Request) (updated Request) {
	updated = request
	prefix := "CF_ENV_VAR_"
	if r.isActions() {
		updated.Params.Vars = make(map[string]string)
		for k, v := range r.environ {
			if strings.HasPrefix(k, prefix) {
				newKey := strings.Replace(k, prefix, "", -1)
				updated.Params.Vars[newKey] = v
			}
		}

		if r.environ["EE_PLATFORM_TEAM"] != "" {
			updated.Params.Vars["EE_PLATFORM_TEAM"] = r.environ["EE_PLATFORM_TEAM"]
		}

		return
	}

	return
}

func (r RequestReader) addAppName(request Request) (updated Request, err error) {
	updated = request
	manifest, err := r.manifestReaderWrite.ReadManifest(request.Params.ManifestPath)
	if err != nil {
		return
	}
	updated.Metadata.AppName = manifest.Applications[0].Name

	return
}

func (r RequestReader) ReadRequest() (request Request, err error) {
	request, err = r.parseRequest()
	if err != nil {
		return
	}

	if request.Params.Command == "check" {
		return request, nil
	}

	if request.Params.CliVersion == "" {
		request.Params.CliVersion = "cf7"
	}

	if request.Params.Command == SSO {
		// We require cf8 because we need the `cf route` command
		request.Params.CliVersion = "cf8"
	}

	if e := request.Verify(r.isActions()); e != nil {
		err = e
		return
	}

	request = r.setDeployedBy(request)
	request = r.setFullPathInRequest(request)
	request = r.addVars(request)
	request, err = r.addGitRefAndVersion(request)
	if err != nil {
		return
	}
	request, err = r.addAppName(request)

	return
}

func (r RequestReader) setDeployedBy(request Request) Request {
	updated := request
	if r.isActions() {

		ref := r.environ["GITHUB_WORKFLOW_REF"]
		firstPart := strings.Split(ref, "@")[0]
		actionsPath := strings.Replace(firstPart, "/.github/", "/actions/", 1)

		p := fmt.Sprintf("https://github.com/%s", actionsPath)
		d := fmt.Sprintf("https://github.com/%s/actions/runs/%s", r.environ["GITHUB_REPOSITORY"], r.environ["GITHUB_RUN_ID"])

		if safe, err := url.Parse(d); err == nil && safe != nil {
			updated.Metadata.DeployedBy = safe.String()
		}

		if safe, err := url.Parse(p); err == nil && safe != nil {
			updated.Metadata.Pipeline = safe.String()
		}
	} else {
		p := fmt.Sprintf(
			"%s/teams/%s/pipelines/%s",
			r.environ["ATC_EXTERNAL_URL"],
			r.environ["BUILD_TEAM_NAME"],
			r.environ["BUILD_PIPELINE_NAME"],
		)

		u := fmt.Sprintf(
			"%s/jobs/%s/builds/%s",
			p,
			r.environ["BUILD_JOB_NAME"],
			r.environ["BUILD_NAME"],
		)

		if safe, err := url.Parse(u); err == nil && safe != nil {
			updated.Metadata.DeployedBy = safe.String()
		}

		if safe, err := url.Parse(p); err == nil && safe != nil {
			updated.Metadata.Pipeline = safe.String()
		}

	}
	return updated
}
