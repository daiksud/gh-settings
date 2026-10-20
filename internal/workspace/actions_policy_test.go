package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestActionsYAMLStrictnessAndRoundtrip(t *testing.T) {
	for _, body := range []string{
		`{}`, `{enabled: false}`, `{allowed_actions: all, sha_pinning_required: false}`,
		`{allowed_actions: local_only}`, `{allowed_actions: selected, selected_actions: {github_owned_allowed: false, verified_allowed: false, patterns_allowed: []}}`,
		`{selected_actions: {patterns_allowed: ['org/*', '!org/bad@*', 'org/*']}}`,
	} {
		t.Run(body, func(t *testing.T) {
			d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: " + body + "}"))
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := EncodeConfig(d)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodeConfig(encoded)
			if err != nil || !reflect.DeepEqual(d, again) {
				t.Fatalf("encoded=%s error=%v", encoded, err)
			}
			if strings.Contains(body, "patterns_allowed: []") && !strings.Contains(string(encoded), "patterns_allowed: []") {
				t.Fatal("empty array lost", string(encoded))
			}
		})
	}
	for _, body := range []string{
		`null`, `[]`, `{enabled: null}`, `{enabled: 0}`, `{enabled: 'false'}`, `{enabled: yes}`, `{allowed_actions: Selected}`, `{allowed_actions: null}`,
		`{sha_pinning_required: null}`, `{unknown: false}`, `{selected_actions: null}`, `{selected_actions: []}`, `{selected_actions: {verified_allowed: null}}`,
		`{selected_actions: {patterns_allowed: null}}`, `{selected_actions: {patterns_allowed: [null]}}`, `{selected_actions: {patterns_allowed: ['']}}`,
		`{selected_actions: {patterns_allowed: [false]}}`, `{selected_actions: {unknown: []}}`, `{allowed_actions: all, selected_actions: {verified_allowed: false}}`,
	} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: " + body + "}")); err == nil {
			t.Fatal("invalid declaration accepted", body)
		}
	}
}
