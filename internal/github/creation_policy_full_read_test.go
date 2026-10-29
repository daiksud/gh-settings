package github

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPublicCreationReadersNormalizeOnlyCompleteStrings(t *testing.T) {
	for _, control := range []settings.CreationPolicyControl{settings.PullRequestPermissionsControl, settings.IssuePermissionsControl} {
		for _, raw := range []string{"ALL", "COLLABORATORS_ONLY", "FUTURE", "", " \t\n "} {
			field, path := "issueCreationPolicy", "repository.settings.general.features.issue_permissions.creation_allowed_by"
			if control == settings.PullRequestPermissionsControl {
				field, path = "pullRequestCreationPolicy", "repository.settings.general.features.pull_request_permissions.creation_allowed_by"
			}
			fields := map[string]any{field: raw, "hasPullRequestsEnabled": false}
			body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
			client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, string(body)), nil }))
			client.NormalizationWarning = func(settings.ObservationNormalization) { t.Fatal("reader emitted unscoped diagnostics") }
			var value string
			var normalization settings.ObservationNormalization
			if control == settings.PullRequestPermissionsControl {
				state, err := client.ReadPRAccess(repository.Repository{Owner: "owner", Name: "current"})
				if err != nil || state.HasPullRequests {
					t.Fatal("complete policy read failed or enabled feature", state, err)
				}
				value, normalization = state.PullRequestCreationPolicy, state.Normalization
			} else {
				state, err := client.ReadIssueCreationPolicy(repository.Repository{Owner: "owner", Name: "current"})
				if err != nil {
					t.Fatal("complete policy read failed", state, err)
				}
				value, normalization = state.Policy, state.Normalization
			}
			if raw == "ALL" || raw == "COLLABORATORS_ONLY" {
				if value != raw || normalization != (settings.ObservationNormalization{}) {
					t.Fatal("canonical policy changed", value, normalization)
				}
			} else if value != "ALL" || normalization != (settings.ObservationNormalization{Path: path, RawValue: raw, Fallback: "all_users"}) {
				t.Fatal("raw normalization evidence lost", raw, value, normalization)
			}
		}
	}
	for _, body := range []string{
		`{"data":{"repository":{"pullRequestCreationPolicy":"FUTURE"}}}`,
		`{"data":{"repository":{"hasPullRequestsEnabled":null,"pullRequestCreationPolicy":"FUTURE"}}}`,
		`{"data":{"repository":{"hasPullRequestsEnabled":"false","pullRequestCreationPolicy":"FUTURE"}}}`,
		`{"data":{"repository":{"hasPullRequestsEnabled":false,"pullRequestCreationPolicy":"FUTURE"}},"errors":[{"message":"partial denied"}]}`,
	} {
		client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, body), nil }))
		if state, err := client.ReadPRAccess(repository.Repository{Owner: "owner", Name: "current"}); err == nil || state != (settings.PRAccessState{}) {
			t.Fatal("failed complete acquisition became default", state, err)
		}
	}
}
