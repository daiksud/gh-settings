package actions_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestMetadataRequiresCheckoutIdentity(t *testing.T) {
	for _, test := range []struct {
		name, wantError  string
		checkout, origin bool
	}{
		{"outside checkout", "resolve current repository", false, false},
		{"missing origin", "resolve current repository", true, false},
		{"missing commit", "resolve checkout commit", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Chdir(t.TempDir())
			if test.checkout {
				git(t, "init", "--quiet")
			}
			if test.origin {
				git(t, "remote", "add", "origin", "https://github.com/owner/current.git")
			}
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"metadata"}, &stdout, &stderr, "fixture-version", nil)
			if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.wantError) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func git(t *testing.T, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func TestMetadataRejectsInvalidOutputValues(t *testing.T) {
	for _, test := range []struct{ name, version, origin string }{
		{"empty version", "", "https://github.com/owner/current.git"},
		{"LF version", "fixture\ninjected=value", "https://github.com/owner/current.git"},
		{"CR version", "fixture\rinjected=value", "https://github.com/owner/current.git"},
		{"LF target", "dev", "https://github.com/owner/current%0Ainjected=value.git"},
		{"empty repository", "dev", "https://github.com/owner/.git"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadataCheckout(t)
			git(t, "remote", "set-url", "origin", test.origin)
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"metadata"}, &stdout, &stderr, test.version, nil)
			if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "metadata values must be nonempty single-line values") {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func metadataCheckout(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Chdir(t.TempDir())
	git(t, "init", "--quiet")
	git(t, "remote", "add", "origin", "https://github.com/owner/current.git")
	git(t, "-c", "commit.gpgsign=false", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--quiet", "--allow-empty", "-m", "test: consumer identity")
}

func TestMetadataOutputFailure(t *testing.T) {
	metadataCheckout(t)
	var stderr bytes.Buffer
	status := actions.Run([]string{"metadata"}, failingWriter{}, &stderr, "fixture-version", nil)
	if status != 1 || !strings.Contains(stderr.String(), "write metadata: output unavailable") {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestUnsupportedActionArguments(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{nil, {"unknown"}, {"metadata", "extra"}, {"plan"}} {
		var stdout, stderr bytes.Buffer
		status := actions.Run(args, &stdout, &stderr, "fixture-version", nil)
		if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage: gh settings action <metadata|plan REPORT_PATH>") {
			t.Fatalf("args=%v status=%d stdout=%q stderr=%q", args, status, stdout.String(), stderr.String())
		}
	}
}
