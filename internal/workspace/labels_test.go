package workspace

import (
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestLabelsConfigOmissionEmptyAndRoundTrip(t *testing.T) {
	for _, test := range []struct {
		input   string
		managed bool
		count   int
	}{
		{"schemaVersion: 1", false, 0},
		{"schemaVersion: 1\nsettings: {labels: []}", true, 0},
		{"schemaVersion: 1\nsettings: {labels: [{name: bug, color: ABCDEF, description: ''}]}", true, 1},
		{"schemaVersion: 1\nsettings: {labels: [{<<: {name: bug, color: ABCDEF}, description: &text Test}, {name: other, description: *text}]}", true, 2},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := DecodeConfig([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			if (got.Labels != nil) != test.managed {
				t.Fatalf("declaration=%+v", got)
			}
			if test.managed && len(*got.Labels) != test.count {
				t.Fatalf("labels=%+v", *got.Labels)
			}
			if test.count == 1 && (*got.Labels)[0].Color != nil && *(*got.Labels)[0].Color != "abcdef" {
				t.Fatal("color not canonical")
			}
			data, err := EncodeConfig(got)
			if err != nil {
				t.Fatal(err)
			}
			round, err := DecodeConfig(data)
			if err != nil || !reflect.DeepEqual(round, got) {
				t.Fatalf("round=%+v err=%v data=%s", round, err, data)
			}
		})
	}
}

func TestLabelsConfigRejectsInvalidDeclarations(t *testing.T) {
	for _, labels := range []string{"null", "{}", "bug", "[null]", "[bug]", "[{}]", "[{name: null}]", "[{name: 42}]", "[{name: ''}]", "[{name: bug, color: 123456}]", "[{name: bug, color: null}]", "[{name: bug, color: '#abcdef'}]", "[{name: bug, description: null}]", "[{name: bug, description: false}]", "[{name: bug, new_name: other}]", "[{name: bug, archived: true}]", "[{name: bug, name: other}]", "[{name: bug}, {name: BUG}]"} {
		got, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  labels: " + labels + "\n"))
		if err == nil || !reflect.DeepEqual(got, settings.Declaration{}) {
			t.Fatalf("accepted %s: %+v err=%v", labels, got, err)
		}
	}
}

func TestLabelsConfigExportSortsNamesAndPreservesOmittedFields(t *testing.T) {
	data := []byte("schemaVersion: 1\nsettings: {labels: [{name: z, description: ''}, {name: a, color: ABCDEF}]}")
	declaration, err := DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeConfig(declaration)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Index(text, "name: a") > strings.Index(text, "name: z") || !strings.Contains(text, "color: abcdef") || !strings.Contains(text, `description: ""`) {
		t.Fatalf("encoded=%s", text)
	}
	if (*declaration.Labels)[0].Color != nil || (*declaration.Labels)[1].Description != nil {
		t.Fatal("injected omitted values")
	}
}
