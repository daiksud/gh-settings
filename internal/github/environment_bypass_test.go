package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentAdminBypassPayloadAndReadback(t *testing.T) {
	for _, mode := range []string{"false", "true", "omitted", "ignored", "missing", "null", "string", "write422"} {
		t.Run(mode, func(t *testing.T) {
			wanted := mode == "true"
			var value any = !wanted
			writes := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				respond := func() (*http.Response, error) {
					state := map[string]any{"id": 1, "name": "production", "protection_rules": []any{}, "deployment_branch_policy": nil, "can_admins_bypass": value}
					if mode == "missing" {
						delete(state, "can_admins_bypass")
					}
					if mode == "null" {
						state["can_admins_bypass"] = nil
					}
					if mode == "string" {
						state["can_admins_bypass"] = "false"
					}
					data, _ := json.Marshal(state)
					return actionsResponse(r, 200, string(data)), nil
				}
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
					case "/repos/owner/current/environments":
						return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case "/repos/owner/current/environments/production":
						return respond()
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						return actionsResponse(r, 200, customEnvironmentEmpty), nil
					}
				}
				if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/environments/production" {
					t.Fatal(r.Method, r.URL)
				}
				writes++
				var payload map[string]any
				json.NewDecoder(r.Body).Decode(&payload)
				want := map[string]any{"can_admins_bypass": wanted}
				if mode == "omitted" {
					want = map[string]any{}
				}
				if !reflect.DeepEqual(payload, want) {
					t.Fatal("bypass default or missing native false", payload, want)
				}
				if mode == "write422" {
					return actionsResponse(r, 422, `{"message":"rejected"}`), nil
				}
				if mode != "ignored" && mode != "omitted" {
					value = wanted
				}
				return respond()
			}).rest}
			request := settings.EnvironmentDeclaration{Name: "production", CanAdminsBypass: &wanted}
			if mode == "omitted" {
				request.CanAdminsBypass = nil
			}
			_, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, request)
			bad := mode == "ignored" || mode == "missing" || mode == "null" || mode == "string" || mode == "write422"
			if (err != nil) != bad {
				t.Fatal(writes, err)
			}
			beforeWrite := mode == "missing" || mode == "null" || mode == "string"
			if beforeWrite && writes != 0 || !beforeWrite && writes != 1 {
				t.Fatal("unexpected mutations", writes)
			}
		})
	}
}
