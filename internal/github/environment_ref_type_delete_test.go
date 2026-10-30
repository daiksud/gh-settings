package github

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentPatternDeletionConfirmsExactRawIdentityAndRetainsEarlierCreation(t *testing.T) {
	for _, rawType := range []string{"future", "", " \t"} {
		for _, mode := range []string{"replace", "canonical companion", "clear", "omitted", "type changed", "fallback type", "name changed", "id changed", "missing type", "null type", "denied", "POST unknown", "POST denied", "DELETE denied"} {
			t.Run(rawType+"/"+mode, func(t *testing.T) {
				current := []settings.EnvironmentPatternState{{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "Release/*"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: rawType, Fallback: "branch"}}}
				wanted := []settings.EnvironmentPattern{{Type: "branch", Name: "Release/*"}}
				if mode == "canonical companion" {
					current = append(current, settings.EnvironmentPatternState{ID: 8, Pattern: wanted[0]})
				}
				if mode == "clear" {
					wanted = []settings.EnvironmentPattern{}
				}
				desired := &wanted
				if mode == "omitted" {
					desired = nil
				}
				var calls []string
				created, deleted := false, false
				base := "/repos/owner/current/environments/production/deployment-branch-policies"
				client := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					calls = append(calls, r.Method+" "+r.URL.Path)
					respond := func(value map[string]any) (*http.Response, error) {
						data, _ := json.Marshal(value)
						return actionsResponse(r, 200, string(data)), nil
					}
					switch {
					case r.Method == "POST" && r.URL.Path == base:
						body, _ := io.ReadAll(r.Body)
						var fields map[string]any
						if err := json.Unmarshal(body, &fields); err != nil || !reflect.DeepEqual(fields, map[string]any{"type": "branch", "name": "Release/*"}) {
							t.Fatal("canonical payload differs from literal intent", string(body), err)
						}
						if mode == "POST denied" {
							return actionsResponse(r, 403, `{}`), nil
						}
						created = true
						kind := "branch"
						if mode == "POST unknown" {
							kind = rawType
						}
						return respond(map[string]any{"id": 9, "name": "Release/*", "type": kind})
					case r.Method == "GET" && r.URL.Path == base+"/7":
						if mode == "denied" {
							return actionsResponse(r, 403, `{}`), nil
						}
						value := map[string]any{"id": 7, "name": "Release/*", "type": rawType}
						switch mode {
						case "type changed":
							value["type"] = "other"
						case "fallback type":
							value["type"] = "branch"
						case "name changed":
							value["name"] = "release/*"
						case "id changed":
							value["id"] = 8
						case "missing type":
							delete(value, "type")
						case "null type":
							value["type"] = nil
						}
						return respond(value)
					case r.Method == "DELETE" && r.URL.Path == base+"/7":
						if mode == "DELETE denied" {
							return actionsResponse(r, 403, `{}`), nil
						}
						deleted = true
						return actionsResponse(r, 204, ""), nil
					default:
						t.Fatal("unexpected request", r.Method, r.URL)
						return nil, nil
					}
				}).rest}
				err := client.applyPatterns(repository.Repository{Owner: "owner", Name: "current"}, "production", current, desired)
				good := mode == "replace" || mode == "canonical companion" || mode == "clear" || mode == "omitted"
				if (err == nil) != good || deleted != (good && mode != "omitted") {
					t.Fatal("raw identity not confirmed safely", calls, created, deleted, err)
				}
				var want []string
				if mode != "canonical companion" && mode != "clear" && mode != "omitted" {
					want = append(want, "POST "+base)
				}
				if mode != "omitted" && !strings.HasPrefix(mode, "POST ") {
					want = append(want, "GET "+base+"/7")
				}
				if good && mode != "omitted" || mode == "DELETE denied" {
					want = append(want, "DELETE "+base+"/7")
				}
				if !reflect.DeepEqual(calls, want) || created != (len(want) > 0 && strings.HasPrefix(want[0], "POST ") && mode != "POST denied") {
					t.Fatal("order or earlier side effects changed", calls, want, created, err)
				}
			})
		}
	}
}
