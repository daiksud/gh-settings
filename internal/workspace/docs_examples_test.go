package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"go.yaml.in/yaml/v4"
)

func TestSettingsReferenceYAMLExamples(t *testing.T) {
	reference := os.DirFS(filepath.Join("..", "..", "docs", "references", "settings"))
	fences := regexp.MustCompile("(?s)```yaml\\n(.*?)\\n```")
	parsed, validated := 0, 0
	err := fs.WalkDir(reference, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := fs.ReadFile(reference, path)
		if err != nil {
			return err
		}
		for i, match := range fences.FindAllSubmatch(data, -1) {
			t.Run(fmt.Sprintf("%s/%d", path, i+1), func(t *testing.T) {
				var document yaml.Node
				if err := yaml.Unmarshal(match[1], &document); err != nil {
					t.Fatal(err)
				}
				parsed++
				text := string(match[1])
				// Complete declarations and repository-root fragments are executable
				// examples. Other snippets illustrate fields within their stated context.
				if strings.HasPrefix(text, "repository:") {
					text = "version: daiksud/gh-settings@v0\n" + text
				}
				if !strings.HasPrefix(text, "version:") {
					return
				}
				model, err := DecodeConfig([]byte(text))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := github.DeclarationFromUI(model); err != nil {
					t.Fatal(err)
				}
				validated++
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if parsed == 0 || validated == 0 {
		t.Fatal("no settings examples checked")
	}
	t.Logf("parsed %d YAML snippets; validated %d complete/root declarations", parsed, validated)
}
