package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIConfigYAMLAliasesMergesAndStringStyles(t *testing.T) {
	for _, test := range []struct {
		name, input string
		want        uimodel.Repository
	}{
		{"root merge", "<<: {version: daiksud/gh-settings@v0, repository: {about: {description: root}}}\n", uimodel.Repository{About: &uimodel.About{Description: uiConfigPtr("root")}}},
		{"settings merge", "version: daiksud/gh-settings@v0\nrepository: {settings: {<<: {general: {features: {issues: true}}}}}\n", uimodel.Repository{Settings: &uimodel.RepositorySettings{General: &uimodel.General{Features: uimodel.Features{Issues: uiConfigPtr(true)}}}}},
		{"value alias", "version: daiksud/gh-settings@v0\nrepository: {about: {description: &text Example, website: *text}}\n", uimodel.Repository{About: &uimodel.About{Description: uiConfigPtr("Example"), Website: uiConfigPtr("Example")}}},
		{"explicit overrides merge", "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {<<: {issues: true}, issues: false}}}}\n", uimodel.Repository{Settings: &uimodel.RepositorySettings{General: &uimodel.General{Features: uimodel.Features{Issues: uiConfigPtr(false)}}}}},
		{"quoted type-looking strings", "version: daiksud/gh-settings@v0\nrepository: {about: {description: 'true', website: 'null'}}\n", uimodel.Repository{About: &uimodel.About{Description: uiConfigPtr("true"), Website: uiConfigPtr("null")}}},
		{"multiline", "version: daiksud/gh-settings@v0\nrepository:\n  about:\n    description: |\n      first line\n      second line\n", uimodel.Repository{About: &uimodel.About{Description: uiConfigPtr("first line\nsecond line\n")}}},
		{"uppercase tagged bool", "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: !!bool TRUE}}}}\n", uimodel.Repository{Settings: &uimodel.RepositorySettings{General: &uimodel.General{Features: uimodel.Features{Issues: uiConfigPtr(true)}}}}},
		{"list element alias", "version: daiksud/gh-settings@v0\nrepository: {about: {description: &topic github, topics: [*topic, settings]}}\n", uimodel.Repository{About: &uimodel.About{Description: uiConfigPtr("github"), Topics: uiConfigPtr([]string{"github", "settings"})}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := DecodeUIConfig([]byte(test.input))
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("decoded %#v; want %#v; error=%v", got, test.want, err)
			}
			encoded, err := EncodeUIConfig(got)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodeUIConfig(encoded)
			if err != nil || !reflect.DeepEqual(again, test.want) {
				t.Fatalf("string style/alias round trip: %s, %v", encoded, err)
			}
		})
	}
}

func TestUIConfigYAMLUnsafeShapesReturnNoPartialValues(t *testing.T) {
	for _, input := range []string{
		"version: daiksud/gh-settings@v0\nrepository: {about: {'description': one, \"description\": two}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {&field description: one, *field: two}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {&merge <<: {description: one}, *merge: {website: two}}}\n",
		"!!str {version: daiksud/gh-settings@v0}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: !!str {description: one}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {topics: !!str []}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {<<: !!str {description: one}}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {<<: !!str [{description: one}]}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {<<: {description: one}, <<: {website: two}}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {42: value}}\n",
		"version: daiksud/gh-settings@v0\nnull: value\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {description: !!binary /w==}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {description: &value false, website: *value}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: !!bool yes}}}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: !!bool on}}}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: !!bool nope}}}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {description: valid}, settings: {general: {features: {issues: null}}}}\n",
		"- version: daiksud/gh-settings@v0\n", "true\n", "", "version: [\n",
		"version: daiksud/gh-settings@v0\n---\nversion: daiksud/gh-settings@v0\n",
	} {
		got, err := DecodeUIConfig([]byte(input))
		if err == nil || !reflect.DeepEqual(got, uimodel.Repository{}) {
			t.Fatalf("unsafe YAML returned partial values: %q, %#v, %v", input, got, err)
		}
	}
}

func TestUIConfigStringEncodingGoldens(t *testing.T) {
	for _, test := range []struct {
		name  string
		about uimodel.About
		want  string
	}{
		{"type-looking strings", uimodel.About{Description: uiConfigPtr("true"), Website: uiConfigPtr("null")}, "version: daiksud/gh-settings@v0\nrepository:\n    about:\n        description: \"true\"\n        website: \"null\"\n"},
		{"multiline string", uimodel.About{Description: uiConfigPtr("first line\nsecond line\n")}, "version: daiksud/gh-settings@v0\nrepository:\n    about:\n        description: |\n            first line\n            second line\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := EncodeUIConfig(uimodel.Repository{About: &test.about})
			if err != nil || string(got) != test.want {
				t.Fatalf("encoded %q; want %q; error=%v", got, test.want, err)
			}
		})
	}
}
