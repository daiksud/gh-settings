package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"go.yaml.in/yaml/v4"
)

func TestSettingsReferenceYAMLExamples(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "docs", "settings", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	fences := regexp.MustCompile("(?s)```yaml\\n(.*?)\\n```")
	parsed, validated := 0, 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for i, match := range fences.FindAllSubmatch(data, -1) {
			t.Run(fmt.Sprintf("%s/%d", filepath.Base(file), i+1), func(t *testing.T) {
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
	}
	if parsed == 0 || validated == 0 {
		t.Fatal("no settings examples checked")
	}
	t.Logf("parsed %d YAML snippets; validated %d complete/root declarations", parsed, validated)
}
