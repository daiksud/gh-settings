package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPlanSettingsReadFailures(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
	}{
		{"forbidden", `{"message":"Resource not accessible"}`, "403", 403},
		{"malformed", `not json`, "read repository settings", 200},
		{"missing", `{}`, "missing description", 200},
		{"null boolean", `{"description":null,"homepage":null,"has_issues":null}`, "null for has_issues", 200},
		{"wrong type", `{"description":123}`, "decode description", 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"plan"}, &stdout, &stderr, commandOperations{plan: func() ([]byte, error) {
				return planSettings(settings.RepositoryDeclaration{}, repository.Repository{Owner: "owner", Name: "current"}, testClient(t, test.body, test.status))
			}})
			if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	client, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true, Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("mutation request: %s", req.Method)
		}
		return nil, errors.New("network unavailable")
	})})
	if err != nil {
		t.Fatal(err)
	}
	got, err := planSettings(settings.RepositoryDeclaration{}, repository.Repository{Owner: "owner", Name: "current"}, client)
	if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "network unavailable") {
		t.Fatalf("output=%q error=%v", got, err)
	}
}

func TestPlanOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"plan"}, brokenWriter{}, &stderr, commandOperations{plan: func() ([]byte, error) { return []byte("No managed repository settings.\n"), nil }})
	if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}

func TestPlanCurrentValidatesFirst(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.Mkdir(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir) // No origin or authentication: validation must win over either error.
	for _, test := range []struct{ input, want string }{
		{"settings: {}\n", "missing schemaVersion"},
		{repositoryYAML("has_issues: [\n"), "decode YAML"},
	} {
		if err := os.WriteFile(filepath.Join(dir, settingsPath), []byte(test.input), 0600); err != nil {
			t.Fatal(err)
		}
		data, err := planCurrent()
		if len(data) != 0 || err == nil || !strings.Contains(err.Error(), "validate "+settingsPath) || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("output=%q error=%v", data, err)
		}
	}
}
