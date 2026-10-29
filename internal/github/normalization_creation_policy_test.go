package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestCreationPolicyProjectionPreservesAcquisition(t *testing.T) {
	for _, control := range []struct {
		kind settings.CreationPolicyControl
		path string
	}{
		{settings.IssuePermissionsControl, "repository.settings.general.features.issue_permissions.creation_allowed_by"},
		{settings.PullRequestPermissionsControl, "repository.settings.general.features.pull_request_permissions.creation_allowed_by"},
	} {
		t.Run(string(control.kind), func(t *testing.T) {
			for _, test := range []struct {
				raw, effective string
				normalized     bool
			}{
				{"ALL", "ALL", false}, {"COLLABORATORS_ONLY", "COLLABORATORS_ONLY", false},
				{"FUTURE", "ALL", true}, {"", "ALL", true}, {" \t\n ", "ALL", true}, {"all", "ALL", true},
			} {
				raw := test.raw
				got, err := prepareCreationPolicyObservation(&raw, control.kind)
				if err != nil || got.Value == nil || *got.Value != test.effective {
					t.Fatal("wrong projection", test, got, err)
				}
				want := settings.ObservationNormalization{}
				if test.normalized {
					want = settings.ObservationNormalization{Path: control.path, RawValue: test.raw, Fallback: "all_users"}
				}
				if got.Normalization != want {
					t.Fatal("original value lost or canonical value normalized", test, got.Normalization)
				}
				raw = "caller mutation"
				if *got.Value != test.effective || got.Normalization != want {
					t.Fatal("projection aliases acquired input", got)
				}
			}
			if got, err := prepareCreationPolicyObservation(nil, control.kind); err != nil || got.Value != nil || got.Normalization != (settings.ObservationNormalization{}) {
				t.Fatal("unacquired policy became default", got, err)
			}
		})
	}
	value := "ALL"
	if got, err := prepareCreationPolicyObservation(&value, settings.CreationPolicyControl("future")); err == nil || got.Value != nil || got.Normalization != (settings.ObservationNormalization{}) {
		t.Fatal("unknown control invented policy", got, err)
	}
}
