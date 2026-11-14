package github

import (
	"encoding/json"
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryVariableSelectedRead(t *testing.T) {
	for _, family := range []struct {
		name string
		read func(*Client, repository.Repository, string) (settings.VariableObservation, error)
	}{
		{"actions", (*Client).ReadVariable},
		{"agents", (*Client).ReadAgentVariable},
	} {
		t.Run(family.name, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				status        int
				body          string
				present, fail bool
			}{
				{"present", 200, `{"name":"DISPLAY_NAME","value":"日本語"}`, true, false},
				{"empty", 200, `{"name":"DISPLAY_NAME","value":""}`, true, false},
				{"absent", 404, `{"message":"not found"}`, false, false},
				{"denied", 403, `{"message":"denied"}`, false, true},
				{"redirect", 302, `{}`, false, true},
				{"missing value", 200, `{"name":"DISPLAY_NAME"}`, false, true},
				{"null", 200, `{"name":"DISPLAY_NAME","value":null}`, false, true},
				{"wrong type", 200, `{"name":"DISPLAY_NAME","value":true}`, false, true},
				{"wrong name", 200, `{"name":"OTHER","value":"x"}`, false, true},
				{"duplicate name", 200, `{"name":"OTHER","name":"DISPLAY_NAME","value":"PRIVATE_CANARY"}`, false, true},
				{"duplicate name reversed", 200, `{"name":"DISPLAY_NAME","name":"OTHER","value":"PRIVATE_CANARY"}`, false, true},
				{"duplicate value", 200, `{"name":"DISPLAY_NAME","value":"other","value":"PRIVATE_CANARY"}`, false, true},
				{"duplicate value reversed", 200, `{"name":"DISPLAY_NAME","value":"PRIVATE_CANARY","value":"other"}`, false, true},
				{"name alias", 200, `{"Name":"OTHER","name":"DISPLAY_NAME","value":"PRIVATE_CANARY"}`, false, true},
				{"name alias reversed", 200, `{"name":"DISPLAY_NAME","Name":"OTHER","value":"PRIVATE_CANARY"}`, false, true},
				{"value alias", 200, `{"name":"DISPLAY_NAME","Value":"other","value":"PRIVATE_CANARY"}`, false, true},
				{"value alias reversed", 200, `{"name":"DISPLAY_NAME","value":"PRIVATE_CANARY","Value":"other"}`, false, true},
				{"trailing", 200, `{"name":"DISPLAY_NAME","value":"PRIVATE_CANARY"} {}`, false, true},
				{"oversized", 200, `{"name":"DISPLAY_NAME","value":"PRIVATE_CANARY","padding":"` + strings.Repeat("x", 1<<20) + `"}`, false, true},
				{"invalid utf8", 200, "{\"name\":\"DISPLAY_NAME\",\"value\":\"\xff\"}", false, true},
				{"unpaired high surrogate", 200, `{"name":"DISPLAY_NAME","value":"\ud800"}`, false, true},
				{"unpaired low surrogate", 200, `{"name":"DISPLAY_NAME","value":"\udc00"}`, false, true},
				{"invalid surrogate pair", 200, `{"name":"DISPLAY_NAME","value":"\ud800\u0041"}`, false, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					calls := 0
					client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						if req.Method != "GET" || req.URL.Path != "/repos/owner/current/"+family.name+"/variables/DISPLAY_NAME" {
							t.Fatalf("unexpected request %s %s", req.Method, req.URL)
						}
						return labelsResponse(req, tc.status, tc.body), nil
					}))
					observed, err := family.read(client, repository.Repository{Owner: "owner", Name: "current"}, "display_name")
					value, valueErr := observed.Value()
					if (err != nil) != tc.fail || !tc.fail && (valueErr != nil || (value != nil) != tc.present) || tc.fail && valueErr == nil || calls != 1 {
						t.Fatalf("observed=%#v err=%v valueErr=%v calls=%d", observed, err, valueErr, calls)
					}
					if err != nil && strings.Contains(err.Error(), "PRIVATE_CANARY") {
						t.Fatal("variable value leaked in error")
					}
					if !tc.fail && tc.present {
						want := ""
						if tc.name == "present" {
							want = "日本語"
						}
						if value.Name != "DISPLAY_NAME" || value.Value != want {
							t.Fatalf("value changed: %#v", value)
						}
					}
				})
			}
		})
	}
}

func TestRepositoryVariableFreshUpsert(t *testing.T) {
	for _, family := range []struct {
		name   string
		upsert func(*Client, repository.Repository, settings.RepositoryVariable) (bool, error)
	}{
		{"actions", (*Client).UpsertVariable},
		{"agents", (*Client).UpsertAgentVariable},
	} {
		t.Run(family.name, func(t *testing.T) {
			for _, tc := range []struct {
				name         string
				firstStatus  int
				first, after string
				writeStatus  int
				wantMethods  string
				fail, pre    bool
			}{
				{"create", 404, `{}`, `{"name":"DISPLAY_NAME","value":"create-value"}`, 201, "GET POST GET", false, false},
				{"update", 200, `{"name":"DISPLAY_NAME","value":"old"}`, `{"name":"DISPLAY_NAME","value":"日本語-v2"}`, 204, "GET PATCH GET", false, false},
				{"matching", 200, `{"name":"DISPLAY_NAME","value":"create-value"}`, ``, 204, "GET", false, false},
				{"preread denied", 403, `{}`, ``, 204, "GET", true, true},
				{"preread ambiguous", 200, `{"name":"DISPLAY_NAME","value":"old","value":"create-value"}`, ``, 204, "GET", true, true},
				{"preread invalid utf8", 200, "{\"name\":\"DISPLAY_NAME\",\"value\":\"\xff\"}", ``, 204, "GET", true, true},
				{"preread unpaired surrogate", 200, `{"name":"DISPLAY_NAME","value":"\ud800"}`, ``, 204, "GET", true, true},
				{"write denied", 404, `{}`, ``, 403, "GET POST", true, false},
				{"wrong status", 404, `{}`, ``, 200, "GET POST", true, false},
				{"readback denied", 404, `{}`, `DENIED`, 201, "GET POST GET", true, false},
				{"readback absent", 404, `{}`, `ABSENT`, 201, "GET POST GET", true, false},
				{"readback mismatch", 404, `{}`, `{"name":"DISPLAY_NAME","value":"old"}`, 201, "GET POST GET", true, false},
				{"readback ambiguous", 404, `{}`, `{"name":"DISPLAY_NAME","value":"old","Value":"create-value"}`, 201, "GET POST GET", true, false},
				{"readback invalid utf8", 404, `{}`, "{\"name\":\"DISPLAY_NAME\",\"value\":\"\xff\"}", 201, "GET POST GET", true, false},
				{"readback unpaired surrogate", 404, `{}`, `{"name":"DISPLAY_NAME","value":"\ud800"}`, 201, "GET POST GET", true, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var methods []string
					desired := "create-value"
					if tc.name == "update" {
						desired = "日本語-v2"
					}
					if strings.Contains(tc.name, "invalid utf8") || strings.Contains(tc.name, "unpaired surrogate") {
						desired = "�"
					}
					client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
						methods = append(methods, req.Method)
						wantPath := "/repos/owner/current/" + family.name + "/variables/DISPLAY_NAME"
						if req.Method == "POST" {
							wantPath = "/repos/owner/current/" + family.name + "/variables"
						}
						if req.URL.Path != wantPath {
							t.Fatalf("path=%s", req.URL.Path)
						}
						if len(methods) == 1 {
							return labelsResponse(req, tc.firstStatus, tc.first), nil
						}
						if req.Method == "POST" || req.Method == "PATCH" {
							var body map[string]string
							if json.NewDecoder(req.Body).Decode(&body) != nil || body["value"] != desired {
								t.Fatal("wrong desired value")
							}
							want := map[string]string{"value": desired}
							if req.Method == "POST" {
								want["name"] = "DISPLAY_NAME"
							}
							if !reflect.DeepEqual(body, want) {
								t.Fatalf("body=%v", body)
							}
							return labelsResponse(req, tc.writeStatus, `{}`), nil
						}
						if tc.after == "DENIED" {
							return labelsResponse(req, 403, `{}`), nil
						}
						if tc.after == "ABSENT" {
							return labelsResponse(req, 404, `{}`), nil
						}
						return labelsResponse(req, 200, tc.after), nil
					}))
					wrote, err := family.upsert(client, repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryVariable{Name: "display_name", Value: desired})
					var pre interface{ ReadFailedBeforeWrite() }
					if (err != nil) != tc.fail || errors.As(err, &pre) != tc.pre || wrote != (!tc.fail && tc.name != "matching") || strings.Join(methods, " ") != tc.wantMethods {
						t.Fatalf("wrote=%v err=%v methods=%v", wrote, err, methods)
					}
				})
			}
		})
	}
}

func TestRepositoryVariableEmptyUpsertIsLocalFailure(t *testing.T) {
	for _, family := range []struct {
		name   string
		upsert func(*Client, repository.Repository, settings.RepositoryVariable) (bool, error)
	}{
		{"actions", (*Client).UpsertVariable},
		{"agents", (*Client).UpsertAgentVariable},
	} {
		t.Run(family.name, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { calls++; return labelsResponse(req, 404, `{}`), nil }))
			_, err := family.upsert(client, repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryVariable{Name: "DISPLAY_NAME", Value: ""})
			var pre interface{ ReadFailedBeforeWrite() }
			if err == nil || !errors.As(err, &pre) || calls != 0 {
				t.Fatal("empty value reached API", err, calls)
			}
		})
	}
}

func TestVariableReadPreservesValidUnicode(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"replacement character", `{"name":"DISPLAY_NAME","value":"�"}`, "�"},
		{"surrogate pair", `{"name":"DISPLAY_NAME","value":"\ud83d\ude00"}`, "😀"},
		{"literal backslash", `{"name":"DISPLAY_NAME","value":"\\ud800"}`, `\ud800`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, tc.body), nil }))
			observed, err := client.ReadVariable(repository.Repository{Owner: "owner", Name: "current"}, "DISPLAY_NAME")
			if err != nil {
				t.Fatal(err)
			}
			value, err := observed.Value()
			if err != nil || value == nil || value.Value != tc.want {
				t.Fatalf("invalid value: %v", err)
			}
		})
	}
}
