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

const exemptionAlice = `{"id":1,"login":"Alice","type":"User"}`
const exemptionBob = `{"id":2,"login":"bob","type":"User"}`
const exemptionCarol = `{"id":3,"login":"carol","type":"User"}`

func TestPullRequestExemptionsRead(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		link       string
		valid      bool
	}{
		{"empty", `[]`, 200, "", true},
		{"users", `[` + exemptionBob + `,` + exemptionAlice + `]`, 200, "", true},
		{"null", `null`, 200, "", false},
		{"object", `{}`, 200, "", false},
		{"missing", `[{"id":1,"login":"alice"}]`, 200, "", false},
		{"null member", `[null]`, 200, "", false},
		{"wrong type", `[{"id":"1","login":"alice","type":"User"}]`, 200, "", false},
		{"unknown actor", `[{"id":1,"login":"alice","type":"Bot"}]`, 200, "", false},
		{"invalid id", `[{"id":0,"login":"alice","type":"User"}]`, 200, "", false},
		{"invalid login", `[{"id":1,"login":"bad/login","type":"User"}]`, 200, "", false},
		{"duplicate id", `[` + exemptionAlice + `,{"id":1,"login":"other","type":"User"}]`, 200, "", false},
		{"duplicate name", `[` + exemptionAlice + `,{"id":2,"login":"alice","type":"User"}]`, 200, "", false},
		{"trailing", `[] {}`, 200, "", false},
		{"denied", `{"message":"private server content"}`, 403, "", false},
		{"absent", `{}`, 404, "", false},
		{"redirect", `{}`, 302, "", false},
		{"pagination", `[]`, 200, `<https://api.github.com/next>; rel="next"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &PullRequestLimitsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Path != "/repos/owner/current/interaction-limits/pulls/bypass-list" || req.URL.RawQuery != "" {
					t.Fatal(req.Method, req.URL)
				}
				response := labelsResponse(req, tc.status, tc.body)
				if response.Header == nil {
					response.Header = http.Header{}
				}
				response.Header.Set("Link", tc.link)
				return response, nil
			})).rest}
			observation, err := c.ReadExemptions(repository.Repository{Owner: "owner", Name: "current"})
			logins, valueErr := observation.Logins()
			if (err == nil) != tc.valid || (valueErr == nil) != tc.valid || calls != 1 {
				t.Fatalf("logins=%v err=%v valueErr=%v calls=%d", logins, err, valueErr, calls)
			}
			if err != nil && strings.Contains(err.Error(), "private server content") {
				t.Fatal("leaked response")
			}
			if tc.name == "users" && !reflect.DeepEqual(logins, []string{"alice", "bob"}) {
				t.Fatal(logins)
			}
		})
	}
}

func TestPullRequestExemptionsUpdate(t *testing.T) {
	for _, tc := range []struct {
		name             string
		desired          []string
		reads            []string
		methods          string
		failureAt        int
		failStatus       int
		fail, pre, wrote bool
	}{
		{name: "add", desired: []string{"alice", "bob"}, reads: []string{`[` + exemptionAlice + `]`, `[` + exemptionBob + `,` + exemptionAlice + `]`}, methods: "GET PUT GET", wrote: true},
		{name: "remove", desired: []string{"alice"}, reads: []string{`[` + exemptionAlice + `,` + exemptionBob + `]`, `[` + exemptionAlice + `]`}, methods: "GET DELETE GET", wrote: true},
		{name: "replace", desired: []string{"alice", "carol"}, reads: []string{`[` + exemptionAlice + `,` + exemptionBob + `]`, `[` + exemptionAlice + `]`, `[` + exemptionCarol + `,` + exemptionAlice + `]`}, methods: "GET DELETE GET PUT GET", wrote: true},
		{name: "clear", desired: []string{}, reads: []string{`[` + exemptionAlice + `,` + exemptionBob + `]`, `[]`}, methods: "GET DELETE GET", wrote: true},
		{name: "matching", desired: []string{"ALICE", "bob"}, reads: []string{`[` + exemptionBob + `,` + exemptionAlice + `]`}, methods: "GET"},
		{name: "empty matching", desired: []string{}, reads: []string{`[]`}, methods: "GET"},
		{name: "preflight failure", desired: []string{"alice"}, reads: []string{`[]`}, methods: "GET", failureAt: 1, failStatus: 403, fail: true, pre: true},
		{name: "remove failure", desired: []string{"carol"}, reads: []string{`[` + exemptionBob + `]`}, methods: "GET DELETE", failureAt: 2, failStatus: 403, fail: true},
		{name: "remove wrong status", desired: []string{}, reads: []string{`[` + exemptionBob + `]`}, methods: "GET DELETE", failureAt: 2, failStatus: 200, fail: true},
		{name: "intermediate denied", desired: []string{"carol"}, reads: []string{`[` + exemptionBob + `]`, `[]`}, methods: "GET DELETE GET", failureAt: 3, failStatus: 403, fail: true},
		{name: "intermediate stale", desired: []string{"carol"}, reads: []string{`[` + exemptionBob + `]`, `[` + exemptionBob + `]`}, methods: "GET DELETE GET", fail: true},
		{name: "intermediate foreign addition", desired: []string{"carol"}, reads: []string{`[` + exemptionBob + `]`, `[` + exemptionAlice + `]`}, methods: "GET DELETE GET", fail: true},
		{name: "intermediate retained identity changed", desired: []string{"alice", "carol"}, reads: []string{`[` + exemptionAlice + `,` + exemptionBob + `]`, `[{"id":99,"login":"alice","type":"User"}]`}, methods: "GET DELETE GET", fail: true},
		{name: "add failed after confirmed remove", desired: []string{"carol"}, reads: []string{`[` + exemptionBob + `]`, `[]`}, methods: "GET DELETE GET PUT", failureAt: 4, failStatus: 403, fail: true, wrote: true},
		{name: "final denied after confirmed remove", desired: []string{"carol"}, reads: []string{`[` + exemptionBob + `]`, `[]`, `[]`}, methods: "GET DELETE GET PUT GET", failureAt: 5, failStatus: 403, fail: true, wrote: true},
		{name: "final wrong membership", desired: []string{"carol"}, reads: []string{`[]`, `[` + exemptionAlice + `]`}, methods: "GET PUT GET", fail: true},
		{name: "final retained identity changed", desired: []string{"alice", "carol"}, reads: []string{`[` + exemptionAlice + `]`, `[{"id":99,"login":"alice","type":"User"},` + exemptionCarol + `]`}, methods: "GET PUT GET", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			reads := 0
			resolved := []string{}
			c := &PullRequestLimitsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				if strings.HasPrefix(req.URL.Path, "/users/") {
					if req.Method != "GET" || strings.Join(methods, " ") != "GET" {
						t.Fatal("resolution must precede writes", req.Method, methods)
					}
					login := strings.TrimPrefix(req.URL.Path, "/users/")
					resolved = append(resolved, login)
					body := exemptionCarol
					if login == "bob" {
						body = exemptionBob
					} else if login != "carol" {
						t.Fatal(login)
					}
					return labelsResponse(req, 200, body), nil
				}
				methods = append(methods, req.Method)
				if req.URL.Path != "/repos/owner/current/interaction-limits/pulls/bypass-list" {
					t.Fatal(req.URL)
				}
				status := 204
				body := `{}`
				if req.Method == "GET" {
					if reads >= len(tc.reads) {
						t.Fatal("extra read")
					}
					status = 200
					body = tc.reads[reads]
					reads++
				} else {
					if req.Method != "PUT" && req.Method != "DELETE" {
						t.Fatal(req.Method)
					}
					var payload map[string][]string
					if json.NewDecoder(req.Body).Decode(&payload) != nil || len(payload) != 1 || len(payload["users"]) == 0 {
						t.Fatal(payload)
					}
					want := []string{"carol"}
					if req.Method == "DELETE" {
						want = []string{"bob"}
						if tc.name == "clear" {
							want = []string{"alice", "bob"}
						}
					} else if tc.name == "add" {
						want = []string{"bob"}
					}
					if !reflect.DeepEqual(payload["users"], want) {
						t.Fatalf("payload %v expected %v", payload, want)
					}
				}
				if len(methods) == tc.failureAt {
					status = tc.failStatus
				}
				return labelsResponse(req, status, body), nil
			})).rest}
			desired, err := settings.NewPullRequestExemptionsDeclaration(tc.desired)
			if err != nil {
				t.Fatal(err)
			}
			wrote, err := c.UpdateExemptions(repository.Repository{Owner: "owner", Name: "current"}, desired)
			wantResolved := []string{}
			if !tc.pre && tc.name != "remove" && tc.name != "clear" && tc.name != "matching" && tc.name != "empty matching" && tc.name != "remove wrong status" {
				wantResolved = []string{"carol"}
				if tc.name == "add" {
					wantResolved = []string{"bob"}
				}
			}
			if !reflect.DeepEqual(resolved, wantResolved) {
				t.Fatalf("resolved=%v wanted=%v", resolved, wantResolved)
			}
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != tc.fail || errors.As(err, &pre) != tc.pre || wrote != tc.wrote || strings.Join(methods, " ") != tc.methods {
				t.Fatalf("wrote=%v err=%v methods=%v", wrote, err, methods)
			}
		})
	}
}

func TestPullRequestExemptionsFullReplacementRemovesBeforeAdding(t *testing.T) {
	desiredNames := make([]string, 100)
	beforeUsers := make([]map[string]any, 100)
	afterUsers := make([]map[string]any, 100)
	for i := range 100 {
		desiredNames[i] = fmt.Sprintf("new-%03d", i)
		beforeUsers[i] = map[string]any{"id": i + 1, "login": fmt.Sprintf("old-%03d", i), "type": "User"}
		afterUsers[i] = map[string]any{"id": i + 101, "login": desiredNames[i], "type": "User"}
	}
	before, _ := json.Marshal(beforeUsers)
	after, _ := json.Marshal(afterUsers)
	calls, resolutions := 0, 0
	c := &PullRequestLimitsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasPrefix(req.URL.Path, "/users/") {
			if req.Method != "GET" || calls != 1 || req.URL.Path != "/users/"+desiredNames[resolutions] {
				t.Fatal(req.Method, req.URL, calls)
			}
			body, _ := json.Marshal(afterUsers[resolutions])
			resolutions++
			return labelsResponse(req, 200, string(body)), nil
		}
		calls++
		if calls > 1 && resolutions != 100 {
			t.Fatal("writes before complete resolution")
		}
		method := []string{"GET", "DELETE", "GET", "PUT", "GET"}[calls-1]
		if req.Method != method {
			t.Fatalf("call%d method%s", calls, req.Method)
		}
		body := "[]"
		status := 200
		switch calls {
		case 1:
			body = string(before)
		case 5:
			body = string(after)
		case 2, 4:
			var payload struct {
				Users []string `json:"users"`
			}
			if json.NewDecoder(req.Body).Decode(&payload) != nil || len(payload.Users) != 100 {
				t.Fatal(payload)
			}
			status = 204
		}
		return labelsResponse(req, status, body), nil
	})).rest}
	desired, err := settings.NewPullRequestExemptionsDeclaration(desiredNames)
	if err != nil {
		t.Fatal(err)
	}
	wrote, err := c.UpdateExemptions(repository.Repository{Owner: "owner", Name: "current"}, desired)
	if err != nil || !wrote || calls != 5 || resolutions != 100 {
		t.Fatal(wrote, err, calls)
	}
}

func TestPullRequestExemptionsResolveBeforeRemoving(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"denied", `{}`, 403}, {"absent", `{}`, 404}, {"redirect", `{}`, 302},
		{"organization", `{"id":3,"login":"carol","type":"Organization"}`, 200},
		{"wrong login", `{"id":3,"login":"other","type":"User"}`, 200},
		{"invalid id", `{"id":0,"login":"carol","type":"User"}`, 200},
		{"duplicate retained id", `{"id":1,"login":"carol","type":"User"}`, 200},
		{"missing", `{"id":3,"login":"carol"}`, 200},
		{"null", `null`, 200}, {"wrong type", `{"id":"3","login":"carol","type":"User"}`, 200},
		{"trailing", exemptionCarol + ` {}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &PullRequestLimitsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" {
					t.Fatal("invalid addition caused a mutation", req.Method)
				}
				if calls == 1 {
					if req.URL.Path != "/repos/owner/current/interaction-limits/pulls/bypass-list" {
						t.Fatal(req.URL)
					}
					return labelsResponse(req, 200, `[`+exemptionAlice+`,`+exemptionBob+`]`), nil
				}
				if calls != 2 || req.URL.Path != "/users/carol" {
					t.Fatal(req.URL, calls)
				}
				return labelsResponse(req, tc.status, tc.body), nil
			})).rest}
			desired, _ := settings.NewPullRequestExemptionsDeclaration([]string{"alice", "carol"})
			wrote, err := c.UpdateExemptions(repository.Repository{Owner: "owner", Name: "current"}, desired)
			var pre interface{ ReadFailedBeforeWrite() }
			if err == nil || wrote || !errors.As(err, &pre) || calls != 2 {
				t.Fatal(wrote, err, calls)
			}
		})
	}
}

func TestPullRequestExemptionsAddedIdentityMustMatchResolution(t *testing.T) {
	calls := 0
	c := &PullRequestLimitsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return labelsResponse(req, 200, `[]`), nil
		case 2:
			if req.Method != "GET" || req.URL.Path != "/users/carol" {
				t.Fatal(req.Method, req.URL)
			}
			return labelsResponse(req, 200, exemptionCarol), nil
		case 3:
			if req.Method != "PUT" {
				t.Fatal(req.Method)
			}
			return labelsResponse(req, 204, `{}`), nil
		case 4:
			return labelsResponse(req, 200, `[{"id":999,"login":"carol","type":"User"}]`), nil
		default:
			t.Fatal("unexpected request")
			return nil, nil
		}
	})).rest}
	desired, _ := settings.NewPullRequestExemptionsDeclaration([]string{"carol"})
	wrote, err := c.UpdateExemptions(repository.Repository{Owner: "owner", Name: "current"}, desired)
	var pre interface{ ReadFailedBeforeWrite() }
	if err == nil || wrote || errors.As(err, &pre) || calls != 4 {
		t.Fatal(wrote, err, calls)
	}
}
