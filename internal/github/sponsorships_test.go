package github

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSponsorshipsReadIsCompleteAndSelective(t *testing.T) {
	for _, value := range []string{"true", "false", "null", `"false"`, "0", "[]", "{}"} {
		t.Run(value, func(t *testing.T) {
			client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				var request struct {
					Query     string
					Variables map[string]any
				}
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				want := `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name, followRenames: false) { hasSponsorshipsEnabled } }`
				if req.Method != "POST" || req.URL.Path != "/graphql" || strings.Join(strings.Fields(request.Query), " ") != want || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
					t.Fatal("unexpected read", request)
				}
				return labelsResponse(req, 200, `{"data":{"repository":{"hasSponsorshipsEnabled":`+value+`}}}`), nil
			}))
			got, err := client.ReadSponsorships(repository.Repository{Owner: "owner", Name: "current"})
			valid := value == "true" || value == "false"
			if (err == nil) != valid || (valid && got.Enabled != (value == "true")) || (!valid && got != (settings.SponsorshipsState{})) {
				t.Fatal(got, err)
			}
		})
	}
	for _, body := range []string{`{"data":{"repository":{}}}`, `{"data":{"repository":null}}`, `{"data":{"repository":{"hasSponsorshipsEnabled":true}},"errors":[{"message":"hidden field"}]}`} {
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, body), nil }))
		if state, err := client.ReadSponsorships(repository.Repository{}); err == nil || state != (settings.SponsorshipsState{}) {
			t.Fatal("incomplete read accepted", state, err)
		}
	}
}

func TestSponsorshipsApplyOnlyDeclaredBooleanAndCombinedFields(t *testing.T) {
	yes, no, policy := true, false, "ALL"
	for _, d := range []settings.RepositoryDeclaration{{HasSponsorships: &yes}, {HasSponsorships: &no}, {HasSponsorships: &no, HasPullRequests: &yes, IssueCreationPolicy: &policy}} {
		calls := 0
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			data, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			var request struct {
				Query     string
				Variables map[string]any
			}
			if err := json.Unmarshal(data, &request); err != nil {
				t.Fatal(err)
			}
			if req.Method != "POST" || req.URL.Path != "/graphql" {
				t.Fatal("unexpected method or path", req.Method, req.URL)
			}
			if calls == 1 || calls == 3 {
				if !strings.Contains(request.Query, "followRenames: false") || strings.Contains(request.Query, "hasSponsorshipsEnabled") || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
					t.Fatal("broad lookup", string(data))
				}
				fields := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current"}
				if d.IssueCreationPolicy != nil {
					value := "ALL"
					if calls == 3 {
						value = *d.IssueCreationPolicy
					}
					fields["issueCreationPolicy"] = value
				}
				body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
				return labelsResponse(req, 200, string(body)), nil
			}
			want := map[string]any{"repositoryId": "opaque-id", "hasSponsorshipsEnabled": *d.HasSponsorships}
			result := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current", "hasSponsorshipsEnabled": *d.HasSponsorships}
			if d.HasPullRequests != nil {
				want["hasPullRequestsEnabled"] = *d.HasPullRequests
				result["hasPullRequestsEnabled"] = *d.HasPullRequests
			}
			if d.IssueCreationPolicy != nil {
				want["issueCreationPolicy"] = *d.IssueCreationPolicy
				result["issueCreationPolicy"] = *d.IssueCreationPolicy
			}
			if calls != 2 || !reflect.DeepEqual(request.Variables, map[string]any{"input": want}) || !strings.Contains(request.Query, "hasSponsorshipsEnabled") || strings.Contains(request.Query, "funding") {
				t.Fatal("unexpected mutation", string(data))
			}
			body, err := json.Marshal(map[string]any{"data": map[string]any{"updateRepository": map[string]any{"repository": result}}})
			if err != nil {
				t.Fatal(err)
			}
			return labelsResponse(req, 200, string(body)), nil
		}))
		wantCalls := 2
		if d.IssueCreationPolicy != nil {
			wantCalls = 3
		}
		if err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, d); err != nil || calls != wantCalls {
			t.Fatal(calls, err)
		}
	}
	client := prAccessTestClient(t, transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("omitted settings reached API"); return nil, nil }))
	if err := client.ApplyRepositoryAccess(repository.Repository{}, settings.RepositoryDeclaration{}); err != nil {
		t.Fatal(err)
	}
}

func TestSponsorshipsMutationRequiresConfirmedBoolean(t *testing.T) {
	for _, value := range []string{"true", "null", `"false"`, "0", "missing", "partial error", "wrong identity"} {
		t.Run(value, func(t *testing.T) {
			calls := 0
			client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return labelsResponse(req, 200, `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current"}}}`), nil
				}
				field, suffix, id := `,"hasSponsorshipsEnabled":`+value, "", "opaque-id"
				switch value {
				case "missing":
					field = ""
				case "partial error":
					field = `,"hasSponsorshipsEnabled":false`
					suffix = `,"errors":[{"message":"partial mutation denied"}]`
				case "wrong identity":
					field = `,"hasSponsorshipsEnabled":false`
					id = "different-id"
				}
				return labelsResponse(req, 200, `{"data":{"updateRepository":{"repository":{"id":"`+id+`","nameWithOwner":"owner/current"`+field+`}}}`+suffix+`}`), nil
			}))
			no := false
			err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{HasSponsorships: &no})
			if err == nil || calls != 2 || !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal(calls, err)
			}
		})
	}
}

func TestSponsorshipsObservationRoutingAndDiscard(t *testing.T) {
	target := repository.Repository{Owner: "owner", Name: "current"}
	calls := 0
	observed, err := observeRepository(target, observationFields(t, "repository.has_sponsorships"), nil, nil, nil, nil, func(repo repository.Repository) (settings.SponsorshipsState, error) {
		calls++
		if repo != target {
			t.Fatal(repo)
		}
		return settings.SponsorshipsState{Enabled: false}, nil
	})
	if err != nil || calls != 1 || len(observed.Fields()) != 1 {
		t.Fatal(observed, calls, err)
	}
	cause := errors.New("sponsorship read unavailable")
	observed, err = observeRepository(target, observationFields(t, "repository.has_issues", "repository.has_sponsorships"), func(repository.Repository) (settings.RepositoryState, error) {
		return settings.RepositoryState{HasIssues: true}, nil
	}, nil, nil, nil, func(repository.Repository) (settings.SponsorshipsState, error) {
		return settings.SponsorshipsState{}, cause
	})
	if err != cause || len(observed.Fields()) != 0 {
		t.Fatal("partial comparison escaped", observed, err)
	}
}
