package workspace

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func TestTopicsConfigPresenceAndRoundTrip(t *testing.T) {
	for _, test := range []struct {
		input   string
		names   []string
		managed bool
	}{
		{"schemaVersion: 1", nil, false},
		{"schemaVersion: 1\nsettings: {topics: {}}", nil, false},
		{"schemaVersion: 1\nsettings: {topics: {names: []}}", []string{}, true},
		{"schemaVersion: 1\nsettings: {topics: {names: [github-cli, settings-as-code]}}", []string{"github-cli", "settings-as-code"}, true},
		{"schemaVersion: 1\nsettings: {topics: {names: [&name github-cli, settings-as-code]}}", []string{"github-cli", "settings-as-code"}, true},
		{"schemaVersion: 1\nsettings: {topics: {<<: {names: [github-cli]}}}", []string{"github-cli"}, true},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := DecodeConfig([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			if (got.Topics.Names != nil) != test.managed {
				t.Fatalf("presence=%+v", got)
			}
			if test.managed && !reflect.DeepEqual(*got.Topics.Names, test.names) {
				t.Fatalf("names=%v", *got.Topics.Names)
			}
			data, err := EncodeConfig(got)
			if err != nil {
				t.Fatal(err)
			}
			round, err := DecodeConfig(data)
			if err != nil || !reflect.DeepEqual(got, round) {
				t.Fatalf("round=%+v error=%v", round, err)
			}
		})
	}
}
func TestTopicsConfigRejectsInvalidLists(t *testing.T) {
	for _, input := range []string{"topics: null", "topics: []", "topics: {names: null}", "topics: {names: github-cli}", "topics: {names: {name: github-cli}}", "topics: {names: [true]}", "topics: {names: [42]}", "topics: {names: [null]}", "topics: {names: [[]]}", "topics: {names: [a, a]}", "topics: {names: [Upper]}", "topics: {names: ['a b']}", "topics: {names: ['']}", "topics: {unknown: []}", "topics: {names: [], names: []}"} {
		got, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  " + input))
		if err == nil || !reflect.DeepEqual(got, settings.Declaration{}) {
			t.Fatalf("accepted %q: %+v %v", input, got, err)
		}
	}
}
func TestTopicsConfigStableEncoding(t *testing.T) {
	names := []string{"z", "a"}
	data, err := EncodeConfig(settings.Declaration{Topics: settings.TopicsDeclaration{Names: &names}})
	if err != nil || !strings.Contains(string(data), "- a\n            - z") || names[0] != "z" {
		t.Fatalf("data=%s names=%v err=%v", data, names, err)
	}
}
