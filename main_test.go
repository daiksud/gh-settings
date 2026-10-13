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
	"go.yaml.in/yaml/v4"
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

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		t.Fatal("version command must not call GitHub")
		return nil, nil
	}})
	if code != 0 || stdout.String() != "gh-settings "+version+"\n" || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunSelectsOperation(t *testing.T) {
	operationError := errors.New("operation failed")
	for _, test := range []struct {
		name              string
		args              []string
		operationError    error
		wantExportCalls   int
		wantValidateCalls int
		wantStatus        int
	}{
		{"export", []string{"export"}, nil, 1, 0, 0},
		{"validate", []string{"validate"}, nil, 0, 1, 0},
		{"export failure", []string{"export"}, operationError, 1, 0, 1},
		{"validate failure", []string{"validate"}, operationError, 0, 1, 1},
		{"help", []string{"--help"}, nil, 0, 0, 0},
		{"short help", []string{"-h"}, nil, 0, 0, 0},
		{"version", []string{"--version"}, nil, 0, 0, 0},
		{"short version", []string{"-v"}, nil, 0, 0, 0},
		{"no arguments", nil, nil, 0, 0, 1},
		{"apply", []string{"apply"}, nil, 0, 0, 1},
		{"plan", []string{"plan"}, nil, 0, 0, 1},
		{"export arguments", []string{"export", "--repo", "owner/other"}, nil, 0, 0, 1},
		{"validate arguments", []string{"validate", "--repo", "owner/other"}, nil, 0, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exportCalls, validateCalls := 0, 0
			status := run(test.args, &stdout, &stderr, commandOperations{
				export: func() ([]byte, error) {
					exportCalls++
					return []byte("settings\n"), test.operationError
				},
				validate: func() error {
					validateCalls++
					return test.operationError
				},
			})
			if exportCalls != test.wantExportCalls || validateCalls != test.wantValidateCalls {
				t.Fatalf("export calls=%d, validate calls=%d", exportCalls, validateCalls)
			}
			if status != test.wantStatus {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
			if test.operationError != nil && (stdout.Len() != 0 || stderr.String() != "gh settings: operation failed\n") {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "validate" && (stdout.Len() != 0 || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "export" && (stdout.String() != "settings\n" || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
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

func TestExportUsesOrigin(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "--quiet", dir}, {"-C", dir, "remote", "add", "origin", "https://github.com/owner/current.git"}} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	t.Chdir(dir)
	t.Setenv("GH_REPO", "other/repository")
	repo, err := currentRepository()
	if err != nil {
		t.Fatal(err)
	}
	if repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
		t.Fatalf("selected %+v", repo)
	}
}

func TestExportUsesConfiguredOriginBeforeInsteadOf(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet", dir},
		{"-C", dir, "remote", "add", "origin", "https://github.com/owner/current.git"},
		{"-C", dir, "remote", "set-url", "--add", "origin", "https://github.com/owner/other.git"},
		{"-C", dir, "config", "--local", "url.https://mirror.example/.insteadOf", "https://github.com/"},
	} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	t.Chdir(dir)
	if output, err := exec.Command("git", "remote", "get-url", "origin").Output(); err != nil || string(output) != "https://mirror.example/owner/current.git\n" {
		t.Fatalf("rewritten origin=%q err=%v", output, err)
	}
	repo, err := currentRepository()
	if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
		t.Fatalf("selected %+v, err=%v", repo, err)
	}
}

func TestExportUsesAliasRewrittenToGitHub(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet", dir},
		{"-C", dir, "remote", "add", "origin", "gh:owner/current.git"},
		{"-C", dir, "config", "--local", "url.https://github.com/.insteadOf", "gh:"},
	} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	t.Chdir(dir)
	if output, err := exec.Command("git", "remote", "get-url", "origin").Output(); err != nil || string(output) != "https://github.com/owner/current.git\n" {
		t.Fatalf("rewritten origin=%q err=%v", output, err)
	}
	repo, err := currentRepository()
	if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
		t.Fatalf("selected %+v, err=%v", repo, err)
	}
}

func TestExportRejectsGlobalOriginOutsideCheckout(t *testing.T) {
	config := filepath.Join(t.TempDir(), "global-gitconfig")
	if err := os.WriteFile(config, []byte("[remote \"origin\"]\n\turl = https://github.com/owner/current.git\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Chdir(t.TempDir())
	if repo, err := currentRepository(); err == nil {
		t.Fatalf("selected global origin outside checkout: %+v", repo)
	}
}

func TestExportUsesRepositoryConfigSources(t *testing.T) {
	for _, source := range []string{"include", "worktree"} {
		t.Run(source, func(t *testing.T) {
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			dir := t.TempDir()
			if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
				t.Fatalf("git init: %v: %s", err, output)
			}
			var commands [][]string
			if source == "include" {
				included := filepath.Join(dir, "included.conf")
				if err := os.WriteFile(included, []byte("[remote \"origin\"]\n\turl = https://github.com/owner/current.git\n"), 0600); err != nil {
					t.Fatal(err)
				}
				commands = [][]string{{"config", "--local", "include.path", included}}
			} else {
				commands = [][]string{
					{"config", "--local", "extensions.worktreeConfig", "true"},
					{"config", "--worktree", "remote.origin.url", "https://github.com/owner/current.git"},
				}
			}
			for _, args := range commands {
				cmd := exec.Command("git", args...)
				cmd.Dir = dir
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %v: %v: %s", args, err, output)
				}
			}
			t.Chdir(dir)
			if output, err := exec.Command("git", "remote", "get-url", "origin").Output(); err != nil || string(output) != "https://github.com/owner/current.git\n" {
				t.Fatalf("origin=%q err=%v", output, err)
			}
			repo, err := currentRepository()
			if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
				t.Fatalf("selected %+v, err=%v", repo, err)
			}
		})
	}
}

func TestExportRepositoryFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	code := run([]string{"export"}, &stdout, &stderr, commandOperations{export: exportCurrent})
	if code == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestUnsupportedCommand(t *testing.T) {
	for _, args := range [][]string{nil, {"apply"}, {"plan"}, {"export", "--repo", "owner/other"}, {"validate", "--repo", "owner/other"}} {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
			t.Fatal("unsupported command invoked API operation")
			return nil, errors.New("unexpected API call")
		}})
		if code == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("args=%v status=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
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

func TestExportOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"export"}, brokenWriter{}, &stderr, commandOperations{export: func() ([]byte, error) { return []byte("description: test\n"), nil }})
	if code == 0 || !strings.Contains(stderr.String(), "write output") {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestRepositoryRemotes(t *testing.T) {
	for _, remote := range []string{"git@github.com:owner/current.git", "github.com:owner/current.git", "ssh://git@github.com/owner/current.git", "../owner/current", "owner/current", "https://gitlab.com/owner/current.git"} {
		t.Run(remote, func(t *testing.T) {
			dir := t.TempDir()
			for _, args := range [][]string{{"init", "--quiet", dir}, {"-C", dir, "remote", "add", "origin", remote}} {
				if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
					t.Fatalf("git: %v: %s", err, out)
				}
			}
			subdir := filepath.Join(dir, "subdirectory")
			if err := os.Mkdir(subdir, 0700); err != nil {
				t.Fatal(err)
			}
			t.Chdir(subdir)
			repo, err := currentRepository()
			if strings.Contains(remote, "github.com") {
				if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
					t.Fatalf("repo=%+v err=%v", repo, err)
				}
			} else if err == nil {
				t.Fatalf("non-GitHub remote selected: %+v", repo)
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

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"--help"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) { t.Fatal("help invoked API operation"); return nil, nil }})
	if status != 0 || !strings.Contains(stdout.String(), "gh settings export") || stderr.Len() != 0 {
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
