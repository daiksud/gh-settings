package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestRepositoryRequiresIndependentPRMessage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Settings struct {
			Rulesets []map[string]any `yaml:"rulesets"`
		} `yaml:"settings"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{{
		"name": "default",
		"rules": []any{
			map[string]any{"type": "deletion"},
			map[string]any{"type": "non_fast_forward"},
			map[string]any{"type": "required_linear_history"},
			map[string]any{"type": "pull_request"},
			map[string]any{
				"type": "required_status_checks",
				"parameters": map[string]any{
					"required_status_checks": []any{
						map[string]any{"context": "pass", "integration_id": 15368},
						map[string]any{"context": "gh-settings", "integration_id": 15368},
						map[string]any{"context": "pr-message", "integration_id": 15368},
					},
				},
			},
		},
	}}
	if !reflect.DeepEqual(config.Settings.Rulesets, want) {
		t.Fatalf("retain the complete inventoried rule membership and manage only its three check providers: %#v", config.Settings.Rulesets)
	}
}
