package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestClassicAdminLiteralCodec(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'release/**', do_not_allow_bypassing_the_above_settings: "
	for _, value := range []string{"true", "false"} {
		ui, err := DecodeConfig([]byte(prefix + value + "}]}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := github.DeclarationFromUI(ui); err != nil {
			t.Fatal(err)
		}
		data, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(ui, again) {
			t.Fatal(string(data), err)
		}
	}
	for _, value := range []string{"null", "'false'", "1", "{}", "[]"} {
		if _, err := DecodeConfig([]byte(prefix + value + "}]}}\n")); err == nil {
			t.Fatal("accepted", value)
		}
	}
}
