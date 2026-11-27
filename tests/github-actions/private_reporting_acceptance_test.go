//go:build acceptance

package githubactions

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBuiltCompositePrivateReporting(t *testing.T) {
	f := suppliedCompositeProduct(t).newFixture(t)
	const base = "/repos/owner/config"
	const token = "reporting-fixture"
	var enabled, exporting atomic.Bool
	var requests, writes atomic.Int64
	serveCompositeAPI(t, f, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token "+token {
			t.Error("unexpected target or credential")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			fmt.Fprint(w, `{"visibility":"public"}`)
		case r.URL.Path == base+"/private-vulnerability-reporting" && r.Method == http.MethodGet:
			fmt.Fprintf(w, `{"enabled":%t}`, enabled.Load())
		case r.URL.Path == base+"/private-vulnerability-reporting" && r.Method == http.MethodPut:
			if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
				t.Error("expected a bodyless enablement request", err)
			}
			enabled.Store(true)
			writes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		case exporting.Load() && (r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, base+"/") || r.Method == http.MethodPost && r.URL.Path == "/graphql"):
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"message":"independent setting unavailable"}`)
		default:
			t.Error("unexpected request", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	file := filepath.Join(f.caller, ".github/settings.yml")
	const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {private_vulnerability_reporting: %s}}}\n"
	f.write(file, fmt.Sprintf(input, "{enabled: true, trusted_reporters: []}"))
	rejected := f.run(t, "apply", token, nil)
	if rejected.status == 0 || len(rejected.outputs) != 0 || requests.Load() != 0 {
		t.Fatal("reserved settings reached the API or published success", rejected)
	}
	rejected = f.run(t, "plan", token, nil)
	if rejected.status == 0 || requests.Load() != 0 || !strings.Contains(rejected.summary, "repository.settings.advanced_security.private_vulnerability_reporting: configuration validation failed") {
		t.Fatal("invalid reporting group lost its location in the summary", rejected)
	}
	f.write(file, fmt.Sprintf(input, "{}"))
	run := func(command string) compositeResult {
		t.Helper()
		result := f.run(t, command, token, nil)
		if result.status != 0 {
			t.Fatalf("%s: %+v", command, result)
		}
		return result
	}
	run("validate")
	if result := run("plan"); !strings.Contains(result.summary, "No managed settings.") || requests.Load() != 0 {
		t.Fatal("empty group acquired reporting", result)
	}
	exporting.Store(true)
	run("export")
	exporting.Store(false)
	exported, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(exported), "private_vulnerability_reporting:\n                enabled: false") {
		t.Fatalf("exported declaration: %s, %v", exported, err)
	}
	run("validate")
	if result := run("plan"); !strings.Contains(result.summary, "No differences for managed settings.") {
		t.Fatal(result)
	}
	f.write(file, fmt.Sprintf(input, "{enabled: true}"))
	if result := run("plan"); !strings.Contains(result.summary, "repository.settings.advanced_security.private_vulnerability_reporting.enabled: false → true") {
		t.Fatal(result)
	}
	run("apply")
	run("apply")
	if result := run("plan"); !strings.Contains(result.summary, "No differences for managed settings.") || writes.Load() != 1 || !enabled.Load() {
		t.Fatal("repeat apply failed to converge", result, writes.Load())
	}
}
