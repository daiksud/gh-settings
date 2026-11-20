//go:build acceptance

package acceptance

import (
	"net/http"
	"sync/atomic"
	"testing"
)

func TestUIPluralOldKeysFailBeforeRemoteWork(t *testing.T) {
	binary := acceptanceBinary(t)
	var requests atomic.Int64
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) }, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		"general: {pull_requests: {merge_commit: {}}}",
		"general: {pull_requests: {merge_commit: {}, merge_commits: {}}}",
		"general: {pull_requests: {squash_merge: {}}}",
		"general: {pull_requests: {squash_merge: {}, squash_merging: {}}}",
		"advanced_security: {code_security: {codeql_analysis: {query_suite: extended}}}",
		"advanced_security: {code_security: {codeql_analysis: {query_suite: extended, query_suites: extended}}}",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {" + body + "}}\n"
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"validate", "plan", "apply", "export"} {
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || result.runErr == nil || result.stdout.Len() != 0 {
				t.Fatalf("%s accepted obsolete input: %v", command, err)
			}
			if requests.Load() != 0 {
				t.Fatalf("%s contacted API", command)
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		}
	}
}
