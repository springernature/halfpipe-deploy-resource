package manifest

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/spf13/afero"
	"strings"
)

type ReaderWriter interface {
	ReadManifest(path string) (manifestparser.Manifest, error)
	WriteManifest(path string, manifest manifestparser.Manifest) error
}

type manifestReadWrite struct {
	fs afero.Afero
}

func NewManifestReadWrite(fs afero.Afero) ReaderWriter {
	return manifestReadWrite{
		fs: fs,
	}
}

func (m manifestReadWrite) ReadManifest(path string) (manifestparser.Manifest, error) {
	return manifestparser.ManifestParser{}.InterpolateAndParse(path, nil, nil)
}

func (m manifestReadWrite) WriteManifest(path string, manifest manifestparser.Manifest) error {
	serialized, err := manifestparser.ManifestParser{}.MarshalManifest(manifest)
	if err != nil {
		return err
	}

	if manifest.GetFirstApp().DiskQuota != "" {
		withCorrectDiskQuota := strings.Replace(string(serialized), "disk-quota:", "disk_quota:", 1)
		return m.fs.WriteFile(path, []byte(withCorrectDiskQuota), 0666)
	}

	return m.fs.WriteFile(path, serialized, 0666)
}
