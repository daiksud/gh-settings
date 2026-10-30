package github

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestApplyRepositorySettings(t *testing.T) {
	empty, escaped, homepage := "", "quote\"\n\\", "https://example.com"
	disabled, enabled := false, true
	for _, test := range []struct {
		name    string
		desired settings.RepositoryDeclaration
		want    map[string]any
	}{
		{"partial", settings.RepositoryDeclaration{HasIssues: &disabled}, map[string]any{"has_issues": false}},
		{"five zero values", settings.RepositoryDeclaration{Description: &empty, Homepage: &empty, HasIssues: &disabled, HasProjects: &disabled, HasWiki: &disabled}, map[string]any{"description": "", "homepage": "", "has_issues": false, "has_projects": false, "has_wiki": false}},
		{"strings and true", settings.RepositoryDeclaration{Description: &escaped, Homepage: &homepage, HasWiki: &enabled}, map[string]any{"description": escaped, "homepage": homepage, "has_wiki": true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				var got map[string]any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if req.Method != http.MethodPatch || req.URL.Path != "/repos/owner/current" || !reflect.DeepEqual(got, test.want) {
					t.Fatalf("request=%s %s body=%v, want PATCH current body=%v", req.Method, req.URL.Path, got, test.want)
				}
				return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})
			err := client.ApplyRepository(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, test.desired)
			if err != nil || calls != 1 {
				t.Fatalf("error=%v calls=%d, want one PATCH", err, calls)
			}
		})
	}
}

func TestApplyRepositoryFailures(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
		transportError   error
	}{
		{name: "permission", body: `{"message":"Resource not accessible"}`, status: 403, want: "403"},
		{name: "server validation", body: `{"message":"Validation Failed"}`, status: 422, want: "422"},
		{name: "bad response", body: `not json`, status: 200, want: "apply repository settings"},
		{name: "transport", transportError: errors.New("network unavailable"), want: "network unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.Path != "/repos/owner/current" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				if test.transportError != nil {
					return nil, test.transportError
				}
				return &http.Response{Request: req, StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})
			value := false
			err := client.ApplyRepository(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{HasIssues: &value})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func writeTestClient(t *testing.T, transport transportFunc) *Client {
	t.Helper()
	client, err := newWriteClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true, Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
