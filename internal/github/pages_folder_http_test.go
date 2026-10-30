package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesFolderHTTPReadRequiresCompleteTypedLegacyState(t *testing.T) {
	for _, folder := range []string{"/", "/docs", "/future", "", " \t"} {
		for _, invalid := range []string{"", "branch", "folder", "missing branch", "null branch", "missing folder", "null folder", "cname", "https_enforced", "public", "mode"} {
			t.Run(fmt.Sprintf("%q/%s", folder, invalid), func(t *testing.T) {
				source := map[string]any{"branch": "main", "path": folder}
				state := map[string]any{"build_type": "legacy", "source": source, "cname": nil, "https_enforced": false, "public": true}
				switch invalid {
				case "branch":
					source["branch"] = "bad branch"
				case "folder":
					source["path"] = false
				case "missing branch":
					delete(source, "branch")
				case "null branch":
					source["branch"] = nil
				case "missing folder":
					delete(source, "path")
				case "null folder":
					source["path"] = nil
				case "cname":
					delete(state, "cname")
				case "https_enforced", "public":
					state[invalid] = nil
				case "mode":
					state["build_type"] = "future"
				}
				body, _ := json.Marshal(state)
				client := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.Method != "GET" || r.URL.Path != "/repos/owner/current/pages" {
						t.Fatal(r.Method, r.URL)
					}
					return actionsResponse(r, 200, string(body)), nil
				}).rest, NormalizationWarning: func(settings.ObservationNormalization) { t.Fatal("pure Read warned") }}
				o, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
				if invalid != "" {
					if err == nil {
						t.Fatal("invalid acquired", invalid)
					}
					if _, err := o.Values(); err == nil {
						t.Fatal("failed read acquired")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				effective, err := o.Values()
				wantFolder := folder
				wantRecord := settings.ObservationNormalization{}
				if folder != "/" && folder != "/docs" {
					wantFolder = "/"
					wantRecord = settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: folder, Fallback: "root"}
				}
				raw, rawErr := o.RawSource()
				if err != nil || rawErr != nil || effective.Source.Branch != "main" || effective.Source.Path != wantFolder || raw.Path != folder || o.FolderNormalization() != wantRecord {
					t.Fatal(effective, raw, o.FolderNormalization(), err, rawErr)
				}
			})
		}
	}
}

func TestPagesFolderHTTPUpdateVerifiesRawSourceAndScopesWarnings(t *testing.T) {
	for _, raw := range []string{"/future", "", " \t"} {
		for _, target := range []string{"/", "/docs", "same branch root", "omitted", "workflow"} {
			for _, outcome := range []string{"applied", "ignored", "branch drift", "folder drift", "visibility drift"} {
				t.Run(fmt.Sprintf("%q/%s/%s", raw, target, outcome), func(t *testing.T) {
					state := map[string]any{"build_type": "legacy", "source": map[string]any{"branch": "main", "path": raw}, "cname": nil, "https_enforced": true, "public": true}
					d := settings.PagesDeclaration{}
					wantPayload := map[string]any{}
					switch target {
					case "same branch root":
						d.Source = &settings.PagesSource{Branch: "main", Path: "/"}
						wantPayload["source"] = map[string]any{"branch": "main", "path": "/"}
					case "omitted":
						d.HTTPSEnforced = actionsUIPtr(false)
						wantPayload["https_enforced"] = false
					case "workflow":
						d.BuildType = actionsUIPtr("workflow")
						wantPayload["build_type"] = "workflow"
					default:
						d.Source = &settings.PagesSource{Branch: "release/site", Path: target}
						wantPayload["source"] = map[string]any{"branch": "release/site", "path": target}
					}
					writes := 0
					var warnings []settings.ObservationNormalization
					client := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
						if r.URL.Path == "/repos/owner/current" {
							if r.Method != "GET" {
								t.Fatal("identity mutation")
							}
							return actionsResponse(r, 200, `{"id":123,"full_name":"owner/current","archived":false}`), nil
						}
						if r.URL.Path != "/repos/owner/current/pages" {
							t.Fatal(r.URL)
						}
						if r.Method == "GET" {
							body, _ := json.Marshal(state)
							return actionsResponse(r, 200, string(body)), nil
						}
						if r.Method != "PUT" {
							t.Fatal("site lifecycle/build mutation", r.Method)
						}
						writes++
						var payload map[string]any
						if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(payload, wantPayload) {
							t.Fatal("unowned payload", payload, wantPayload)
						}
						if outcome != "ignored" {
							for key, value := range payload {
								state[key] = value
							}
						}
						if state["build_type"] == "workflow" {
							state["source"] = map[string]any{"branch": "latent", "path": "/latent"}
						}
						if outcome == "branch drift" {
							state["source"].(map[string]any)["branch"] = "other"
						}
						if outcome == "folder drift" {
							state["source"].(map[string]any)["path"] = "/different"
						}
						if outcome == "visibility drift" {
							state["public"] = false
						}
						return actionsResponse(r, 204, ""), nil
					}).rest, NormalizationWarning: func(n settings.ObservationNormalization) { warnings = append(warnings, n) }}
					wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, d)
					success := outcome == "applied" || target == "workflow" && (outcome == "branch drift" || outcome == "folder drift")
					if (err == nil) != success || wrote != success || writes != 1 {
						t.Fatal(wrote, err, writes, success)
					}
					var pre interface{ ReadFailedBeforeWrite() }
					if errors.As(err, &pre) {
						t.Fatal("write attempt classified no-write", err)
					}
					want := []settings.ObservationNormalization(nil)
					if target != "omitted" && target != "workflow" {
						want = append(want, settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: raw, Fallback: "root"})
						if outcome == "ignored" || outcome == "folder drift" {
							value := raw
							if outcome == "folder drift" {
								value = "/different"
							}
							want = append(want, settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: value, Fallback: "root"})
						}
					}
					if !reflect.DeepEqual(warnings, want) {
						t.Fatal("warning scope/freshness", warnings, want)
					}
				})
			}
		}
	}
}
