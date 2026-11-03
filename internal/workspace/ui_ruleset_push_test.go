package workspace

import (
	"reflect"
	"testing"
)

func TestPushRulesetUIRoundtripAndTypes(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - ruleset_name: push
        enforcement_status: active
        target_pushes: true
        rules:
          restrict_file_paths:
            restricted_file_paths: ['secrets/**']
            allowed_exceptions: ['secrets/example']
          restrict_file_path_length: {maximum_file_path_length: 240}
          restrict_file_extensions: {restricted_file_extensions: ['.exe']}
          restrict_file_size: {maximum_file_size: 20, allowed_exceptions: []}
`
	got, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if (*got.Settings.Rulesets)[0].TargetPushes == nil || !*(*got.Settings.Rulesets)[0].TargetPushes {
		t.Fatal("target lost")
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("roundtrip lost", err)
	}
	for _, value := range []string{`target_pushes: null`, `target_pushes: {}`, `rules: {restrict_file_size: {maximum_file_size: 1.5}}`, `rules: {restrict_file_paths: {restricted_file_paths: [null]}}`, `rules: {restrict_file_size: {maximum_file_size: 20, allowed_exceptions: null}}`, `rules: {restrict_file_extensions: {restricted_file_extensions: [true]}}`} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    rulesets:\n      - " + value + "\n")); err == nil {
			t.Fatal("invalid push UI accepted", value)
		}
	}
}
