package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestLoadYAMLRootPreservesSource(t *testing.T) {
	root, err := loadYAMLRoot([]byte(repositoryYAML("description: &text Example\nhomepage: *text\n")))
	if err != nil {
		t.Fatal(err)
	}
	if root.Kind != yaml.MappingNode || root.Tag != "!!map" || root.Line != 1 {
		t.Fatalf("root kind=%d tag=%q line=%d", root.Kind, root.Tag, root.Line)
	}
	repository := root.Content[3].Content[1]
	description, homepage := repository.Content[1], repository.Content[3]
	if description.Line != 4 || homepage.Line != 5 || homepage.Kind != yaml.AliasNode || homepage.Alias != description {
		t.Fatalf("description=%+v homepage=%+v; source lines and alias must be preserved", description, homepage)
	}
}

func TestLoadYAMLRootErrors(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
	}{
		{"empty", "", "decode YAML:"},
		{"malformed", "schemaVersion: [\n", "decode YAML:"},
		{"multiple documents", "schemaVersion: 1\n---\nschemaVersion: 1\n", "expected one YAML document"},
		{"scalar root", "true\n", "expected a root mapping"},
		{"sequence root", "- true\n", "expected a root mapping"},
		{"contradictory root tag", "!!str {schemaVersion: 1}\n", "expected a root mapping"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, err := loadYAMLRoot([]byte(test.input))
			if root != nil || err == nil || !strings.HasPrefix(err.Error(), test.want) {
				t.Fatalf("root=%+v error=%v, want prefix %q", root, err, test.want)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name, input, errorText string
	}{
		{"all fields", repositoryYAML("description: Example\nhomepage: https://example.com\nhas_issues: true\nhas_projects: false\nhas_wiki: true\n"), ""},
		{"omitted fields", repositoryYAML("has_issues: false\n"), ""},
		{"empty strings", repositoryYAML("description: ''\nhomepage: \"\"\n"), ""},
		{"quoted null is a string", repositoryYAML("description: 'null'\n"), ""},
		{"version only", "schemaVersion: 1\n", ""},
		{"empty settings", "schemaVersion: 1\nsettings: {}\n", ""},
		{"empty repository", "schemaVersion: 1\nsettings:\n  repository: {}\n", ""},
		{"missing schema version", "settings: {}\n", "missing schemaVersion"},
		{"empty mapping has no version", "{}\n", "missing schemaVersion"},
		{"unsupported schema version", "schemaVersion: 2\n", "unsupported schemaVersion"},
		{"string schema version", "schemaVersion: '1'\n", "schemaVersion must be integer 1"},
		{"null schema version", "schemaVersion: null\n", "schemaVersion must be integer 1"},
		{"float schema version", "schemaVersion: 1.0\n", "schemaVersion must be integer 1"},
		{"duplicate schema version", "schemaVersion: 1\nschemaVersion: 1\n", "duplicate key"},
		{"old flat format", "schemaVersion: 1\nhas_issues: false\n", "unknown root key"},
		{"root metadata not supported", "schemaVersion: 1\nversion: 1\n", "unknown root key"},
		{"future group not supported", "schemaVersion: 1\nsettings:\n  labels: []\n", "unknown settings group"},
		{"null settings", "schemaVersion: 1\nsettings: null\n", "settings must be a mapping"},
		{"null repository", "schemaVersion: 1\nsettings:\n  repository: null\n", "settings.repository must be a mapping"},
		{"tagged root mapping as string", "!!str {schemaVersion: 1}\n", "expected a root mapping"},
		{"tagged settings mapping as string", "schemaVersion: 1\nsettings: !!str {}\n", "mapping must have YAML map tag"},
		{"tagged repository mapping as string", "schemaVersion: 1\nsettings:\n  repository: !!str {}\n", "mapping must have YAML map tag"},
		{"unknown field", repositoryYAML("has_issues: true\nvisibility: private\n"), "unknown setting"},
		{"duplicate field", repositoryYAML("has_issues: true\nhas_issues: false\n"), "duplicate key"},
		{"duplicate quoted field", repositoryYAML("'has_issues': true\n\"has_issues\": false\n"), "duplicate key"},
		{"duplicate alias key", repositoryYAML("&field has_issues: true\n*field: false\n"), "duplicate key"},
		{"alias merge key", repositoryYAML("&merge <<: {has_issues: true}\n*merge: {visibility: private}\n"), "merge key cannot be an alias"},
		{"string instead of boolean", repositoryYAML("has_issues: \"false\"\n"), "must be a boolean"},
		{"legacy boolean yes", repositoryYAML("has_issues: !!bool yes\n"), "spelled true or false"},
		{"legacy boolean on", repositoryYAML("has_issues: !!bool on\n"), "spelled true or false"},
		{"tagged uppercase boolean", repositoryYAML("has_issues: !!bool TRUE\n"), ""},
		{"integer instead of boolean", repositoryYAML("has_issues: 0\n"), "must be a boolean"},
		{"invalid tagged boolean", repositoryYAML("has_issues: !!bool nope\n"), "must be a boolean"},
		{"boolean instead of string", repositoryYAML("description: false\n"), "must be a string"},
		{"binary instead of string", repositoryYAML("description: !!binary /w==\n"), "must be a string"},
		{"null instead of string", repositoryYAML("description: null\n"), "must be a string"},
		{"empty scalar is null", repositoryYAML("homepage:\n"), "must be a string"},
		{"null instead of boolean", repositoryYAML("has_wiki: null\n"), "must be a boolean"},
		{"integer instead of string", repositoryYAML("homepage: 42\n"), "must be a string"},
		{"nested mapping", repositoryYAML("description:\n  text: Example\n"), "must be a string"},
		{"alias value", repositoryYAML("description: &text Example\nhomepage: *text\n"), ""},
		{"alias wrong type", repositoryYAML("description: &value false\nhomepage: *value\n"), "must be a string"},
		{"nonstring key", repositoryYAML("42: Example\n"), "mapping keys must be strings"},
		{"null key", "schemaVersion: 1\nnull: ignored\n", "mapping keys must be strings"},
		{"bare repository wrapper", "schemaVersion: 1\nrepository:\n  has_issues: true\n", "unknown root key"},
		{"sequence root", "- has_issues: true\n", "expected a root mapping"},
		{"scalar root", "true\n", "expected a root mapping"},
		{"empty document", "", "decode YAML"},
		{"malformed YAML", "has_issues: [\n", "decode YAML"},
		{"two documents", "has_issues: true\n---\nhas_wiki: false\n", "expected one YAML document"},
		{"merge key", repositoryYAML("<<: {has_issues: true}\n"), ""},
		{"mistagged merge mapping", repositoryYAML("<<: !!str {has_issues: true}\n"), "mapping must have YAML map tag"},
		{"mistagged merge sequence", repositoryYAML("<<: !!str [{has_issues: true}]\n"), "sequence must have YAML sequence tag"},
		{"duplicate merge key", repositoryYAML("<<: {has_issues: true}\n<<: {has_wiki: false}\n"), "already defined"},
		{"explicit value overrides merge", repositoryYAML("<<: {has_issues: true}\nhas_issues: false\n"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSettings([]byte(tt.input))
			if tt.errorText == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.errorText) {
				t.Fatalf("error=%v, want %q", err, tt.errorText)
			}
		})
	}
}

func repositoryYAML(fields string) string {
	return "schemaVersion: 1\nsettings:\n  repository:\n    " + strings.ReplaceAll(strings.TrimSuffix(fields, "\n"), "\n", "\n    ") + "\n"
}

func TestValidateCommandReadsOnlySettingsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, settingsPath)
	input := []byte(repositoryYAML("description: ''\nhas_issues: false\n"))
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(dir, "nested")) // No origin or GitHub authentication is available.
	var stdout, stderr bytes.Buffer
	code := run([]string{"validate"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		t.Fatal("validate must not call export or GitHub")
		return nil, nil
	}, validate: validateCurrent})
	if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(unchanged, input) {
		t.Fatalf("settings changed: %q, error=%v", unchanged, err)
	}
}

func TestValidateCommandMissingFile(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate"}, &stdout, &stderr, commandOperations{validate: validateCurrent}); code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), settingsPath) {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestValidateCommandPreservesTrailingSpaceInGitRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo ")
	if err := os.MkdirAll(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(dir, settingsPath), []byte(repositoryYAML("has_issues: false\n")), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate"}, &stdout, &stderr, commandOperations{validate: validateCurrent}); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
