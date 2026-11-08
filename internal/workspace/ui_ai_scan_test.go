package workspace

import (
	"strings"
	"testing"
)

func TestAIScanPullRequestsUIRoundTripAndStrictPresence(t *testing.T) {
	for _, v := range []string{"true", "false"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {ai_scan: {ai_scan_for_pull_requests: " + v + "}}}}}\n"
		ui, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		got := ui.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity.AIScan.AIScanForPullRequests
		if got == nil || *got != (v == "true") {
			t.Fatal(got)
		}
		encoded, err := EncodeConfig(ui)
		if err != nil || !strings.Contains(string(encoded), "ai_scan_for_pull_requests: "+v) {
			t.Fatal(string(encoded), err)
		}
	}
	for _, v := range []string{"null", "'false'", "1", "[]", "{}"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {ai_scan: {ai_scan_for_pull_requests: " + v + "}}}}}\n")); err == nil {
			t.Fatal(v)
		}
	}
}
