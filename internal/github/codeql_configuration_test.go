package github

import (
	"encoding/json"
	"errors"
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
		{settings.CodeQLConfigurationDeclaration{ThreatModel: &threat}, map[string]any{"threat_model": "remote_and_local"}},
		{settings.CodeQLConfigurationDeclaration{ThreatModel: &threat, QuerySuite: &suite}, map[string]any{"threat_model": "remote_and_local", "query_suite": "extended"}},
		{settings.CodeQLConfigurationDeclaration{QuerySuite: &suite}, map[string]any{"query_suite": "extended"}},
		{settings.CodeQLConfigurationDeclaration{Languages: &langs}, map[string]any{"languages": []any{"actions", "javascript-typescript"}}},
		{settings.CodeQLConfigurationDeclaration{Runner: &settings.CodeQLRunner{Type: "standard_github_runner"}}, map[string]any{"runner_type": "standard", "runner_label": nil}},
		{settings.CodeQLConfigurationDeclaration{Runner: &settings.CodeQLRunner{Type: "labeled_runner", Label: "another-pool"}}, map[string]any{"runner_type": "labeled", "runner_label": "another-pool"}},
	} {
		for _, mode := range []string{"ok", "accepted", "companion", "pending", "denied", "verify-denied"} {
			raw := codeQLConfigRaw()
			if test.d.ThreatModel != nil {
				raw["languages"] = []string{"csharp"}
			}
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
					} else if test.d.ThreatModel != nil {
						raw["runner_label"] = "collateral-pool"
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

func TestCodeQLConfigurationPreservesOmittedThreatModel(t *testing.T) {
	for _, drift := range []bool{false, true} {
		raw := codeQLConfigRaw()
		raw["threat_model"] = "remote_and_local"
		writes := 0
		c := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/repos/owner/current" {
				return actionsResponse(r, 200, `{"visibility":"public","owner":{"type":"User"}}`), nil
			}
			if r.Method == "GET" {
				b, _ := json.Marshal(raw)
				return actionsResponse(r, 200, string(b)), nil
			}
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if r.Method != "PATCH" || len(payload) != 1 || payload["query_suite"] != "extended" {
				t.Fatal("Preview or unlisted setting sent", payload)
			}
			writes++
			raw["query_suite"] = "extended"
			if drift {
				raw["threat_model"] = "remote"
			}
			return actionsResponse(r, 202, `{"run_id":42}`), nil
		}).rest}
		repo := repository.Repository{Owner: "owner", Name: "current"}
		observed, err := c.ReadConfiguration(repo)
		if err != nil {
			t.Fatal(err)
		}
		state, _ := observed.State()
		if state.ThreatModel != "remote_and_local_sources" {
			t.Fatal("Preview value normalized", state)
		}
		suite := "extended"
		changed, err := c.UpdateConfiguration(repo, settings.CodeQLConfigurationDeclaration{QuerySuite: &suite})
		if (err == nil) == drift || changed == drift || writes != 1 {
			t.Fatal(drift, changed, err, writes)
		}
	}
}

func TestCodeQLThreatModelRawReadback(t *testing.T) {
	for _, rawThreat := range []string{"remote", "remote_and_local"} {
		for _, mode := range []string{"ok", "pre-unknown", "missing", "null", "type", "unknown", "ui-spelling", "mismatch", "languages", "runner", "transport"} {
			t.Run(rawThreat+"/"+mode, func(t *testing.T) {
				raw := codeQLConfigRaw()
				raw["languages"] = []string{"csharp", "java-kotlin"}
				desired := "remote_sources"
				raw["threat_model"] = "remote_and_local"
				if rawThreat == "remote_and_local" {
					desired = "remote_and_local_sources"
					raw["threat_model"] = "remote"
				}
				if mode == "pre-unknown" {
					raw["threat_model"] = "future"
				}
				var methods []string
				writes := 0
				c := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/repos/owner/current" {
						if r.Method != "GET" {
							t.Fatal(r.Method)
						}
						return actionsResponse(r, 200, `{"visibility":"public","owner":{"type":"User"}}`), nil
					}
					if r.URL.Path != "/repos/owner/current/code-scanning/default-setup" {
						t.Fatal(r.URL)
					}
					methods = append(methods, r.Method)
					if r.Method == "PATCH" {
						writes++
						var payload map[string]any
						if json.NewDecoder(r.Body).Decode(&payload) != nil || !reflect.DeepEqual(payload, map[string]any{"threat_model": rawThreat}) {
							t.Fatal(payload)
						}
						if mode != "mismatch" {
							raw["threat_model"] = rawThreat
						}
						// A successful acknowledgment is never the verification read.
						return actionsResponse(r, 202, `{"run_id":42}`), nil
					}
					if r.Method != "GET" {
						t.Fatal(r.Method)
					}
					if writes > 0 {
						switch mode {
						case "missing":
							delete(raw, "threat_model")
						case "null":
							raw["threat_model"] = nil
						case "type":
							raw["threat_model"] = false
						case "unknown":
							raw["threat_model"] = "future"
						case "ui-spelling":
							raw["threat_model"] = desired
						case "languages":
							raw["languages"] = []string{"csharp"}
						case "runner":
							raw["runner_label"] = "changed-pool"
						case "transport":
							return nil, errors.New("connection lost")
						}
					}
					b, _ := json.Marshal(raw)
					return actionsResponse(r, 200, string(b)), nil
				}).rest}
				changed, err := c.UpdateConfiguration(repository.Repository{Owner: "owner", Name: "current"}, settings.CodeQLConfigurationDeclaration{ThreatModel: &desired})
				var pre interface{ ReadFailedBeforeWrite() }
				wantMethods := []string{"GET", "PATCH", "GET"}
				if mode == "pre-unknown" {
					wantMethods = []string{"GET"}
				}
				if (err == nil) != (mode == "ok") || changed != (mode == "ok") || errors.As(err, &pre) != (mode == "pre-unknown") || !reflect.DeepEqual(methods, wantMethods) {
					t.Fatal(changed, err, methods)
				}
			})
		}
	}
}
