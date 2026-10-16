package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestApplySettings(t *testing.T) {
	for _, test := range []struct{ name, input, want string }{
		{"partial", repositoryYAML("has_issues: false\n"), `{"has_issues":false}`},
		{"five zero values", repositoryYAML("description: ''\nhomepage: ''\nhas_issues: false\nhas_projects: false\nhas_wiki: false\n"), `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`},
		{"strings and true", repositoryYAML("description: \"quote\\\"\\n\\\\\"\nhomepage: https://example.com\nhas_wiki: true\n"), `{"description":"quote\"\n\\","homepage":"https://example.com","has_wiki":true}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			desired, err := workspace.DecodeConfig([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			client := applyTestClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				if req.Method != http.MethodPatch || req.URL.Path != "/repos/owner/current" || string(body) != test.want {
					t.Fatalf("request=%s %s body=%s, want PATCH current body=%s", req.Method, req.URL.Path, body, test.want)
				}
				return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})
			got, err := applySettings(desired, repository.Repository{Owner: "owner", Name: "current"}, client)
			if err != nil || string(got) != "Applied repository settings.\n" || calls != 1 {
				t.Fatalf("output=%q error=%v calls=%d", got, err, calls)
			}
		})
	}
}

func applyTestClient(t *testing.T, transport transportFunc) *api.RESTClient {
	t.Helper()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	client, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true, Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestApplySettingsFailures(t *testing.T) {
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
			client := applyTestClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch {
					t.Fatalf("unexpected method: %s", req.Method)
				}
				if test.transportError != nil {
					return nil, test.transportError
				}
				return &http.Response{Request: req, StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})
			var stdout, stderr bytes.Buffer
			value := false
			code := run([]string{"apply"}, &stdout, &stderr, commandOperations{apply: func() ([]byte, error) {
				return applySettings(settings.RepositoryDeclaration{HasIssues: &value}, repository.Repository{Owner: "owner", Name: "current"}, client)
			}})
			if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestApplyCurrentValidatesAndNoopsLocally(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.Mkdir(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir) // No origin: invalid input and no-op must finish before origin/authentication.
	for _, test := range []struct{ input, want string }{
		{"settings: {}\n", "missing schemaVersion"},
		{repositoryYAML("has_issues: [\n"), "decode YAML"},
		{repositoryYAML("description: null\n"), "must be a string"},
		{repositoryYAML("unknown: false\n"), "unknown setting"},
		{"schemaVersion: 1\n", ""},
		{"schemaVersion: 1\nsettings: {}\n", ""},
		{"schemaVersion: 1\nsettings:\n  repository: {}\n", ""},
	} {
		if err := os.WriteFile(filepath.Join(dir, settingsPath), []byte(test.input), 0600); err != nil {
			t.Fatal(err)
		}
		data, err := applyCurrent()
		if test.want == "" {
			if err != nil || string(data) != "No managed repository settings.\n" {
				t.Fatalf("output=%q error=%v", data, err)
			}
		} else if len(data) != 0 || err == nil || !strings.Contains(err.Error(), "validate "+settingsPath) || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("output=%q error=%v", data, err)
		}
	}
}

func TestApplyOutputFailure(t *testing.T) {
	for _, message := range []string{"Applied repository settings.\n", "No managed repository settings.\n"} {
		var stderr bytes.Buffer
		code := run([]string{"apply"}, brokenWriter{}, &stderr, commandOperations{apply: func() ([]byte, error) { return []byte(message), nil }})
		if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
			t.Fatalf("status=%d stderr=%q", code, stderr.String())
		}
	}
}
