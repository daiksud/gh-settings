package workspace

import (
	"strings"
	"testing"
)

func TestUIAIScanRepositoriesRoundtrip(t *testing.T) {
	for _, enabled := range []string{"true", "false"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {ai_scan: {ai_scan_for_repositories: " + enabled + "}}}}}\n"
		value, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeConfig(value)
		if err != nil || !strings.Contains(string(data), "ai_scan_for_repositories: "+enabled) {
			t.Fatal(string(data), err)
		}
	}
}

func TestUIAIScanRepositoriesRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"null", "'false'", "1", "[]", "{}"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {ai_scan: {ai_scan_for_repositories: " + value + "}}}}}\n")); err == nil {
			t.Fatal(value)
		}
	}
}
