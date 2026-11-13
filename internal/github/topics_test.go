package github

import (
	"encoding/json"
	"errors"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func topicsClient(t *testing.T, handle transportFunc) *Client {
	t.Helper()
	rest, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true, Transport: handle, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{rest: rest}
}
func TestReadTopics(t *testing.T) {
	for _, body := range []string{`{"names":["z","a"]}`, `{"names":[]}`, `{}`, `{"names":null}`, `{"names":"a"}`, `{"names":[null]}`, `{"names":["a","a"]}`, `{"names":["Upper"]}`, `not json`} {
		t.Run(body, func(t *testing.T) {
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" || req.URL.Path != "/repos/owner/current/topics" {
					t.Fatalf("request=%s %s", req.Method, req.URL)
				}
				return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			}))
			got, err := client.ReadTopics(repository.Repository{Owner: "owner", Name: "current"})
			valid := body == `{"names":["z","a"]}` || body == `{"names":[]}`
			if valid && (err != nil || got == nil) || !valid && (err == nil || got != nil) {
				t.Fatalf("names=%v error=%v", got, err)
			}
			if strings.Contains(body, "z") && !reflect.DeepEqual(got, []string{"a", "z"}) {
				t.Fatalf("names=%v", got)
			}
		})
	}
}
func TestApplyTopicsReconciliation(t *testing.T) {
	for _, test := range []struct {
		name, before, after                 string
		names                               []string
		readStatus, putStatus, verifyStatus int
		wantCalls                           []string
		wantWrote, wantError, prewrite      bool
	}{
		{name: "exact match", names: []string{"a", "z"}, before: `{"names":["a","z"]}`, wantCalls: []string{"GET"}},
		{name: "reordered match", names: []string{"z", "a"}, before: `{"names":["a","z"]}`, wantCalls: []string{"GET"}},
		{name: "empty match", names: []string{}, before: `{"names":[]}`, wantCalls: []string{"GET"}},
		{name: "replace full set", names: []string{"z", "a"}, before: `{"names":["old"]}`, after: `{"names":["z","a"]}`, wantCalls: []string{"GET", "PUT", "GET"}, wantWrote: true},
		{name: "clear", names: []string{}, before: `{"names":["old"]}`, after: `{"names":[]}`, wantCalls: []string{"GET", "PUT", "GET"}, wantWrote: true},
		{name: "denied read", readStatus: 403, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "redirected read", readStatus: 302, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "missing names", before: `{}`, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "null names", before: `{"names":null}`, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "wrong type", before: `{"names":false}`, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "invalid name", before: `{"names":["UPPER"]}`, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "duplicate name", before: `{"names":["a","a"]}`, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "malformed read", before: `{`, wantCalls: []string{"GET"}, wantError: true, prewrite: true},
		{name: "rejected PUT", before: `{"names":["old"]}`, putStatus: 403, wantCalls: []string{"GET", "PUT"}, wantError: true},
		{name: "redirected PUT", before: `{"names":["old"]}`, putStatus: 302, wantCalls: []string{"GET", "PUT"}, wantError: true},
		{name: "denied verification", before: `{"names":["old"]}`, verifyStatus: 403, wantCalls: []string{"GET", "PUT", "GET"}, wantError: true},
		{name: "redirected verification", before: `{"names":["old"]}`, verifyStatus: 302, wantCalls: []string{"GET", "PUT", "GET"}, wantError: true},
		{name: "incomplete verification", before: `{"names":["old"]}`, after: `{}`, wantCalls: []string{"GET", "PUT", "GET"}, wantError: true},
		{name: "mismatching verification", before: `{"names":["old"]}`, after: `{"names":["old"]}`, wantCalls: []string{"GET", "PUT", "GET"}, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []string
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				calls = append(calls, req.Method)
				if req.URL.Path != "/repos/owner/current/topics" || len(calls) > len(test.wantCalls) || req.Method != test.wantCalls[len(calls)-1] {
					t.Fatalf("unexpected request %s %s; calls=%v", req.Method, req.URL, calls)
				}
				body, status := test.before, test.readStatus
				if req.Method == "PUT" {
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					wantNames := []any{}
					if len(test.names) > 0 {
						wantNames = []any{"a", "z"}
					}
					if !reflect.DeepEqual(payload, map[string]any{"names": wantNames}) {
						t.Fatalf("payload=%v", payload)
					}
					// The acknowledgement cannot establish success; only the next GET can.
					body, status = `{"names":["acknowledgement-only"]}`, test.putStatus
				} else if len(calls) > 1 {
					body, status = test.after, test.verifyStatus
				}
				if status == 0 {
					status = http.StatusOK
				}
				response := labelsResponse(req, status, body)
				if status == 302 {
					response.Header.Set("Location", "https://api.github.com/repos/other/wrong/topics")
				}
				return response, nil
			})
			wrote, err := client.ApplyTopics(repository.Repository{Owner: "owner", Name: "current"}, settings.TopicsDeclaration{Names: &test.names})
			var marker interface {
				error
				ReadFailedBeforeWrite()
			}
			if wrote != test.wantWrote || (err != nil) != test.wantError || errors.As(err, &marker) != test.prewrite || !reflect.DeepEqual(calls, test.wantCalls) {
				t.Fatalf("wrote=%v error=%v calls=%v prewrite=%v", wrote, err, calls, marker != nil)
			}
			if err != nil && !test.prewrite && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatalf("uncertain effects missing: %v", err)
			}
		})
	}
}

func TestApplyTopicsRepeatAndOmission(t *testing.T) {
	current := []string{"old"}
	var calls []string
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method)
		if req.Method == "PUT" {
			var payload struct {
				Names []string `json:"names"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			current = payload.Names
		}
		body, _ := json.Marshal(map[string]any{"names": current})
		return labelsResponse(req, 200, string(body)), nil
	})
	repo := repository.Repository{Owner: "owner", Name: "current"}
	if wrote, err := client.ApplyTopics(repo, settings.TopicsDeclaration{}); wrote || err != nil || len(calls) != 0 {
		t.Fatal(wrote, err, calls)
	}
	names := []string{"z", "a"}
	for i := range 2 {
		wrote, err := client.ApplyTopics(repo, settings.TopicsDeclaration{Names: &names})
		if err != nil || wrote != (i == 0) {
			t.Fatal(wrote, err)
		}
	}
	if !reflect.DeepEqual(calls, []string{"GET", "PUT", "GET", "GET"}) || !reflect.DeepEqual(names, []string{"z", "a"}) {
		t.Fatal(calls, names)
	}
}

func TestApplyTopicsTransportFailureProvenance(t *testing.T) {
	for failureAt := range 3 {
		t.Run([]string{"preflight", "PUT", "verification"}[failureAt], func(t *testing.T) {
			cause, calls := errors.New("connection lost"), 0
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				stage := calls
				calls++
				if stage == failureAt {
					return nil, cause
				}
				return labelsResponse(req, 200, `{"names":["old"]}`), nil
			})
			names := []string{}
			wrote, err := client.ApplyTopics(repository.Repository{Owner: "owner", Name: "current"}, settings.TopicsDeclaration{Names: &names})
			var marker interface {
				error
				ReadFailedBeforeWrite()
			}
			if wrote || !errors.Is(err, cause) || errors.As(err, &marker) != (failureAt == 0) || calls != failureAt+1 {
				t.Fatal(wrote, err, calls)
			}
		})
	}
}
