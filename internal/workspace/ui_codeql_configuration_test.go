package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"strings"
	"testing"
)

func TestCodeQLConfigurationUIRoundTrip(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
    settings:
        advanced_security:
            code_security:
                codeql_analysis:
                    languages:
                        - github_actions
                        - go
                    query_suite: extended
                    runner:
                        type: labeled_runner
                        label: scan-pool
`
	ui, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	d, err := github.DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	if d.Security.CodeQLDefaultSetup != nil || d.Security.CodeQLConfiguration.QuerySuite == nil || d.Security.CodeQLConfiguration.Runner.Label != "scan-pool" {
		t.Fatal(d)
	}
	encoded, err := EncodeConfig(ui)
	if err != nil || string(encoded) != input {
		t.Fatal(string(encoded), err)
	}
	for _, bad := range []string{"languages: []", "languages: [go, go]", "languages: [actions]", "query_suite: unknown", "runner: {type: standard_github_runner, label: ''}", "runner: {type: labeled_runner}", "threat_model: remote", "threat_model: remote_and_local", "threat_model: null", "threat_model: false", "threat_model: []", "threat_model: {}", "threat_model: unknown", "threat_model: ''", "default_setup: false, threat_model: remote_sources", "default_setup: false, query_suite: extended"} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {codeql_analysis: {" + bad + "}}}}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal(bad)
		}
	}
	if !strings.Contains(input, "query_suite: extended") {
		t.Fatal("fixture")
	}
}

func TestCodeQLThreatModelUIRoundTrip(t *testing.T) {
	for _, value := range []string{"remote_sources", "remote_and_local_sources"} {
		input := "version: daiksud/gh-settings@v0\nrepository:\n    settings:\n        advanced_security:\n            code_security:\n                codeql_analysis:\n                    threat_model: " + value + "\n"
		ui, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil || d.Security.CodeQLConfiguration.ThreatModel == nil || *d.Security.CodeQLConfiguration.ThreatModel != value {
			t.Fatal(d, err)
		}
		encoded, err := EncodeConfig(ui)
		if err != nil || string(encoded) != input {
			t.Fatal(string(encoded), err)
		}
		projected, err := github.UIFromExportDeclaration(d)
		if err != nil || projected.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity.CodeQLAnalysis.ThreatModel == nil || *projected.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity.CodeQLAnalysis.ThreatModel != value {
			t.Fatal(projected, err)
		}
	}
}
