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

const mcpBaseline = `{"review_effort":{"configured":null,"effective":"balanced"},"available_review_efforts":["lite","balanced","max"],"severity_filter":"no_filter","custom_instructions_enabled":true,"mcp_enabled":true}`

func mcpDisabled() string {
	return strings.Replace(mcpBaseline, `"mcp_enabled":true`, `"mcp_enabled":false`, 1)
}

func TestCopilotMCPRead(t *testing.T) {
	type testCase struct {
		name, body    string
		status        int
		enabled, fail bool
	}
	cases := []testCase{
		{"enabled", mcpBaseline, 200, true, false},
		{"disabled", mcpDisabled(), 200, false, false},
		{"unowned preview effort", strings.ReplaceAll(strings.Replace(mcpBaseline, `"configured":null`, `"configured":"max"`, 1), `"effective":"balanced"`, `"effective":"max"`), 200, true, false},
		{"unknown additional fields", strings.Replace(mcpBaseline, `"mcp_enabled":true`, `"mcp_enabled":true,"future":{"anything":1}`, 1), 200, true, false},
		{"denied", `{}`, 403, false, true},
		{"missing endpoint", `{}`, 404, false, true},
		{"redirect", mcpBaseline, 302, false, true},
		{"malformed", `{`, 200, false, true},
		{"trailing", mcpBaseline + ` {}`, 200, false, true},
	}
	fields := map[string]string{
		"mcp": `"mcp_enabled":true`, "instructions": `"custom_instructions_enabled":true`,
		"severity": `"severity_filter":"no_filter"`, "effort": `"review_effort":{"configured":null,"effective":"balanced"}`,
		"available": `"available_review_efforts":["lite","balanced","max"]`,
	}
	for name, field := range fields {
		key := strings.SplitN(field, ":", 2)[0]
		for _, replacement := range []string{"null", "false", `"wrong"`, "12"} {
			// false is valid only for boolean controls.
			if replacement == "false" && (name == "mcp" || name == "instructions") {
				continue
			}
			// String severity remains type-correct but unsupported.
			cases = append(cases, testCase{name + " " + replacement, strings.Replace(mcpBaseline, field, key+":"+replacement, 1), 200, false, true})
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(mcpBaseline), &object); err != nil {
			t.Fatal(err)
		}
		delete(object, strings.Trim(key, `"`))
		missing, _ := json.Marshal(object)
		cases = append(cases, testCase{name + " absent", string(missing), 200, false, true})
	}
	for _, replacement := range []struct{ old, new string }{
		{`"configured":null,`, ``}, {`"configured":null`, `"configured":false`}, {`"configured":null`, `"configured":""`},
		{`,"effective":"balanced"`, ``}, {`"effective":"balanced"`, `"effective":null`}, {`"effective":"balanced"`, `"effective":1`}, {`"effective":"balanced"`, `"effective":""`},
		{`["lite","balanced","max"]`, `["balanced",null]`}, {`["lite","balanced","max"]`, `["balanced",1]`}, {`["lite","balanced","max"]`, `["balanced","balanced"]`}, {`["lite","balanced","max"]`, `[""]`},
	} {
		cases = append(cases, testCase{replacement.new + " invalid nested field", strings.Replace(mcpBaseline, replacement.old, replacement.new, 1), 200, false, true})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &CopilotMCPClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Path != "/repos/owner/current/copilot/code-review/settings" {
					t.Fatal(req.Method, req.URL)
				}
				return labelsResponse(req, tc.status, tc.body), nil
			})).rest}
			observation, err := c.Read(repository.Repository{Owner: "owner", Name: "current"})
			value, valueErr := observation.Value()
			if (err != nil) != tc.fail || calls != 1 || (valueErr != nil) != tc.fail || !tc.fail && value != tc.enabled {
				t.Fatalf("value=%v err=%v valueErr=%v calls=%d", value, err, valueErr, calls)
			}
		})
	}
}

func TestCopilotMCPUpdate(t *testing.T) {
	type testCase struct {
		name, before, after, methods         string
		desired                              bool
		readStatus, writeStatus, afterStatus int
		fail, pre                            bool
	}
	cases := []testCase{
		{"disable", mcpBaseline, mcpDisabled(), "GET PATCH GET", false, 200, 200, 200, false, false},
		{"enable", mcpDisabled(), mcpBaseline, "GET PATCH GET", true, 200, 200, 200, false, false},
		{"matches enabled", mcpBaseline, "", "GET", true, 200, 200, 200, false, false},
		{"matches disabled", mcpDisabled(), "", "GET", false, 200, 200, 200, false, false},
		{"preread denied", mcpBaseline, "", "GET", false, 403, 200, 200, true, true},
		{"preread incomplete", `{"mcp_enabled":true}`, "", "GET", false, 200, 200, 200, true, true},
		{"write rejected", mcpBaseline, "", "GET PATCH", false, 200, 422, 200, true, false},
		{"write unexpected", mcpBaseline, "", "GET PATCH", false, 200, 202, 200, true, false},
		{"readback denied", mcpBaseline, mcpDisabled(), "GET PATCH GET", false, 200, 200, 403, true, false},
		{"readback invalid", mcpBaseline, `{}`, "GET PATCH GET", false, 200, 200, 200, true, false},
		{"readback mismatch", mcpBaseline, mcpBaseline, "GET PATCH GET", false, 200, 200, 200, true, false},
	}
	for _, change := range []struct{ name, old, new string }{
		{"instructions", `"custom_instructions_enabled":true`, `"custom_instructions_enabled":false`},
		{"configured effort", `"configured":null`, `"configured":"balanced"`},
		{"effective effort", `"effective":"balanced"`, `"effective":"lite"`},
		{"availability", `["lite","balanced","max"]`, `["lite","balanced"]`},
		{"severity", `"severity_filter":"no_filter"`, `"severity_filter":"low"`},
	} {
		cases = append(cases, testCase{change.name + " changed", mcpBaseline, strings.Replace(mcpDisabled(), change.old, change.new, 1), "GET PATCH GET", false, 200, 200, 200, true, false})
	}
	maxBefore := strings.ReplaceAll(strings.Replace(mcpBaseline, `"configured":null`, `"configured":"max"`, 1), `"effective":"balanced"`, `"effective":"max"`)
	cases = append(cases, testCase{"preserves unowned preview max", maxBefore, strings.Replace(maxBefore, `"mcp_enabled":true`, `"mcp_enabled":false`, 1), "GET PATCH GET", false, 200, 200, 200, false, false})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			c := &CopilotMCPClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				if req.URL.Path != "/repos/owner/current/copilot/code-review/settings" {
					t.Fatal(req.URL)
				}
				if len(methods) == 1 {
					return labelsResponse(req, tc.readStatus, tc.before), nil
				}
				if req.Method == "PATCH" {
					var payload map[string]any
					if json.NewDecoder(req.Body).Decode(&payload) != nil || !reflect.DeepEqual(payload, map[string]any{"mcp_enabled": tc.desired}) {
						t.Fatalf("payload=%v", payload)
					}
					// Ignore a plausible PATCH response: only the following GET verifies state.
					return labelsResponse(req, tc.writeStatus, mcpDisabled()), nil
				}
				return labelsResponse(req, tc.afterStatus, tc.after), nil
			})).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, tc.desired)
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != tc.fail || errors.As(err, &pre) != tc.pre || strings.Join(methods, " ") != tc.methods || wrote != (!tc.fail && strings.Contains(tc.methods, "PATCH")) {
				t.Fatalf("wrote=%v err=%v methods=%v", wrote, err, methods)
			}
			if tc.fail && !tc.pre && !strings.Contains(err.Error(), "write effects may already have applied") {
				t.Fatal(err)
			}
		})
	}
}

func TestCopilotMCPTransportErrors(t *testing.T) {
	for _, stage := range []string{"GET", "PATCH", "readback"} {
		t.Run(stage, func(t *testing.T) {
			calls := 0
			c := &CopilotMCPClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if stage == "GET" || stage == "PATCH" && req.Method == "PATCH" || stage == "readback" && calls == 3 {
					return nil, errors.New("secret transport diagnostic")
				}
				return labelsResponse(req, 200, mcpBaseline), nil
			})).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, false)
			var pre interface{ ReadFailedBeforeWrite() }
			if wrote || err == nil || errors.As(err, &pre) != (stage == "GET") || strings.Contains(err.Error(), "secret") {
				t.Fatalf("wrote=%v err=%v", wrote, err)
			}
		})
	}
}
