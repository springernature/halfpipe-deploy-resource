package manifest

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type ReaderWriter interface {
	ReadManifest(path string) (Manifest, error)
	ReadManifestNew(path string) (manifestparser.Manifest, error)
	WriteManifest(path string, application Application) error
	WriteManifestNew(path string, manifest manifestparser.Manifest) error
}

type manifestReadWrite struct {
	fs afero.Afero
}

func NewManifestReadWrite(fs afero.Afero) ReaderWriter {
	return manifestReadWrite{
		fs: fs,
	}
}

func (m manifestReadWrite) ReadManifest(path string) (man Manifest, err error) {
	manifestBytes, err := m.fs.ReadFile(path)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(manifestBytes, &man)
	return
}

func (m manifestReadWrite) ReadManifestNew(path string) (manifestparser.Manifest, error) {
	return manifestparser.ManifestParser{}.InterpolateAndParse(path, nil, nil)
}

func (m manifestReadWrite) WriteManifest(path string, application Application) (err error) {
	manifest := Manifest{
		Applications: []Application{
			application,
		},
	}

	out, err := yaml.Marshal(manifest)
	if err != nil {
		return
	}

	return m.fs.WriteFile(path, out, 0666)
}

func (m manifestReadWrite) WriteManifestNew(path string, manifest manifestparser.Manifest) error {
	serialized, err := manifestparser.ManifestParser{}.MarshalManifest(manifest)
	if err != nil {
		return err
	}

	return m.fs.WriteFile(path, serialized, 0666)
}
