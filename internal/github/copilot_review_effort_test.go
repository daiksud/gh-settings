package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

const effortBaseline = `{"review_effort":{"configured":null,"effective":"balanced"},"available_review_efforts":["lite","balanced","max"],"severity_filter":"no_filter","custom_instructions_enabled":true,"mcp_enabled":true}`

func effortBody(configured string) string {
	return strings.Replace(effortBaseline, `"configured":null`, `"configured":`+configured, 1)
}

func TestCopilotReviewEffortRead(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		fail       bool
	}{
		{"default", effortBaseline, 200, false},
		{"default effective lite", strings.Replace(effortBaseline, `"effective":"balanced"`, `"effective":"lite"`, 1), 200, false},
		{"future available choice", strings.Replace(effortBaseline, `["lite","balanced","max"]`, `["lite","balanced","max","future"]`, 1), 200, false},
		{"explicit", effortBody(`"balanced"`), 200, false},
		{"max configured", effortBody(`"max"`), 200, false},
		{"max effective", strings.Replace(effortBaseline, `"effective":"balanced"`, `"effective":"max"`, 1), 200, false},
		{"configured absent", strings.Replace(effortBaseline, `"configured":null,`, "", 1), 200, true},
		{"configured bool", effortBody(`false`), 200, true},
		{"effective null", strings.Replace(effortBaseline, `"effective":"balanced"`, `"effective":null`, 1), 200, true},
		{"available absent", strings.Replace(effortBaseline, `"available_review_efforts":["lite","balanced","max"],`, "", 1), 200, true},
		{"available null", strings.Replace(effortBaseline, `["lite","balanced","max"]`, `null`, 1), 200, true},
		{"available null member", strings.Replace(effortBaseline, `["lite","balanced","max"]`, `["balanced",null]`, 1), 200, true},
		{"available number", strings.Replace(effortBaseline, `["lite","balanced","max"]`, `["balanced",1]`, 1), 200, true},
		{"missing sibling", strings.Replace(effortBaseline, `,"mcp_enabled":true`, "", 1), 200, true},
		{"null sibling", strings.Replace(effortBaseline, `"custom_instructions_enabled":true`, `"custom_instructions_enabled":null`, 1), 200, true},
		{"invalid severity", strings.Replace(effortBaseline, `"no_filter"`, `false`, 1), 200, true},
		{"trailing", effortBaseline + ` {}`, 200, true},
		{"denied", `{}`, 403, true},
		{"redirect", `{}`, 302, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &CopilotReviewEffortClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Path != "/repos/owner/current/copilot/code-review/settings" {
					t.Fatal(req.Method, req.URL)
				}
				return labelsResponse(req, tc.status, tc.body), nil
			})).rest}
			observed, err := c.Read(repository.Repository{Owner: "owner", Name: "current"})
			_, valueErr := observed.Value()
			if (err != nil) != tc.fail || calls != 1 || (valueErr != nil) != tc.fail {
				t.Fatalf("err=%v valueErr=%v calls=%d", err, valueErr, calls)
			}
		})
	}
}

func TestCopilotReviewEffortUpdate(t *testing.T) {
	for _, tc := range []struct {
		name, desired, before, after, methods, payload string
		readStatus, writeStatus                        int
		fail, pre                                      bool
	}{
		{"max", "max", effortBaseline, effortBody(`"max"`), "GET PATCH GET", "max", 200, 200, false, false},
		{"matching max", "max", effortBody(`"max"`), "", "GET", "", 200, 200, false, false},
		{"unavailable max", "max", strings.Replace(effortBaseline, `["lite","balanced","max"]`, `["lite","balanced"]`, 1), "", "GET", "", 200, 200, true, true},
		{"max readback mismatch", "max", effortBaseline, effortBaseline, "GET PATCH GET", "max", 200, 200, true, false},
		{"max sibling changed", "max", effortBaseline, strings.Replace(effortBody(`"max"`), `"mcp_enabled":true`, `"mcp_enabled":false`, 1), "GET PATCH GET", "max", 200, 200, true, false},
		{"lite", "lite", effortBaseline, effortBody(`"lite"`), "GET PATCH GET", "lite", 200, 200, false, false},
		{"explicit balanced despite effective match", "balanced", effortBaseline, effortBody(`"balanced"`), "GET PATCH GET", "balanced", 200, 200, false, false},
		{"default", "default", effortBody(`"lite"`), effortBaseline, "GET PATCH GET", "inherit", 200, 200, false, false},
		{"ambiguous default matches", "default", effortBaseline, "", "GET", "", 200, 200, false, false},
		{"explicit matches", "balanced", effortBody(`"balanced"`), "", "GET", "", 200, 200, false, false},
		{"unsupported desired", "future", effortBaseline, "", "", "", 200, 200, true, true},
		{"denied", "lite", effortBaseline, "", "GET", "", 403, 200, true, true},
		{"unsupported observation", "lite", effortBody(`"future"`), "", "GET", "", 200, 200, true, true},
		{"fresh unavailable", "lite", strings.Replace(effortBaseline, `["lite","balanced","max"]`, `["balanced"]`, 1), "", "GET", "", 200, 200, true, true},
		{"rejected", "lite", effortBaseline, "", "GET PATCH", "lite", 200, 422, true, false},
		{"unexpected write status", "lite", effortBaseline, "", "GET PATCH", "lite", 200, 202, true, false},
		{"readback invalid", "lite", effortBaseline, `{}`, "GET PATCH GET", "lite", 200, 200, true, false},
		{"readback mismatch", "lite", effortBaseline, effortBaseline, "GET PATCH GET", "lite", 200, 200, true, false},
		{"sibling changed", "lite", effortBaseline, strings.Replace(effortBody(`"lite"`), `"mcp_enabled":true`, `"mcp_enabled":false`, 1), "GET PATCH GET", "lite", 200, 200, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			c := &CopilotReviewEffortClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				if len(methods) == 1 {
					return labelsResponse(req, tc.readStatus, tc.before), nil
				}
				if req.Method == "PATCH" {
					var payload map[string]any
					if json.NewDecoder(req.Body).Decode(&payload) != nil || !reflect.DeepEqual(payload, map[string]any{"review_effort": tc.payload}) {
						t.Fatal(payload)
					}
					// A plausible PATCH body must never substitute for the independent GET.
					return labelsResponse(req, tc.writeStatus, effortBody(`"`+tc.desired+`"`)), nil
				}
				return labelsResponse(req, 200, tc.after), nil
			})).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, tc.desired)
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != tc.fail || errors.As(err, &pre) != tc.pre || strings.Join(methods, " ") != tc.methods || wrote != (!tc.fail && strings.Contains(tc.methods, "PATCH")) {
				t.Fatalf("wrote=%v err=%v methods=%v", wrote, err, methods)
			}
		})
	}
}
