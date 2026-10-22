package githubactions

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestManualReleaseUsesOneNonCancellingPublicationLane(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var release struct {
		On          map[string]yaml.Node `yaml:"on"`
		Concurrency struct {
			Group  string `yaml:"group"`
			Cancel bool   `yaml:"cancel-in-progress"`
		} `yaml:"concurrency"`
	}
	if err := yaml.Unmarshal(data, &release); err != nil {
		t.Fatal(err)
	}
	if len(release.On) != 1 || release.On["workflow_dispatch"].Kind == 0 {
		t.Fatal("release publication must remain manually dispatched")
	}
	if release.Concurrency.Group != "release-${{ github.repository }}" || release.Concurrency.Cancel {
		t.Fatal("all versions must share one publication lane without cancelling a running release")
	}
}
