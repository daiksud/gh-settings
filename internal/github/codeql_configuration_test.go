package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func codeQLConfigRaw() map[string]any {
	return map[string]any{"state": "configured", "languages": []string{"go", "python"}, "query_suite": "default", "runner_type": "labeled", "runner_label": "scan-pool", "threat_model": "remote"}
}
func TestCodeQLConfigurationCompleteReads(t *testing.T) {
	for _, field := range []string{"state", "languages", "query_suite", "runner_type", "runner_label", "threat_model"} {
		for _, kind := range []string{"missing", "null", "wrong"} {
			raw := codeQLConfigRaw()
			switch kind {
			case "missing":
				delete(raw, field)
			case "null":
				raw[field] = nil
			case "wrong":
				raw[field] = true
			}
			// Explicit null is valid only for standard runners, not the labeled fixture.
			c := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("read mutated")
				}
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, `{"visibility":"public","owner":{"type":"User"}}`), nil
				}
				b, _ := json.Marshal(raw)
				return actionsResponse(r, 200, string(b)), nil
			}).rest}
			if _, err := c.ReadConfiguration(repository.Repository{Owner: "owner", Name: "current"}); err == nil {
				t.Fatal(field, kind)
			}
		}
	}
	for _, status := range []int{403, 404, 409, 422, 500} {
		c := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) { return actionsResponse(r, status, `{}`), nil }).rest}
		if _, err := c.ReadConfiguration(repository.Repository{Owner: "owner", Name: "current"}); err == nil {
			t.Fatal(status)
		}
	}
}
func TestCodeQLConfigurationNarrowWritesAndCompanionVerification(t *testing.T) {
	suite := "extended"
	threat := "remote_and_local_sources"
	langs := []string{"github_actions", "javascript_typescript"}
	for _, test := range []struct {
		d    settings.CodeQLConfigurationDeclaration
		want map[string]any
	}{
		{settings.CodeQLConfigurationDeclaration{QuerySuite: &suite}, map[string]any{"query_suite": "extended"}},
		{settings.CodeQLConfigurationDeclaration{Languages: &langs}, map[string]any{"languages": []any{"actions", "javascript-typescript"}}},
		{settings.CodeQLConfigurationDeclaration{Runner: &settings.CodeQLRunner{Type: "standard_github_runner"}}, map[string]any{"runner_type": "standard", "runner_label": nil}},
		{settings.CodeQLConfigurationDeclaration{Runner: &settings.CodeQLRunner{Type: "labeled_runner", Label: "another-pool"}}, map[string]any{"runner_type": "labeled", "runner_label": "another-pool"}},
		{settings.CodeQLConfigurationDeclaration{ThreatModel: &threat}, map[string]any{"threat_model": "remote_and_local"}},
	} {
		for _, mode := range []string{"ok", "accepted", "companion", "pending", "denied", "verify-denied"} {
			raw := codeQLConfigRaw()
			writes := 0
			c := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, `{"visibility":"public","owner":{"type":"User"}}`), nil
				}
				if r.URL.Path != "/repos/owner/current/code-scanning/default-setup" {
					t.Fatal(r.URL)
				}
				if r.Method == "GET" {
					if writes > 0 && mode == "verify-denied" {
						return actionsResponse(r, 403, `{}`), nil
					}
					b, _ := json.Marshal(raw)
					return actionsResponse(r, 200, string(b)), nil
				}
				writes++
				b, _ := io.ReadAll(r.Body)
				var payload map[string]any
				json.Unmarshal(b, &payload)
				if r.Method != "PATCH" || !reflect.DeepEqual(payload, test.want) {
					t.Fatal(string(b), test.want)
				}
				if mode == "denied" {
					return actionsResponse(r, 422, `{}`), nil
				}
				if mode != "pending" {
					for k, v := range payload {
						raw[k] = v
					}
				}
				if mode == "companion" {
					if test.d.QuerySuite == nil {
						raw["query_suite"] = "extended"
					} else {
						raw["threat_model"] = "remote_and_local"
					}
				}
				status := 200
				if mode == "accepted" {
					status = 202
				}
				return actionsResponse(r, status, `{"run_id":42}`), nil
			}).rest}
			changed, err := c.UpdateConfiguration(repository.Repository{Owner: "owner", Name: "current"}, test.d)
			success := mode == "ok" || mode == "accepted"
			if (err == nil) != success || changed != success || writes != 1 {
				t.Fatal(fmt.Sprint(test.want), mode, changed, err, writes)
			}
			if success {
				changed, err = c.UpdateConfiguration(repository.Repository{Owner: "owner", Name: "current"}, test.d)
				if err != nil || changed || writes != 1 {
					t.Fatal("matching retried", changed, err, writes)
				}
			}
		}
	}
}
