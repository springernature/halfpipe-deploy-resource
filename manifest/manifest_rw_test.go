package manifest

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestReadManifest(t *testing.T) {
	manifestReadWriter := NewManifestReadWrite(afero.Afero{Fs: afero.NewMemMapFs()})

	t.Run("It has disk-quote: 1g", func(t *testing.T) {
		file, _ := os.CreateTemp("", "manifest")
		defer os.Remove(file.Name())

		manifest := []byte(`
applications:
- name: myApp
  disk-quote: 1337G
`)
		file.Write(manifest)
		man, err := manifestReadWriter.ReadManifest(file.Name())
		assert.NoError(t, err)
		assert.Equal(t, man.GetFirstApp().Name, "myApp")
	})

	t.Run("It has disk-quote: 1337G", func(t *testing.T) {
		file, _ := os.CreateTemp("", "manifest")
		defer os.Remove(file.Name())

		manifest := []byte(`
applications:
- name: myApp
  disk_quote: 1g
`)
		file.Write(manifest)
		man, err := manifestReadWriter.ReadManifest(file.Name())
		assert.NoError(t, err)
		assert.Equal(t, man.GetFirstApp().Name, "myApp")
	})
}

func TestWriteManifest(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	manifestReadWriter := NewManifestReadWrite(fs)

	t.Run("It writes out manifest in the correct way", func(t *testing.T) {
		manifest := manifestparser.Manifest{
			Applications: []manifestparser.Application{
				{
					Name:      "myApp",
					DiskQuota: "1337G",
					RemainingManifestFields: map[string]interface{}{
						"routes": []map[string]string{
							{"route": "myRoute.com"},
						},
						"env": map[string]string{
							"VAR1":       "ONE",
							"VAR2":       "TWO",
							"disk-quota": "shouldNotChange",
						},
					},
				},
			},
		}

		expectedManifest := `applications:
- name: myApp
  disk_quota: 1337G
  env:
    VAR1: ONE
    VAR2: TWO
    disk-quota: shouldNotChange
  routes:
  - route: myRoute.com
`
		path := "/path/to/manifest.yml"
		err := manifestReadWriter.WriteManifest(path, manifest)
		assert.NoError(t, err)

		fileBytes, _ := fs.ReadFile(path)
		assert.Equal(t, expectedManifest, string(fileBytes))
	})

	t.Run("It writes out manifest in the correct even though there is no disk_quota", func(t *testing.T) {
		manifest := manifestparser.Manifest{
			Applications: []manifestparser.Application{
				{
					Name: "myApp",
					RemainingManifestFields: map[string]interface{}{
						"routes": []map[string]string{
							{"route": "myRoute.com"},
						},
						"env": map[string]string{
							"VAR1":       "ONE",
							"VAR2":       "TWO",
							"disk-quota": "shouldNotChange",
						},
					},
				},
			},
		}

		expectedManifest := `applications:
- name: myApp
  env:
    VAR1: ONE
    VAR2: TWO
    disk-quota: shouldNotChange
  routes:
  - route: myRoute.com
`
		path := "/path/to/manifest.yml"
		err := manifestReadWriter.WriteManifest(path, manifest)
		assert.NoError(t, err)

		fileBytes, _ := fs.ReadFile(path)
		assert.Equal(t, expectedManifest, string(fileBytes))
	})
}
