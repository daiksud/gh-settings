package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

var codeQLLanguageAPI = map[string]string{"github_actions": "actions", "c_cpp": "c-cpp", "csharp": "csharp", "go": "go", "java_kotlin": "java-kotlin", "javascript_typescript": "javascript-typescript", "python": "python", "ruby": "ruby", "swift": "swift"}

var codeQLThreatModelAPI = map[string]string{"remote_sources": "remote", "remote_and_local_sources": "remote_and_local"}

func (c *CodeQLDefaultSetupClient) ReadConfiguration(repo repository.Repository) (settings.CodeQLConfigurationObservation, error) {
	if err := readPersonalPublicSecurityTarget(c.rest, repo); err != nil {
		return settings.CodeQLConfigurationObservation{}, err
	}
	var raw map[string]json.RawMessage
	if err := c.rest.Get(codeQLDefaultSetupPath(repo), &raw); err != nil {
		return settings.CodeQLConfigurationObservation{}, fmt.Errorf("read CodeQL configuration: %w", err)
	}
	var state *string
	if json.Unmarshal(raw["state"], &state) != nil || state == nil || *state != "configured" {
		return settings.CodeQLConfigurationObservation{}, fmt.Errorf("CodeQL configuration requires an already configured default setup; no prerequisite is auto-enabled")
	}
	return decodeCodeQLConfiguration(raw)
}

func decodeCodeQLConfiguration(raw map[string]json.RawMessage) (settings.CodeQLConfigurationObservation, error) {
	var langs *[]string
	var suite, runner, threat *string
	var label *string
	if json.Unmarshal(raw["languages"], &langs) != nil || langs == nil || json.Unmarshal(raw["query_suite"], &suite) != nil || suite == nil || json.Unmarshal(raw["runner_type"], &runner) != nil || runner == nil || raw["runner_label"] == nil || json.Unmarshal(raw["runner_label"], &label) != nil || json.Unmarshal(raw["threat_model"], &threat) != nil || threat == nil {
		return settings.CodeQLConfigurationObservation{}, fmt.Errorf("incomplete/null or invalid CodeQL configuration")
	}
	s := settings.CodeQLConfigurationState{QuerySuite: *suite}
	for _, apiLang := range *langs {
		choice := ""
		for ui, native := range codeQLLanguageAPI {
			if native == apiLang {
				choice = ui
			}
		}
		if choice == "" {
			return settings.CodeQLConfigurationObservation{}, fmt.Errorf("unsupported CodeQL language")
		}
		s.Languages = append(s.Languages, choice)
	}
	switch *runner {
	case "standard":
		s.Runner.Type = "standard_github_runner"
	case "labeled":
		s.Runner.Type = "labeled_runner"
	default:
		return settings.CodeQLConfigurationObservation{}, fmt.Errorf("unsupported CodeQL runner type")
	}
	if label != nil {
		s.Runner.Label = *label
	}
	for ui, native := range codeQLThreatModelAPI {
		if native == *threat {
			s.ThreatModel = ui
		}
	}
	if s.ThreatModel == "" {
		return settings.CodeQLConfigurationObservation{}, fmt.Errorf("unsupported CodeQL threat model")
	}

	return settings.NewCodeQLConfigurationObservation(s)
}

// UpdateConfiguration sends only declared members and verifies all observed
// configuration companions. Default-setup state is never sent or enabled here.
func (c *CodeQLDefaultSetupClient) UpdateConfiguration(repo repository.Repository, d settings.CodeQLConfigurationDeclaration) (bool, error) {
	if err := d.Validate(); err != nil {
		return false, &securityPreWriteFailure{err}
	}
	if d.Empty() {
		return false, nil
	}
	current, err := c.ReadConfiguration(repo)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	_, changes, err := settings.CompareCodeQLConfiguration(current, d)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	if len(changes) == 0 {
		return false, nil
	}
	before, _ := current.State()
	expected := settings.ExpectedCodeQLConfiguration(before, d)
	payload := map[string]any{}
	if d.Languages != nil {
		langs := []string{}
		for _, v := range expected.Languages {
			langs = append(langs, codeQLLanguageAPI[v])
		}
		payload["languages"] = langs
	}
	if d.QuerySuite != nil {
		payload["query_suite"] = *d.QuerySuite
	}
	if d.ThreatModel != nil {
		payload["threat_model"] = codeQLThreatModelAPI[*d.ThreatModel]
	}
	if d.Runner != nil {
		if d.Runner.Type == "standard_github_runner" {
			payload["runner_type"] = "standard"
			payload["runner_label"] = nil
		} else {
			payload["runner_type"] = "labeled"
			payload["runner_label"] = d.Runner.Label
		}
	}

	body, _ := json.Marshal(payload)
	var response json.RawMessage
	if err := c.rest.Patch(codeQLDefaultSetupPath(repo), bytes.NewReader(body), &response); err != nil {
		return false, fmt.Errorf("update CodeQL configuration (write effects and analysis may already have started): %w", err)
	}
	actual, err := c.ReadConfiguration(repo)
	if err != nil {
		return false, fmt.Errorf("verify CodeQL configuration (write effects and analysis may already have started): %w", err)
	}
	state, err := actual.State()
	if err != nil {
		return false, fmt.Errorf("verify CodeQL configuration (write effects may already have applied): %w", err)
	}
	if !reflect.DeepEqual(expected, state) {
		return false, fmt.Errorf("CodeQL configuration or omitted companions are not confirmed; validation may be pending (write effects and analysis may already have started)")
	}
	return true, nil
}
