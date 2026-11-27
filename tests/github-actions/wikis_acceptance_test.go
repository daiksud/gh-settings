//go:build acceptance

package githubactions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBuiltCompositeWikis(t *testing.T) {
	f := suppliedCompositeProduct(t).newFixture(t)
	const base = "/repos/owner/config"
	const token = "wiki-fixture"
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
		case r.Method == http.MethodPatch && r.URL.Path == base:
			var patch map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil || !reflect.DeepEqual(patch, map[string]any{"has_wiki": true}) {
				t.Errorf("unexpected wiki edit %v: %v", patch, err)
			}
			enabled.Store(true)
			writes.Add(1)
			fallthrough
		case r.Method == http.MethodGet && r.URL.Path == base:
			fmt.Fprintf(w, `{"visibility":"public","default_branch":"main","description":"","homepage":"","has_wiki":%t,"has_issues":true,"has_projects":false,"has_discussions":false,"is_template":false,"web_commit_signoff_required":false}`, enabled.Load())
		case exporting.Load() && (r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, base+"/") || r.Method == http.MethodPost && r.URL.Path == "/graphql"):
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"message":"independent setting unavailable"}`)
		default:
			t.Error("unexpected request", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	file := filepath.Join(f.caller, ".github/settings.yml")
	const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {wikis: %s}}}}\n"
	f.write(file, fmt.Sprintf(input, "{enabled: true, restrict_editing_to_collaborators_only: true}"))
	rejected := f.run(t, "plan", token, nil)
	if rejected.status == 0 || requests.Load() != 0 || !strings.Contains(rejected.summary, "repository.settings.general.features.wikis: configuration validation failed") {
		t.Fatal("reserved wiki setting reached the API or lost its location", rejected)
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
	if result := run("plan"); !strings.Contains(result.summary, "No managed settings.") || requests.Load() != 0 {
		t.Fatal("empty Wikis group acquired settings", result)
	}
	exporting.Store(true)
	run("export")
	exporting.Store(false)
	exported, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(exported), "wikis:\n                    enabled: false") || strings.Contains(string(exported), "restrict_editing_to_collaborators_only") {
		t.Fatalf("exported Wikis: %s, %v", exported, err)
	}
	run("validate")
	f.write(file, fmt.Sprintf(input, "{enabled: true}"))
	if result := run("plan"); !strings.Contains(result.summary, "repository.settings.general.features.wikis.enabled: false → true") {
		t.Fatal(result)
	}
	run("apply")
	run("apply")
	if result := run("plan"); !strings.Contains(result.summary, "No differences for managed settings.") || writes.Load() != 1 || !enabled.Load() {
		t.Fatal("repeat wiki apply failed to converge", result, writes.Load())
	}
}
