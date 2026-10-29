package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRepositoryCreationPolicyVerificationUsesRawEvidence(t *testing.T) {
	for _, test := range []struct {
		control settings.CreationPolicyControl
		path    string
	}{
		{settings.IssuePermissionsControl, "repository.settings.general.features.issue_permissions.creation_allowed_by"},
		{settings.PullRequestPermissionsControl, "repository.settings.general.features.pull_request_permissions.creation_allowed_by"},
	} {
		t.Run(string(test.control), func(t *testing.T) {
			intent := func(value string) settings.RepositoryDeclaration {
				if test.control == settings.IssuePermissionsControl {
					return settings.RepositoryDeclaration{IssueCreationPolicy: &value}
				}
				return settings.RepositoryDeclaration{PullRequestCreationPolicy: &value}
			}
			observation := func(raw string, normalized bool) repositoryCreationObservations {
				value := raw
				record := settings.ObservationNormalization{}
				if normalized {
					value = "ALL"
					record = settings.ObservationNormalization{Path: test.path, RawValue: raw, Fallback: "all_users"}
				}
				return repositoryCreationObservations{test.control: {Value: &value, Normalization: record}}
			}
			for _, desired := range []string{"ALL", "COLLABORATORS_ONLY"} {
				if !repositoryCreationPoliciesMatch(observation(desired, false), intent(desired)) {
					t.Fatal("canonical raw value did not verify", desired)
				}
				if repositoryCreationPoliciesMatch(nil, intent(desired)) {
					t.Fatal("missing requested observation verified")
				}
				for _, raw := range []string{"FUTURE", "OTHER", "", " \t "} {
					actual := observation(raw, true)
					if repositoryCreationPoliciesMatch(actual, intent(desired)) {
						t.Fatal("fallback verified an unknown raw value", desired, raw)
					}
					if got, valid := actual.rawValue(test.control); !valid || got != raw {
						t.Fatal("exact raw token lost", raw, got, valid)
					}
				}
			}
			for _, corrupt := range []func(*preparedCreationPolicy){
				func(p *preparedCreationPolicy) { p.Value = nil },
				func(p *preparedCreationPolicy) { *p.Value = "COLLABORATORS_ONLY" },
				func(p *preparedCreationPolicy) { p.Normalization.Path = "wrong" },
				func(p *preparedCreationPolicy) { p.Normalization.Fallback = "collaborators_only" },
				func(p *preparedCreationPolicy) { p.Normalization.RawValue = "ALL" },
				func(p *preparedCreationPolicy) { p.Normalization.RawValue = "COLLABORATORS_ONLY" },
			} {
				actual := observation("FUTURE", true)
				value := actual[test.control]
				corrupt(&value)
				actual[test.control] = value
				if _, valid := actual.rawValue(test.control); valid {
					t.Fatal("forged provenance manufactured raw evidence", actual)
				}
			}
			if repositoryCreationPoliciesMatch(observation("FUTURE", false), intent("FUTURE")) {
				t.Fatal("unsupported unnormalized value verified invalid intent")
			}
		})
	}
	description := "unrelated"
	if !repositoryCreationPoliciesMatch(repositoryCreationObservations{settings.IssuePermissionsControl: {}, settings.PullRequestPermissionsControl: {}}, settings.RepositoryDeclaration{Description: &description}) {
		t.Fatal("unmanaged malformed policy entered verification")
	}
	all, collaborators := "ALL", "COLLABORATORS_ONLY"
	if repositoryCreationPoliciesMatch(repositoryCreationObservations{settings.IssuePermissionsControl: {Value: &all}, settings.PullRequestPermissionsControl: {Value: &all}}, settings.RepositoryDeclaration{IssueCreationPolicy: &all, PullRequestCreationPolicy: &collaborators}) {
		t.Fatal("one matching policy hid another mismatch")
	}
}
