package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"go.yaml.in/yaml/v4"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestExportFiveSettings covers the first export.feature.md scenario.
func TestExportFiveSettings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"export"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		client, err := api.NewRESTClient(api.ClientOptions{
			Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true,
			Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.Path != "/repos/owner/current" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}
				return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"id":123,"private":true}`))}, nil
			}),
		})
		if err != nil {
			t.Fatal(err)
		}
		return exportSettings(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, client)
	}})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d, stderr=%q", code, stderr.String())
	}
	output := exportedRepository(t, stdout.Bytes())
	expected := map[string]any{"description": "Example repository", "homepage": "https://example.com", "has_issues": true, "has_projects": false, "has_wiki": true}
	if len(output) != len(expected) {
		t.Fatalf("keys=%v", output)
	}
	for key, value := range expected {
		if output[key] != value {
			t.Errorf("%s = %#v, want %#v", key, output[key], value)
		}
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func exportedRepository(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	groups, ok := root["settings"].(map[string]any)
	if !ok || len(root) != 2 || root["schemaVersion"] != 1 || len(groups) != 1 {
		t.Fatalf("export root=%v, want schemaVersion: 1 and settings.repository", root)
	}
	repository, ok := groups["repository"].(map[string]any)
	if !ok {
		t.Fatalf("settings=%v, want repository mapping", groups)
	}
	return repository
}

func testClient(t *testing.T, body string, status int) *api.RESTClient {
	t.Helper()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	client, err := api.NewRESTClient(api.ClientOptions{
		Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true,
		Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("mutation request: %s", req.Method)
			}
			return &http.Response{Request: req, StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestReadRepositorySettings(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	got, err := readRepositorySettings(repo, testClient(t, `{"description":null,"homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, 200))
	if err != nil || got.Description == nil || *got.Description != "" || got.Homepage == nil || *got.Homepage != "https://example.com" || got.HasIssues == nil || !*got.HasIssues || got.HasProjects == nil || *got.HasProjects || got.HasWiki == nil || !*got.HasWiki {
		t.Fatalf("settings=%+v err=%v, want typed values for all five fields", got, err)
	}
	_, err = readRepositorySettings(repo, testClient(t, `{"message":"Resource not accessible"}`, 403))
	if err == nil || !strings.Contains(err.Error(), "read repository settings:") {
		t.Fatalf("read error=%v, want repository read context", err)
	}
}

func TestExportLiteralOutput(t *testing.T) {
	for _, test := range []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "five values",
			response: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`,
			want:     "schemaVersion: 1\nsettings:\n    repository:\n        description: Example repository\n        homepage: https://example.com\n        has_issues: true\n        has_projects: false\n        has_wiki: true\n",
		},
		{
			name:     "zero values",
			response: `{"description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`,
			want:     "schemaVersion: 1\nsettings:\n    repository:\n        description: \"\"\n        homepage: \"\"\n        has_issues: false\n        has_projects: false\n        has_wiki: false\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, test.response, 200))
			if err != nil || string(got) != test.want {
				t.Fatalf("output=%q err=%v, want %q", got, err, test.want)
			}
		})
	}
}

func TestExportPreservesZeroValues(t *testing.T) {
	data, err := exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`, 200))
	if err != nil {
		t.Fatal(err)
	}
	result := exportedRepository(t, data)
	if len(result) != 5 {
		t.Fatalf("fields: %v", result)
	}
	for _, key := range []string{"description", "homepage"} {
		if value, ok := result[key]; !ok || value != "" {
			t.Errorf("%s=%#v, present=%v", key, value, ok)
		}
	}
	for _, key := range []string{"has_issues", "has_projects", "has_wiki"} {
		if value, ok := result[key]; !ok || value != false {
			t.Errorf("%s=%#v, present=%v", key, value, ok)
		}
	}
}

func TestExportReadFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"export"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, `{"message":"Resource not accessible"}`, 403))
	}})
	if code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "403") {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestExportRejectsIncompleteResponse(t *testing.T) {
	for _, body := range []string{
		`not json`,
		`{"homepage":null,"has_issues":true,"has_projects":false,"has_wiki":true}`,
		`{"description":null,"has_issues":true,"has_projects":false,"has_wiki":true}`,
		`{"description":"ok","homepage":"","has_issues":true,"has_projects":false}`,
		`{"description":"ok","homepage":"","has_issues":"true","has_projects":false,"has_wiki":true}`,
		`{"description":"ok","homepage":"","has_issues":null,"has_projects":false,"has_wiki":true}`,
	} {
		t.Run(body, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run([]string{"export"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
				return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, body, 200))
			}})
			if status == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestExportNetworkFailure(t *testing.T) {
	client, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true, Transport: transportFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("network unavailable") })})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := run([]string{"export"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, client)
	}})
	if status == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "network unavailable") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestExportNormalizesUnsetStrings(t *testing.T) {
	for _, body := range []string{
		`{"description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`,
		`{"description":null,"homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`,
		`{"description":"","homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`,
	} {
		t.Run(body, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run([]string{"export"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
				return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, body, 200))
			}})
			if status != 0 || stderr.Len() != 0 {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
			result := exportedRepository(t, stdout.Bytes())
			if len(result) != 5 {
				t.Fatalf("fields=%v", result)
			}
			for _, key := range []string{"description", "homepage"} {
				value, ok := result[key]
				if !ok || value != "" {
					t.Errorf("%s=%#v, present=%v; want empty string", key, value, ok)
				}
			}
			for _, key := range []string{"has_issues", "has_projects", "has_wiki"} {
				if value, ok := result[key]; !ok || value != false {
					t.Errorf("%s=%#v present=%v", key, value, ok)
				}
			}
		})
	}
}

func TestDecodeRepositoryResponse(t *testing.T) {
	response := map[string]json.RawMessage{
		"description":  json.RawMessage(`null`),
		"homepage":     json.RawMessage(`"https://example.com"`),
		"has_issues":   json.RawMessage(`true`),
		"has_projects": json.RawMessage(`false`),
		"has_wiki":     json.RawMessage(`true`),
		"private":      json.RawMessage(`true`),
	}
	got, err := decodeRepositoryResponse(response)
	if err != nil || got.Description == nil || *got.Description != "" || got.Homepage == nil || *got.Homepage != "https://example.com" || got.HasIssues == nil || !*got.HasIssues || got.HasProjects == nil || *got.HasProjects || got.HasWiki == nil || !*got.HasWiki {
		t.Fatalf("settings=%+v err=%v, want all five present with normalized values", got, err)
	}

	for _, test := range []struct {
		name      string
		field     string
		value     json.RawMessage
		wantError string
	}{
		{"missing field", "homepage", nil, "GitHub response is missing homepage"},
		{"null boolean", "has_issues", json.RawMessage(`null`), "GitHub returned null for has_issues"},
		{"wrong string type", "homepage", json.RawMessage(`123`), "decode homepage: json: cannot unmarshal number into Go value of type string"},
		{"wrong boolean type", "has_issues", json.RawMessage(`"true"`), "decode has_issues: json: cannot unmarshal string into Go value of type bool"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := make(map[string]json.RawMessage, len(response))
			for key, value := range response {
				invalid[key] = value
			}
			if test.value == nil {
				delete(invalid, test.field)
			} else {
				invalid[test.field] = test.value
			}
			_, err := decodeRepositoryResponse(invalid)
			if err == nil || err.Error() != test.wantError {
				t.Fatalf("error=%v, want %q", err, test.wantError)
			}
		})
	}
}
