package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestCopilotInstructionsRead(t *testing.T) {
	for _, tc := range []struct {
		name          string
		status        int
		body          string
		enabled, fail bool
	}{
		{"on", 200, `{"custom_instructions_enabled":true,"mcp_enabled":false}`, true, false},
		{"off", 200, `{"custom_instructions_enabled":false,"review_effort":{"effective":"balanced"}}`, false, false},
		{"denied", 403, `{"message":"denied"}`, false, true},
		{"not found", 404, `{}`, false, true},
		{"redirect", 302, `{}`, false, true},
		{"missing", 200, `{}`, false, true},
		{"null", 200, `{"custom_instructions_enabled":null}`, false, true},
		{"string", 200, `{"custom_instructions_enabled":"false"}`, false, true},
		{"trailing", 200, `{"custom_instructions_enabled":true} {}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &CopilotInstructionsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Path != "/repos/owner/current/copilot/code-review/settings" {
					t.Fatalf("unexpected request %s %s", req.Method, req.URL)
				}
				return labelsResponse(req, tc.status, tc.body), nil
			})).rest}
			observed, err := c.Read(repository.Repository{Owner: "owner", Name: "current"})
			enabled, valueErr := observed.Value()
			if (err != nil) != tc.fail || calls != 1 || tc.fail && valueErr == nil || !tc.fail && (valueErr != nil || enabled != tc.enabled) {
				t.Fatalf("enabled=%v err=%v valueErr=%v calls=%d", enabled, err, valueErr, calls)
			}
		})
	}
}

func TestCopilotInstructionsUpdate(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		before, desired                      bool
		readStatus, writeStatus, afterStatus int
		afterBody, methods                   string
		fail, pre                            bool
	}{
		{"disable", true, false, 200, 200, 200, `{"custom_instructions_enabled":false}`, "GET PATCH GET", false, false},
		{"enable", false, true, 200, 200, 200, `{"custom_instructions_enabled":true}`, "GET PATCH GET", false, false},
		{"matching off", false, false, 200, 200, 200, ``, "GET", false, false},
		{"matching on", true, true, 200, 200, 200, ``, "GET", false, false},
		{"preread denied", true, false, 403, 200, 200, ``, "GET", true, true},
		{"write rejected", true, false, 200, 422, 200, ``, "GET PATCH", true, false},
		{"unexpected status", true, false, 200, 202, 200, ``, "GET PATCH", true, false},
		{"readback denied", true, false, 200, 200, 403, `{}`, "GET PATCH GET", true, false},
		{"readback missing", true, false, 200, 200, 200, `{}`, "GET PATCH GET", true, false},
		{"readback mismatch", true, false, 200, 200, 200, `{"custom_instructions_enabled":true}`, "GET PATCH GET", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			c := &CopilotInstructionsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				if req.URL.Path != "/repos/owner/current/copilot/code-review/settings" {
					t.Fatalf("unexpected path %s", req.URL)
				}
				if len(methods) == 1 {
					return labelsResponse(req, tc.readStatus, fmt.Sprintf(`{"custom_instructions_enabled":%t}`, tc.before)), nil
				}
				if req.Method == "PATCH" {
					var body map[string]any
					if json.NewDecoder(req.Body).Decode(&body) != nil || !reflect.DeepEqual(body, map[string]any{"custom_instructions_enabled": tc.desired}) {
						t.Fatalf("wrong payload %v", body)
					}
					return labelsResponse(req, tc.writeStatus, fmt.Sprintf(`{"custom_instructions_enabled":%t}`, tc.desired)), nil
				}
				if req.Method != "GET" {
					t.Fatalf("unexpected method %s", req.Method)
				}
				return labelsResponse(req, tc.afterStatus, tc.afterBody), nil
			})).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, tc.desired)
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != tc.fail || errors.As(err, &pre) != tc.pre || wrote != (!tc.fail && tc.before != tc.desired) || strings.Join(methods, " ") != tc.methods {
				t.Fatalf("wrote=%v err=%v methods=%v", wrote, err, methods)
			}
		})
	}
}
