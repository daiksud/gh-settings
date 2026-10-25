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

func propertySchemaFixture() []map[string]any {
	rows := []map[string]any{}
	for name, kind := range map[string]string{"text": "string", "tags": "multi_select", "flag": "true_false", "keep": "string"} {
		rows = append(rows, map[string]any{"property_name": name, "value_type": kind, "source_type": "organization", "required": false, "require_explicit_values": false, "default_value": nil, "values_editable_by": "org_and_repo_actors", "allowed_values": []string{"a", "b"}})
	}
	return rows
}
func TestPropertiesDeclaredPatchFreshSchemaAndReadback(t *testing.T) {
	for _, failure := range []string{"", "regex", "missing-regex", "regex-error", "regex-owner", "regex-name", "regex-type", "fresh-regex", "schema-denied", "value404", "missing-constraint", "default", "required", "not-editable", "wrong-owner", "identity", "fresh-schema", "fresh-values", "write422", "redirect", "readback", "ignored", "unowned", "archived", "pagination"} {
		t.Run(failure, func(t *testing.T) {
			writes, reads, identities := 0, 0, 0
			state := map[string]any{"text": "before", "tags": []string{"a"}, "flag": "true", "keep": "retained"}
			schemas := propertySchemaFixture()
			c := propertyTestClient(t, failure, func(r *http.Request) (*http.Response, error) {
				response := func(v any) *http.Response { b, _ := json.Marshal(v); return actionsResponse(r, 200, string(b)) }
				switch r.URL.Path {
				case "/repos/owner/current":
					if r.Method != "GET" {
						t.Fatal("identity mutation")
					}
					identities++
					id := 123
					if failure == "identity" && identities > 1 {
						id = 456
					}
					kind := "Organization"
					if failure == "wrong-owner" {
						kind = "User"
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"id":%d,"full_name":"owner/current","archived":%t,"owner":{"id":12,"login":"owner","type":%q}}`, id, failure == "archived", kind)), nil
				case "/orgs/owner/properties/schema":
					if r.Method != "GET" {
						t.Fatal("schema mutation")
					}
					reads++
					if failure == "schema-denied" {
						return actionsResponse(r, 403, `{}`), nil
					}
					for _, s := range schemas {
						if s["property_name"] == "text" {
							switch failure {
							case "missing-constraint":
								delete(s, "required")
							case "default":
								s["default_value"] = "fallback"
							case "required":
								s["required"] = true
							case "not-editable":
								s["values_editable_by"] = "org_actors"
							case "fresh-schema":
								if reads > 1 {
									s["default_value"] = "changed"
								}
							}
						}
					}
					return response(schemas), nil
				case "/repos/owner/current/properties/values":
					if r.Method == "GET" {
						if failure == "value404" || failure == "readback" && writes > 0 {
							return actionsResponse(r, 404, `{}`), nil
						}
						if failure == "fresh-values" && reads > 1 {
							state["keep"] = "external"
						}
						rows := []map[string]any{}
						for n, v := range state {
							rows = append(rows, map[string]any{"property_name": n, "value": v})
						}
						resp := response(rows)
						if failure == "pagination" {
							resp.Header.Set("Link", "<https://other.invalid>; rel=next")
						}
						return resp, nil
					}
					if r.Method != "PATCH" {
						t.Fatal("unexpected mutation", r.Method)
					}
					writes++
					var body struct {
						Properties []struct {
							Name  string `json:"property_name"`
							Value any    `json:"value"`
						} `json:"properties"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					payload := map[string]any{}
					for _, p := range body.Properties {
						payload[p.Name] = p.Value
					}
					if !reflect.DeepEqual(payload, map[string]any{"text": nil, "tags": []any{}, "flag": "false"}) {
						t.Fatal("unowned/missing payload", payload)
					}
					if failure == "write422" {
						return actionsResponse(r, 422, `{}`), nil
					}
					if failure == "redirect" {
						resp := actionsResponse(r, 307, `{}`)
						resp.Header.Set("Location", "https://other.invalid")
						return resp, nil
					}
					if failure != "ignored" {
						delete(state, "text")
						state["tags"] = []string{}
						state["flag"] = "false"
					}
					if failure == "unowned" {
						state["keep"] = "changed"
					}
					return actionsResponse(r, 204, ""), nil
				default:
					t.Fatal("unrelated request", r.Method, r.URL)
					return nil, nil
				}
			})
			d := settings.PropertyDeclarations{"text": nil, "tags": []string{}, "flag": "false"}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, d)
			if (err == nil) != (failure == "") || wrote != (failure == "") {
				t.Fatal(wrote, err)
			}
			pre := failure != "" && failure != "write422" && failure != "redirect" && failure != "readback" && failure != "ignored" && failure != "unowned"
			var marker interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &marker) != pre {
				t.Fatal("certainty", err)
			}
			want := 1
			if pre {
				want = 0
			}
			if writes != want {
				t.Fatal("write count", writes)
			}
			if failure == "" {
				wrote, err = c.Update(repository.Repository{Owner: "owner", Name: "current"}, d)
				if wrote || err != nil || writes != 1 {
					t.Fatal("not idempotent", wrote, err, writes)
				}
			}
		})
	}
}
func TestPropertiesReadRejectsMalformedCollections(t *testing.T) {
	for _, raw := range []string{`[] []`, `[] invalid`, `null`, `{}`, `[{"property_name":"text"}]`, `[{"property_name":"text","value":false}]`, `[{"property_name":"text","value":"x"},{"property_name":"text","value":"y"}]`} {
		c := propertyTestClient(t, "", func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/repos/owner/current" {
				return actionsResponse(r, 200, `{"id":123,"full_name":"owner/current","archived":false,"owner":{"id":12,"login":"owner","type":"Organization"}}`), nil
			}
			if r.URL.Path == "/orgs/owner/properties/schema" {
				b, _ := json.Marshal(propertySchemaFixture())
				return actionsResponse(r, 200, string(b)), nil
			}
			return actionsResponse(r, 200, raw), nil
		})
		if _, err := c.Read(repository.Repository{Owner: "owner", Name: "current"}); err == nil {
			t.Fatal("malformed collection accepted", raw)
		}
	}
}

func TestPropertiesSelectedReadOnlyQueriesSelectedStringConstraints(t *testing.T) {
	c := propertyTestClient(t, "regex-error", func(r *http.Request) (*http.Response, error) {
		response := func(v any) *http.Response {
			body, _ := json.Marshal(v)
			return actionsResponse(r, http.StatusOK, string(body))
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			return actionsResponse(r, http.StatusOK, `{"id":123,"full_name":"owner/current","archived":false,"owner":{"id":12,"login":"owner","type":"Organization"}}`), nil
		case "/orgs/owner/properties/schema":
			return response(propertySchemaFixture()), nil
		case "/repos/owner/current/properties/values":
			return response([]map[string]any{
				{"property_name": "text", "value": "unselected"},
				{"property_name": "tags", "value": []string{"a"}},
				{"property_name": "flag", "value": "true"},
				{"property_name": "keep", "value": "unselected"},
			}), nil
		default:
			t.Fatal("unrelated request", r.Method, r.URL)
			return nil, nil
		}
	})
	// The helper would return a partial GraphQL error and fail if either unselected
	// string property were queried. Selecting a non-string property must not
	// require constraints for values that will remain unmanaged.
	o, err := c.ReadSelected(repository.Repository{Owner: "owner", Name: "current"}, []string{"tags"})
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := o.SelectedDeclaration([]string{"tags"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(declaration, settings.PropertyDeclarations{"tags": []string{"a"}}) {
		t.Fatal(declaration)
	}
}

func TestPropertiesSelectedUpdateOnlyQueriesDeclaredStringConstraints(t *testing.T) {
	writes := 0
	tags := []string{"a"}
	c := propertyTestClient(t, "regex-error", func(r *http.Request) (*http.Response, error) {
		response := func(v any) *http.Response {
			body, _ := json.Marshal(v)
			return actionsResponse(r, http.StatusOK, string(body))
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			return actionsResponse(r, http.StatusOK, `{"id":123,"full_name":"owner/current","archived":false,"owner":{"id":12,"login":"owner","type":"Organization"}}`), nil
		case "/orgs/owner/properties/schema":
			return response(propertySchemaFixture()), nil
		case "/repos/owner/current/properties/values":
			if r.Method == http.MethodPatch {
				writes++
				tags = []string{"b"}
				return actionsResponse(r, http.StatusNoContent, ""), nil
			}
			return response([]map[string]any{
				{"property_name": "text", "value": "unselected"},
				{"property_name": "tags", "value": tags},
				{"property_name": "flag", "value": "true"},
				{"property_name": "keep", "value": "unselected"},
			}), nil
		default:
			t.Fatal("unrelated request", r.Method, r.URL)
			return nil, nil
		}
	})
	wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, settings.PropertyDeclarations{"tags": []string{"b"}})
	if err != nil || !wrote || writes != 1 {
		t.Fatalf("wrote=%v writes=%d error=%v", wrote, writes, err)
	}
}

func propertyTestClient(t *testing.T, failure string, handle transportFunc) *PropertiesClient {
	t.Helper()
	reads := 0
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/graphql" {
			return handle(r)
		}
		var query struct {
			Query     string
			Variables map[string]string
		}
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" || strings.Join(strings.Fields(query.Query), " ") != strings.Join(strings.Fields(propertyRegexQuery), " ") || query.Variables["owner"] != "owner" || len(query.Variables) != 2 {
			t.Fatal("unexpected schema query", query)
		}
		reads++
		property := map[string]any{"propertyName": query.Variables["property"], "valueType": "STRING", "regex": nil}
		org := map[string]any{"databaseId": 12, "login": "owner", "repositoryCustomProperty": property}
		response := map[string]any{"data": map[string]any{"organization": org}}
		switch failure {
		case "regex":
			property["regex"] = ".*"
		case "missing-regex":
			delete(property, "regex")
		case "regex-error":
			response["errors"] = []map[string]any{{"message": "partial schema denied"}}
		case "regex-owner":
			org["databaseId"] = 13
		case "regex-name":
			property["propertyName"] = "other"
		case "regex-type":
			property["valueType"] = "URL"
		case "fresh-regex":
			if reads > 1 {
				property["regex"] = "changed"
			}
		}
		body, _ := json.Marshal(response)
		return actionsResponse(r, 200, string(body)), nil
	})
	return &PropertiesClient{rest: actionsClient(t, transport).rest, graphql: prAccessTestClient(t, transport).graphql}
}
