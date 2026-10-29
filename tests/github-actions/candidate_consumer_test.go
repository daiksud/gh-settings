package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestHostedConsumerFixturesUseCurrentConfiguration(t *testing.T) {
	for _, test := range []struct{ action, checkout string }{
		{"candidate-smoke", ".action"},
		{"release-verify", ".published"},
	} {
		t.Run(test.action, func(t *testing.T) {
			action, _ := readReleaseAction(t, test.action)
			var prepare string
			for _, step := range action.Runs.Steps {
				if step.Name == "Create a configuration-only consumer checkout" {
					if prepare != "" {
						t.Fatal("duplicate consumer preparation")
					}
					prepare = step.Run
				}
			}
			if prepare == "" {
				t.Fatal("missing consumer preparation")
			}
			consumer := t.TempDir()
			template := t.TempDir()
			var environment []string
			for _, value := range os.Environ() {
				key := strings.SplitN(value, "=", 2)[0]
				if key != "GH_TOKEN" && key != "GITHUB_TOKEN" && !strings.HasPrefix(key, "GIT_") {
					environment = append(environment, value)
				}
			}
			environment = append(environment, "GH_TOKEN=", "GITHUB_TOKEN=", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TEMPLATE_DIR="+template)
			checkout := exec.Command("git", "init", "--quiet", filepath.Join(consumer, test.checkout))
			checkout.Env = environment
			if output, err := checkout.CombinedOutput(); err != nil {
				t.Fatalf("prepare independent Action checkout: %v\n%s", err, output)
			}
			command := exec.Command("bash", "-e", "-o", "pipefail", "-c", prepare)
			command.Dir, command.Env = consumer, environment
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("execute actual consumer preparation: %v\n%s", err, output)
			}
			data, err := os.ReadFile(filepath.Join(consumer, ".github/settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			ui, err := workspace.DecodeConfig(data)
			if err != nil {
				t.Fatalf("hosted fixture is not current configuration: %v", err)
			}
			declaration, err := github.DeclarationFromUI(ui)
			if err != nil {
				t.Fatal(err)
			}
			if description := declaration.Repository.Description; description == nil || *description != "Hosted composite fixture" {
				t.Fatalf("consumer fixture lost its declared description: %+v", declaration.Repository)
			}
		})
	}
}
