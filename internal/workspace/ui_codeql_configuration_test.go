package workspace

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
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
                    scan_settings:
                        runner_type: labeled_runner
                        runner_label: scan-pool
                        query_suite: extended
                        threat_model: remote_sources
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
	exported, err := github.UIFromExportDeclaration(d)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeConfig(exported)
	if err != nil || string(encoded) != input {
		t.Fatal(string(encoded), err)
	}
	for _, bad := range []string{
		"languages: []", "languages: [go, go]", "languages: [actions]",
		"scan_settings: {query_suite: unknown}", "scan_settings: {query_suite: [default]}",
		"scan_settings: {runner_type: standard_github_runner, runner_label: ''}",
		"scan_settings: {runner_type: labeled_runner}", "scan_settings: {runner_label: scan-pool}",
		"scan_settings: {threat_model: remote}", "scan_settings: {threat_model: remote_and_local}",
		"scan_settings: {threat_model: null}", "scan_settings: {threat_model: false}",
		"scan_settings: {threat_model: []}", "scan_settings: {threat_model: {}}",
		"scan_settings: {threat_model: unknown}", "scan_settings: {threat_model: ''}",
		"default_setup: false, scan_settings: {threat_model: remote_sources}",
		"default_setup: false, scan_settings: {query_suite: extended}",
		"runner: {type: standard_github_runner}", "threat_model: remote_sources",
		"query_suites: extended", "scan_settings: {query_suites: extended}",
	} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {code_security: {codeql_analysis: {" + bad + "}}}}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal(bad)
		}
	}
}

func TestCodeQLThreatModelUIRoundTrip(t *testing.T) {
	for _, value := range []string{"remote_sources", "remote_and_local_sources"} {
		input := "version: daiksud/gh-settings@v0\nrepository:\n    settings:\n        advanced_security:\n            code_security:\n                codeql_analysis:\n                    scan_settings:\n                        threat_model: " + value + "\n"
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
		if err != nil || projected.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity.CodeQLAnalysis.ScanSettings.ThreatModel == nil || *projected.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity.CodeQLAnalysis.ScanSettings.ThreatModel != value {
			t.Fatal(projected, err)
		}
	}
}
