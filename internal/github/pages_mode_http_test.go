package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesModeHTTPReadRequiresCompleteSourceWithoutInventingBranch(t *testing.T) {
	for _, mode := range []string{"FUTURE_MODE", "", " \t", "workflow"} {
		for _, folder := range []string{"/", "/docs", "/future"} {
			for _, invalid := range []string{"", "missing source", "null source", "missing branch", "null branch", "bad branch", "missing folder", "null folder", "typed folder", "public", "cname"} {
				t.Run(fmt.Sprintf("%q/%s/%s", mode, folder, invalid), func(t *testing.T) {
					source := map[string]any{"branch": "release/site", "path": folder}
					state := map[string]any{"build_type": mode, "source": source, "cname": nil, "https_enforced": true, "public": false}
					switch invalid {
					case "missing source":
						delete(state, "source")
					case "null source":
						state["source"] = nil
					case "missing branch":
						delete(source, "branch")
					case "null branch":
						source["branch"] = nil
					case "bad branch":
						source["branch"] = "bad branch"
					case "missing folder":
						delete(source, "path")
					case "null folder":
						source["path"] = nil
					case "typed folder":
						source["path"] = false
					case "public":
						state["public"] = nil
					case "cname":
						delete(state, "cname")
					}
					body, _ := json.Marshal(state)
					c := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
						if r.Method != "GET" || r.URL.Path != "/repos/owner/current/pages" {
							t.Fatal(r.Method, r.URL)
						}
						return actionsResponse(r, 200, string(body)), nil
					}).rest, NormalizationWarning: func(settings.ObservationNormalization) { t.Fatal("pure read warned") }}
					o, err := c.Read(repository.Repository{Owner: "owner", Name: "current"})
					valid := invalid == "" || mode == "workflow" && invalid != "public" && invalid != "cname"
					if (err == nil) != valid {
						t.Fatal(o, err, valid)
					}
					if !valid {
						if _, err := o.Values(); err == nil {
							t.Fatal("incomplete acquired")
						}
						return
					}
					effective, _ := o.Values()
					raw, err := o.RawBuildType()
					if err != nil || raw != mode {
						t.Fatal(raw, err)
					}
					if mode == "workflow" {
						if effective.Source != nil || o.ModeNormalization() != (settings.ObservationNormalization{}) || o.FolderNormalization() != (settings.ObservationNormalization{}) {
							t.Fatal("inactive source acquired", effective)
						}
						return
					}
					wantFolder := folder
					wantRecord := settings.ObservationNormalization{}
					if folder == "/future" {
						wantFolder = "/"
						wantRecord = settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"}
					}
					if effective.BuildType != "legacy" || effective.Source == nil || *effective.Source != (settings.PagesSource{Branch: "release/site", Path: wantFolder}) || o.ModeNormalization() != (settings.ObservationNormalization{Path: settings.PagesModePath, RawValue: mode, Fallback: "deploy_from_a_branch"}) || o.FolderNormalization() != wantRecord {
						t.Fatal("incomplete provenance", effective, o.ModeNormalization(), o.FolderNormalization())
					}
				})
			}
		}
	}
}

func TestPagesModeHTTPUpdateUsesRawEvidenceAndIndependentOwnership(t *testing.T) {
	for _, rawMode := range []string{"FUTURE_MODE", "", " \t"} {
		for _, intent := range []string{"legacy", "workflow", "legacy source", "omitted mode", "source only"} {
			for _, outcome := range []string{"applied", "ignored", "mode drift", "folder drift"} {
				t.Run(fmt.Sprintf("%q/%s/%s", rawMode, intent, outcome), func(t *testing.T) {
					state := map[string]any{"build_type": rawMode, "source": map[string]any{"branch": "main", "path": "/future"}, "cname": nil, "https_enforced": true, "public": true}
					d := settings.PagesDeclaration{}
					wantPayload := map[string]any{}
					switch intent {
					case "legacy", "workflow":
						d.BuildType = actionsUIPtr(intent)
						wantPayload["build_type"] = intent
					case "legacy source":
						d.BuildType = actionsUIPtr("legacy")
						d.Source = &settings.PagesSource{Branch: "main", Path: "/"}
						wantPayload["build_type"] = "legacy"
						wantPayload["source"] = map[string]any{"branch": "main", "path": "/"}
					case "omitted mode":
						d.HTTPSEnforced = actionsUIPtr(false)
						wantPayload["https_enforced"] = false
					case "source only":
						d.Source = &settings.PagesSource{Branch: "main", Path: "/"}
					}
					writes := 0
					var warnings []settings.ObservationNormalization
					c := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
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
							t.Fatal("lifecycle/build mutation", r.Method)
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
						if outcome == "mode drift" {
							state["build_type"] = "OTHER_MODE"
						}
						if outcome == "folder drift" {
							state["source"].(map[string]any)["path"] = "/different"
						}
						return actionsResponse(r, 204, ""), nil
					}).rest, NormalizationWarning: func(n settings.ObservationNormalization) { warnings = append(warnings, n) }}
					wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, d)
					success := intent != "source only" && (outcome == "applied" || intent == "workflow" && outcome == "folder drift")
					wantWrites := 1
					if intent == "source only" {
						wantWrites = 0
					}
					if (err == nil) != success || wrote != success || writes != wantWrites {
						t.Fatal(wrote, err, writes, success)
					}
					var pre interface{ ReadFailedBeforeWrite() }
					if errors.As(err, &pre) != (intent == "source only") {
						t.Fatal("wrong write certainty", err)
					}
					if intent == "source only" && strings.Contains(err.Error(), rawMode) && rawMode != "" && rawMode != " \t" {
						t.Fatal("unmanaged mode disclosed", err)
					}
					var want []settings.ObservationNormalization
					appendWarnings := func(mode, folder string) {
						if d.BuildType != nil && mode != "legacy" && mode != "workflow" {
							want = append(want, settings.ObservationNormalization{Path: settings.PagesModePath, RawValue: mode, Fallback: "deploy_from_a_branch"})
						}
						if d.Source != nil && folder != "/" && folder != "/docs" {
							want = append(want, settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: folder, Fallback: "root"})
						}
					}
					appendWarnings(rawMode, "/future")
					if writes > 0 {
						appendWarnings(state["build_type"].(string), state["source"].(map[string]any)["path"].(string))
					}
					if !reflect.DeepEqual(warnings, want) {
						t.Fatal("warning scope/freshness", warnings, want)
					}
				})
			}
		}
	}
}
