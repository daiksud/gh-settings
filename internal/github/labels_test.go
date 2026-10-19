package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func labelsResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestLabelsPaginationAndCompleteRead(t *testing.T) {
	for _, test := range []struct {
		name   string
		last   string
		status int
		want   int
	}{
		{"later page", `[{"name":"last","color":"ABCDEF","description":null,"default":true}]`, 200, 101},
		{"exact page", `[]`, 200, 100},
		{"later failure", `{"message":"denied"}`, 403, 0},
		{"later invalid state", `[{"name":"last","color":"bad","description":""}]`, 200, 0},
		{"later incomplete state", `[{"name":"last","color":"abcdef"}]`, 200, 0},
		{"cross page duplicate", `[{"name":"LABEL-0","color":"abcdef","description":""}]`, 200, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Path != "/repos/owner/current/labels" || req.URL.Query().Get("per_page") != "100" || req.URL.Query().Get("page") != fmt.Sprint(calls) {
					t.Fatalf("request=%s %s", req.Method, req.URL)
				}
				if calls == 1 {
					labels := make([]map[string]any, 100)
					for i := range labels {
						labels[i] = map[string]any{"name": fmt.Sprintf("label-%d", i), "color": "abcdef", "description": ""}
					}
					body, _ := json.Marshal(labels)
					return labelsResponse(req, 200, string(body)), nil
				}
				if calls != 2 {
					t.Fatal("unexpected extra page")
				}
				return labelsResponse(req, test.status, test.last), nil
			}))
			got, err := client.ReadLabels(repository.Repository{Owner: "owner", Name: "current"})
			values, readErr := got.Values()
			if calls != 2 || (test.want == 0 && (err == nil || readErr == nil || values != nil)) || (test.want > 0 && (err != nil || readErr != nil || len(values) != test.want)) {
				t.Fatalf("got=%v err=%v calls=%d", got, err, calls)
			}
			if test.want == 101 && (values[100].Color != "abcdef" || values[100].Description != "") {
				t.Fatalf("last=%+v", values[100])
			}
		})
	}
}

func TestReadLabelsRejectsIncompleteState(t *testing.T) {
	for _, body := range []string{`[{"color":"abcdef","description":""}]`, `[{"name":"a","description":""}]`, `[{"name":42,"color":"abcdef","description":""}]`, `[{"name":"a","color":false,"description":""}]`, `[{"name":"","color":"abcdef","description":""}]`, `null`, `{}`, `[null]`, `not json`, `[{"name":"a","color":"abcdef"}]`, `[{"name":null,"color":"abcdef","description":""}]`, `[{"name":"a","color":null,"description":""}]`, `[{"name":"a","color":"#abcdef","description":""}]`, `[{"name":"a","color":"abcdef","description":false}]`, `[{"name":"a","color":"abcdef","description":""},{"name":"A","color":"abcdef","description":""}]`} {
		t.Run(body, func(t *testing.T) {
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, body), nil }))
			got, err := client.ReadLabels(repository.Repository{Owner: "owner", Name: "current"})
			values, readErr := got.Values()
			if err == nil || readErr == nil || values != nil {
				t.Fatalf("got=%v err=%v", got, err)
			}
		})
	}
}

func TestReadLabelsCompleteEmptyObservation(t *testing.T) {
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, `[]`), nil }))
	observed, err := client.ReadLabels(repository.Repository{Owner: "owner", Name: "current"})
	values, readErr := observed.Values()
	if err != nil || readErr != nil || values == nil || len(values) != 0 {
		t.Fatalf("values=%+v error=%v read error=%v", values, err, readErr)
	}
}

func TestApplyLabelsExactPayloadsAndNames(t *testing.T) {
	color, empty := "123456", ""
	for _, name := range []string{"area/docs", "a?b#c%", ".", "..", "✨ help wanted"} {
		t.Run(name, func(t *testing.T) {
			before := settings.LabelState{Name: name, Color: "abcdef", Description: "old"}
			changes := []settings.LabelChange{{Operation: "update", Name: name, Before: &before, After: &settings.LabelDeclaration{Name: name, Description: &empty}}}
			calls := 0
			client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if req.Method != "PATCH" || req.URL.Path != "/repos/owner/current/labels/"+name || !reflect.DeepEqual(payload, map[string]any{"description": ""}) {
					t.Fatalf("request=%s %s payload=%v", req.Method, req.URL, payload)
				}
				if strings.ContainsAny(name, "/?#%") && req.URL.EscapedPath() == req.URL.Path {
					t.Fatalf("unescaped path=%s", req.URL)
				}
				if (name == "." || name == "..") && !strings.Contains(req.URL.EscapedPath(), "%2E") {
					t.Fatalf("dot not encoded: %s", req.URL)
				}
				return labelsResponse(req, 200, `{}`), nil
			}))
			if err := client.ApplyLabels(repository.Repository{Owner: "owner", Name: "current"}, changes); err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
	changes := []settings.LabelChange{
		{Operation: "create", Name: "default", After: &settings.LabelDeclaration{Name: "default"}},
		{Operation: "create", Name: "explicit", After: &settings.LabelDeclaration{Name: "explicit", Color: &color, Description: &empty}},
		{Operation: "delete", Name: "old", Before: &settings.LabelState{Name: "old", Color: "abcdef", Description: "old"}},
	}
	calls := 0
	client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1, 2:
			var payload map[string]any
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"name": "default"}
			if calls == 2 {
				want = map[string]any{"name": "explicit", "color": "123456", "description": ""}
			}
			if req.Method != "POST" || req.URL.Path != "/repos/owner/current/labels" || !reflect.DeepEqual(payload, want) {
				t.Fatalf("request=%s %s payload=%v", req.Method, req.URL, payload)
			}
		case 3:
			if req.Method != "DELETE" || req.URL.Path != "/repos/owner/current/labels/old" || req.Body != nil {
				t.Fatalf("request=%+v", req)
			}
		default:
			t.Fatal("unexpected request")
		}
		return labelsResponse(req, 204, ``), nil
	}))
	if err := client.ApplyLabels(repository.Repository{Owner: "owner", Name: "current"}, changes); err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestApplyLabelsPrevalidationAndPartialFailure(t *testing.T) {
	changes := []settings.LabelChange{
		{Operation: "create", Name: "new", After: &settings.LabelDeclaration{Name: "new"}},
		{Operation: "delete", Name: "old", Before: &settings.LabelState{Name: "old", Color: "abcdef", Description: ""}},
	}
	calls := 0
	client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return labelsResponse(req, 403, `{"message":"denied"}`), nil
	}))
	invalid := append(append([]settings.LabelChange{}, changes...), settings.LabelChange{Operation: "rename", Name: "bad"})
	if err := client.ApplyLabels(repository.Repository{}, invalid); err == nil || calls != 0 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if err := client.ApplyLabels(repository.Repository{Owner: "owner", Name: "current"}, changes); err == nil || calls != 1 || !strings.Contains(err.Error(), "earlier successful writes remain applied") {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
