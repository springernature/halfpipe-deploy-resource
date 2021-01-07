package config

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/afero"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

type RequestReader struct {
	osArgs  []string
	environ map[string]string
	stdin   io.Reader
	fs      afero.Afero
}

func NewRequestReader(osArgs []string, environ map[string]string, stdin io.Reader, fs afero.Afero) RequestReader {
	return RequestReader{
		osArgs:  osArgs,
		environ: environ,
		stdin:   stdin,
		fs:      fs,
	}
}

func (r RequestReader) isActions() bool {
	return r.environ["GITHUB_WORKSPACE"] != ""
}

func (r RequestReader) actionRequest() (request Request) {
	request.Source = Source{
		API:      r.environ["INPUT_API"],
		Org:      r.environ["INPUT_ORG"],
		Space:    r.environ["INPUT_SPACE"],
		Username: r.environ["INPUT_USERNAME"],
		Password: r.environ["INPUT_PASSWORD"],
	}

	request.Params = Params{
		Command:      r.environ["INPUT_COMMAND"],
		AppPath:      r.environ["INPUT_APPPATH"],
		ManifestPath: r.environ["INPUT_MANIFESTPATH"],
		TestDomain:   r.environ["INPUT_TESTDOMAIN"],
	}

	return
}

func (r RequestReader) concourseRequest() (request Request, err error) {
	data, err := ioutil.ReadAll(r.stdin)
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
		fmt.Println("Is action")
		request = r.actionRequest()
		return
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

	if updatedRequest.Params.AppPath != "" {
		updatedRequest.Params.AppPath = path.Join(r.baseDir(), updatedRequest.Params.AppPath)
	}

	if updatedRequest.Params.DockerTag != "" {
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
		updated.Metadata = Metadata{
			GitRef:  r.environ["GITHUB_SHA"],
			Version: r.environ["GITHUB_RUN_NUMBER"],
		}
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

func (r RequestReader) ReadRequest() (request Request, err error) {
	fmt.Println(10)
	request, err = r.parseRequest()
	if err != nil {
		return
	}
	fmt.Println(20)
	if request.Params.CliVersion == "" {
		request.Params.CliVersion = "cf6"
	}

	if e := request.Verify(r.isActions()); e != nil {
		err = e
		return
	}

	request = r.setFullPathInRequest(request)
	request, err = r.addGitRefAndVersion(request)

	return
}
