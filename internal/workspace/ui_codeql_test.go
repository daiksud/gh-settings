package workspace

import (
	"strings"
	"testing"
)

func TestCodeQLDefaultSetupUIRoundTripAndStrictPresence(t *testing.T) {
	for _, v := range []string{"true", "false"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {codeql_analysis: {default_setup: " + v + "}}}}}\n"
		ui, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		got := ui.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity.CodeQLAnalysis.DefaultSetup
		if got == nil || *got != (v == "true") {
			t.Fatal(got)
		}
		encoded, err := EncodeConfig(ui)
		if err != nil || !strings.Contains(string(encoded), "default_setup: "+v) {
			t.Fatal(string(encoded), err)
		}
	}
	for _, v := range []string{"null", "'false'", "1", "[]", "{}"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {codeql_analysis: {default_setup: " + v + "}}}}}\n")); err == nil {
			t.Fatal(v)
		}
	}
}
