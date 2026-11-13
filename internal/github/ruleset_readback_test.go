package github

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEveryRulesetMutationRequiresIndependentReadback(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, empty := range []bool{false, true} {
			beforeJSON := basicRulesetJSON
			if empty {
				beforeJSON = strings.Replace(beforeJSON, `[{"type":"deletion"},{"type":"non_fast_forward"}]`, `[]`, 1)
			}
			afterJSON := strings.Replace(beforeJSON, `"enforcement":"active"`, `"enforcement":"disabled"`, 1)
			for _, test := range []struct {
				name, body       string
				status           int
				transportFailure bool
			}{
				{name: "matching", body: afterJSON},
				{name: "matching reordered", body: strings.Replace(afterJSON, `[{"type":"deletion"},{"type":"non_fast_forward"}]`, `[{"type":"non_fast_forward"},{"type":"deletion"}]`, 1)},
				{name: "denied", status: 403},
				{name: "missing object", status: 404},
				{name: "redirect", status: 302},
				{name: "transport", transportFailure: true},
				{name: "malformed", body: `{`},
				{name: "null", body: `null`},
				{name: "missing ID", body: strings.Replace(afterJSON, `"id":7,`, ``, 1)},
				{name: "wrong ID", body: strings.Replace(afterJSON, `"id":7`, `"id":8`, 1)},
				{name: "foreign owner", body: strings.Replace(afterJSON, `Owner/Current`, `other/repository`, 1)},
				{name: "inherited", body: strings.Replace(afterJSON, `"source_type":"Repository"`, `"source_type":"Organization"`, 1)},
				{name: "wrong name", body: strings.Replace(afterJSON, `"name":"basic"`, `"name":"changed"`, 1)},
				{name: "wrong target", body: strings.Replace(afterJSON, `"target":"branch"`, `"target":"tag"`, 1)},
				{name: "ignored enforcement", body: beforeJSON},
				{name: "normalized fallback", body: strings.Replace(afterJSON, `"enforcement":"disabled"`, `"enforcement":"FUTURE"`, 1)},
				{name: "missing bypass visibility", body: strings.Replace(afterJSON, `"bypass_actors":[],`, ``, 1)},
				{name: "bypass changed", body: strings.Replace(afterJSON, `"bypass_actors":[]`, `"bypass_actors":[{"actor_type":"RepositoryRole","actor_id":2,"bypass_mode":"always"}]`, 1)},
				{name: "conditions changed", body: strings.Replace(afterJSON, `"~DEFAULT_BRANCH"`, `"~ALL"`, 1)},
				{name: "rules incomplete", body: strings.Replace(afterJSON, `"rules":[`, `"unknown_rules":[`, 1)},
				{name: "rules null", body: afterJSON[:strings.Index(afterJSON, `"rules":`)] + `"rules":null}`},
			} {
				t.Run(operation+"/"+map[bool]string{false: "basic", true: "empty"}[empty]+"/"+test.name, func(t *testing.T) {
					before, err := decodeRulesetPolicyFromTest([]byte(beforeJSON))
					if err != nil {
						t.Fatal(err)
					}
					after, err := decodeRulesetPolicyFromTest([]byte(afterJSON))
					if err != nil {
						t.Fatal(err)
					}
					var prior *settings.RulesetDeclaration
					var id int64
					wantCalls := []string{"POST", "GET"}
					if operation == "update" {
						prior, id, wantCalls = &before, 7, []string{"GET", "PUT", "GET"}
					}
					change, err := settings.NewRulesetChange(operation, id, prior, &after)
					if err != nil {
						t.Fatal(err)
					}
					var calls []string
					cause := errors.New("readback connection lost")
					rest, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true,
						CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
						Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
							calls = append(calls, req.Method)
							if len(calls) > len(wantCalls) || req.Method != wantCalls[len(calls)-1] {
								t.Fatalf("unexpected request %s %s; calls=%v", req.Method, req.URL, calls)
							}
							wantPath := "/repos/owner/current/rulesets/7"
							if req.Method == "POST" {
								wantPath = "/repos/owner/current/rulesets"
							}
							if req.URL.Path != wantPath || req.Method == "GET" && req.URL.RawQuery != "includes_parents=false" {
								t.Fatal(req.URL)
							}
							if len(calls) == len(wantCalls) {
								if test.transportFailure {
									return nil, cause
								}
								status := test.status
								if status == 0 {
									status = 200
								}
								response := labelsResponse(req, status, test.body)
								if status == 302 {
									response.Header.Set("Location", "https://api.github.com/repos/other/wrong/rulesets/7")
								}
								return response, nil
							}
							if req.Method == "GET" {
								return labelsResponse(req, 200, beforeJSON), nil
							}
							return labelsResponse(req, 200, afterJSON), nil
						})})
					if err != nil {
						t.Fatal(err)
					}
					client := &Client{rest: rest}
					gotID, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
					if gotID != 7 || (err == nil) != strings.HasPrefix(test.name, "matching") || !reflect.DeepEqual(calls, wantCalls) {
						t.Fatal(gotID, err, calls)
					}
					var prewrite interface {
						error
						ReadFailedBeforeWrite()
					}
					if errors.As(err, &prewrite) || err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
						t.Fatal("lost uncertain effects", err)
					}
					if test.transportFailure && !errors.Is(err, cause) {
						t.Fatal("lost transport cause", err)
					}
				})
			}
		}
	}
}
