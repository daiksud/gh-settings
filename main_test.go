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
	code := run([]string{"export"}, &stdout, &stderr, func() ([]byte, error) {
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
	})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d, stderr=%q", code, stderr.String())
	}
	var output map[string]any
	if err := yaml.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
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
	code := run([]string{"--version"}, &stdout, &stderr, func() ([]byte, error) {
		t.Fatal("version command must not call GitHub")
		return nil, nil
	})
	if code != 0 || stdout.String() != "gh-settings "+version+"\n" || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

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
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
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
	code := run([]string{"export"}, &stdout, &stderr, func() ([]byte, error) {
		return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, `{"message":"Resource not accessible"}`, 403))
	})
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

func TestExportRepositoryFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	code := run([]string{"export"}, &stdout, &stderr, exportCurrent)
	if code == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestUnsupportedCommand(t *testing.T) {
	for _, args := range [][]string{nil, {"apply"}, {"plan"}, {"validate"}, {"export", "--repo", "owner/other"}} {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr, func() ([]byte, error) {
			t.Fatal("unsupported command invoked API operation")
			return nil, errors.New("unexpected API call")
		})
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
			status := run([]string{"export"}, &stdout, &stderr, func() ([]byte, error) {
				return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, body, 200))
			})
			if status == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestExportOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"export"}, brokenWriter{}, &stderr, func() ([]byte, error) { return []byte("description: test\n"), nil })
	if code == 0 || !strings.Contains(stderr.String(), "write output") {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestRepositoryRemotes(t *testing.T) {
	for _, remote := range []string{"git@github.com:owner/current.git", "ssh://git@github.com/owner/current.git", "../owner/current", "owner/current", "https://gitlab.com/owner/current.git"} {
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
				if err != nil || repo.Owner != "owner" || repo.Name != "current" {
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
	status := run([]string{"export"}, &stdout, &stderr, func() ([]byte, error) {
		return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, client)
	})
	if status == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "network unavailable") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"--help"}, &stdout, &stderr, func() ([]byte, error) { t.Fatal("help invoked API operation"); return nil, nil })
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
			status := run([]string{"export"}, &stdout, &stderr, func() ([]byte, error) {
				return exportSettings(repository.Repository{Owner: "owner", Name: "current"}, testClient(t, body, 200))
			})
			if status != 0 || stderr.Len() != 0 {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
			var result map[string]any
			if err := yaml.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
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
