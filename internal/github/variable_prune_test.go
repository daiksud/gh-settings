package github

import (
	"encoding/json"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"strings"
	"testing"
)

func TestVariablePruneInventoryCompletesThirtyItemPages(t *testing.T) {
	for _, family := range []string{"actions", "agents"} {
		for _, mode := range []string{"success", "page denied", "short page", "count changed", "duplicate", "missing value", "missing count", "null list", "trailing", "excess"} {
			t.Run(family+"/"+mode, func(t *testing.T) {
				client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
					if r.Method != "GET" || r.URL.Path != "/repos/owner/current/"+family+"/variables" || r.URL.Query().Get("per_page") != "30" {
						t.Fatal(r.Method, r.URL)
					}
					page := r.URL.Query().Get("page")
					total := 31
					values := []map[string]any{}
					if page == "1" {
						for i := 0; i < 30; i++ {
							values = append(values, map[string]any{"name": fmt.Sprintf("VAR_%02d", i), "value": "value"})
						}
					} else {
						values = append(values, map[string]any{"name": "VAR_30", "value": ""})
					}
					switch mode {
					case "page denied":
						if page == "2" {
							return labelsResponse(r, 403, `{}`), nil
						}
					case "short page":
						if page == "1" {
							values = values[:29]
						}
					case "count changed":
						if page == "2" {
							total = 32
						}
					case "duplicate":
						if page == "2" {
							values[0]["name"] = "var_00"
						}
					case "missing value":
						delete(values[0], "value")
					case "missing count":
						return labelsResponse(r, 200, `{"variables":[]}`), nil
					case "null list":
						return labelsResponse(r, 200, `{"total_count":0,"variables":null}`), nil
					case "trailing":
						return labelsResponse(r, 200, `{"total_count":0,"variables":[]} {}`), nil
					case "excess":
						total = 0
					}
					body, _ := json.Marshal(map[string]any{"total_count": total, "variables": values})
					return labelsResponse(r, 200, string(body)), nil
				}))
				repo := repository.Repository{Owner: "owner", Name: "current"}
				var observed settings.VariableInventory
				var err error
				if family == "actions" {
					observed, err = client.ReadVariables(repo)
				} else {
					observed, err = client.ReadAgentVariables(repo)
				}
				values, valueErr := observed.Values()
				if (err == nil) != (mode == "success") || mode == "success" && (valueErr != nil || len(values) != 31) || mode != "success" && valueErr == nil {
					t.Fatal(mode, len(values), err, valueErr)
				}
			})
		}
	}
}
func TestVariablePruneDeleteExactValueAndInventoryReadback(t *testing.T) {
	for _, family := range []string{"actions", "agents"} {
		for _, mode := range []string{"success", "changed", "missing", "denied", "retained", "readback denied", "bad status"} {
			t.Run(family+"/"+mode, func(t *testing.T) {
				deletes := 0
				client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
					if !strings.HasPrefix(r.URL.Path, "/repos/owner/current/"+family+"/variables") {
						t.Fatal(r.URL)
					}
					if r.Method == "DELETE" {
						deletes++
						if mode == "denied" {
							return labelsResponse(r, 403, `{}`), nil
						}
						if mode == "bad status" {
							return labelsResponse(r, 200, `{}`), nil
						}
						return labelsResponse(r, 204, ""), nil
					}
					if strings.HasSuffix(r.URL.Path, "/OLD") {
						if mode == "missing" {
							return labelsResponse(r, 404, `{}`), nil
						}
						value := "before"
						if mode == "changed" {
							value = "new"
						}
						return labelsResponse(r, 200, fmt.Sprintf(`{"name":"OLD","value":%q}`, value)), nil
					}
					if mode == "readback denied" {
						return labelsResponse(r, 403, `{}`), nil
					}
					if mode == "retained" {
						return labelsResponse(r, 200, `{"total_count":1,"variables":[{"name":"OLD","value":"before"}]}`), nil
					}
					return labelsResponse(r, 200, `{"total_count":0,"variables":[]}`), nil
				}))
				change, _ := settings.NewVariableDelete(settings.RepositoryVariable{Name: "OLD", Value: "before"})
				repo := repository.Repository{Owner: "owner", Name: "current"}
				var err error
				if family == "actions" {
					err = client.DeleteVariable(repo, change)
				} else {
					err = client.DeleteAgentVariable(repo, change)
				}
				if (err == nil) != (mode == "success") || ((mode == "changed" || mode == "missing") && deletes != 0) {
					t.Fatal(mode, deletes, err)
				}
			})
		}
	}
}
func TestEnvironmentVariablePruneBindsParentIdentity(t *testing.T) {
	for _, mode := range []string{"success", "changed during list", "changed before delete", "changed after delete"} {
		t.Run(mode, func(t *testing.T) {
			parentReads := 0
			deletes := 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.EscapedPath() == "/repos/owner/current/environments/prod%2Fblue" {
					parentReads++
					id := 7
					if mode == "changed during list" && parentReads > 1 || mode == "changed before delete" && parentReads > 2 || mode == "changed after delete" && parentReads > 3 {
						id = 8
					}
					return labelsResponse(r, 200, fmt.Sprintf(`{"id":%d,"name":"prod/blue"}`, id)), nil
				}
				if !strings.Contains(r.URL.EscapedPath(), "/environments/prod%2Fblue/variables") {
					t.Fatal(r.Method, r.URL)
				}
				if r.Method == "DELETE" {
					deletes++
					return labelsResponse(r, 204, ""), nil
				}
				if strings.HasSuffix(r.URL.Path, "/OLD") {
					return labelsResponse(r, 200, `{"name":"OLD","value":"before"}`), nil
				}
				if deletes > 0 {
					return labelsResponse(r, 200, `{"total_count":0,"variables":[]}`), nil
				}
				return labelsResponse(r, 200, `{"total_count":1,"variables":[{"name":"OLD","value":"before"}]}`), nil
			}))
			repo := repository.Repository{Owner: "owner", Name: "current"}
			inventory, err := client.ReadEnvironmentVariables(repo, "prod/blue")
			if mode == "changed during list" {
				if err == nil || deletes != 0 {
					t.Fatal(err, deletes)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			id, _ := inventory.EnvironmentID("prod/blue")
			change, _ := settings.NewVariableDelete(settings.RepositoryVariable{Name: "OLD", Value: "before"})
			err = client.DeleteEnvironmentVariable(repo, "prod/blue", id, change)
			if (err == nil) != (mode == "success") || mode == "changed before delete" && deletes != 0 {
				t.Fatal(mode, err, deletes)
			}
		})
	}
}
