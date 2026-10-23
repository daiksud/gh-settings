package workspace

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func TestConfigLeafPresence(t *testing.T) {
	empty := ""
	booleanText, nullText, multiline := "true", "null", "first line\nsecond line\n"
	disabled := false
	for _, test := range []struct {
		name       string
		repository settings.Declaration
		want       string
	}{
		{
			name: "all absent",
			want: "schemaVersion: 1\nsettings:\n    repository: {}\n",
		},
		{
			name:       "some present zero",
			repository: settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, HasIssues: &disabled}},
			want:       "schemaVersion: 1\nsettings:\n    repository:\n        description: \"\"\n        has_issues: false\n",
		},
		{
			name:       "all present zero",
			repository: settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty, HasIssues: &disabled, HasProjects: &disabled, HasWiki: &disabled}},
			want:       "schemaVersion: 1\nsettings:\n    repository:\n        description: \"\"\n        homepage: \"\"\n        has_issues: false\n        has_projects: false\n        has_wiki: false\n",
		},
		{
			name:       "strings that resemble other YAML types",
			repository: settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &booleanText, Homepage: &nullText}},
			want:       "schemaVersion: 1\nsettings:\n    repository:\n        description: \"true\"\n        homepage: \"null\"\n",
		},
		{
			name:       "multiline string",
			repository: settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &multiline}},
			want:       "schemaVersion: 1\nsettings:\n    repository:\n        description: |\n            first line\n            second line\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := EncodeConfig(test.repository)
			if err != nil || string(got) != test.want {
				t.Fatalf("output=%q err=%v, want %q", got, err, test.want)
			}
		})
	}
}

func TestConfigRoundTrip(t *testing.T) {
	description, homepage, empty := "Example", "https://example.com", ""
	enabled, disabled := true, false
	tests := []struct {
		name        string
		declaration settings.Declaration
	}{
		{name: "all omitted"},
		{name: "partial zero values", declaration: settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, HasIssues: &disabled}}},
		{name: "all fields", declaration: settings.Declaration{Repository: settings.RepositoryDeclaration{
			Description: &description, Homepage: &homepage,
			HasIssues: &enabled, HasProjects: &disabled, HasWiki: &enabled,
		}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := EncodeConfig(test.declaration)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeConfig(encoded)
			if err != nil || !reflect.DeepEqual(got, test.declaration) {
				t.Fatalf("declaration=%+v error=%v, want %+v", got, err, test.declaration)
			}
		})
	}
}

func TestParseConfigValues(t *testing.T) {
	stringValue := func(value string) *string { return &value }
	boolValue := func(value bool) *bool { return &value }
	for _, test := range []struct {
		name, input string
		want        settings.Declaration
	}{
		{"all fields", repositoryYAML("description: Example\nhomepage: https://example.com\nhas_issues: true\nhas_projects: false\nhas_wiki: true\n"), settings.Declaration{Repository: settings.RepositoryDeclaration{
			Description: stringValue("Example"), Homepage: stringValue("https://example.com"),
			HasIssues: boolValue(true), HasProjects: boolValue(false), HasWiki: boolValue(true),
		}}},
		{"partial zero values", repositoryYAML("description: ''\nhas_issues: false\n"), settings.Declaration{Repository: settings.RepositoryDeclaration{
			Description: stringValue(""), HasIssues: boolValue(false),
		}}},
		{"version only", "schemaVersion: 1\n", settings.Declaration{Repository: settings.RepositoryDeclaration{}}},
		{"empty settings", "schemaVersion: 1\nsettings: {}\n", settings.Declaration{Repository: settings.RepositoryDeclaration{}}},
		{"empty repository", "schemaVersion: 1\nsettings:\n  repository: {}\n", settings.Declaration{Repository: settings.RepositoryDeclaration{}}},
		{"root merge", "<<: {schemaVersion: 1, settings: {repository: {has_wiki: false}}}\n", settings.Declaration{Repository: settings.RepositoryDeclaration{
			HasWiki: boolValue(false),
		}}},
		{"settings merge", "schemaVersion: 1\nsettings:\n  <<: {repository: {has_projects: true}}\n", settings.Declaration{Repository: settings.RepositoryDeclaration{
			HasProjects: boolValue(true),
		}}},
		{"alias and merge", repositoryYAML("description: &text Example\nhomepage: *text\n<<: {has_issues: !!bool TRUE, has_wiki: false}\n"), settings.Declaration{Repository: settings.RepositoryDeclaration{
			Description: stringValue("Example"), Homepage: stringValue("Example"),
			HasIssues: boolValue(true), HasWiki: boolValue(false),
		}}},
		{"explicit value overrides merge", repositoryYAML("<<: {has_issues: true}\nhas_issues: false\n"), settings.Declaration{Repository: settings.RepositoryDeclaration{
			HasIssues: boolValue(false),
		}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			declaration, err := DecodeConfig([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(declaration, test.want) {
				t.Fatalf("declaration=%+v, want %+v", declaration, test.want)
			}
		})
	}
}

func TestParseConfigErrorsReturnNoValues(t *testing.T) {
	for _, input := range []string{
		"settings:\n  repository:\n    has_issues: false\n",
		repositoryYAML("description: Example\nhas_wiki: null\n"),
		repositoryYAML("description: Example\nvisibility: private\n"),
		repositoryYAML("has_issues: &value false\nhomepage: *value\n"),
	} {
		declaration, err := DecodeConfig([]byte(input))
		if err == nil || !reflect.DeepEqual(declaration, settings.Declaration{Repository: settings.RepositoryDeclaration{}}) {
			t.Fatalf("declaration=%+v error=%v, want zero declaration and error", declaration, err)
		}
	}
}

func TestParseConfigValidation(t *testing.T) {
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
		{"future group not supported", "schemaVersion: 1\nsettings:\n  environments: []\n", "unknown settings group"},
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
			_, err := DecodeConfig([]byte(tt.input))
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
