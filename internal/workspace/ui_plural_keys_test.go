package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"strings"
	"testing"
)

func TestUIPluralKeysRejectOldAndMixedNames(t *testing.T) {
	for _, body := range []string{
		"general: {pull_requests: {merge_commit: {}}}",
		"general: {pull_requests: {merge_commit: {}, merge_commits: {}}}",
		"general: {pull_requests: {squash_merge: {}}}",
		"general: {pull_requests: {squash_merge: {}, squash_merging: {}}}",
		"advanced_security: {code_security: {codeql_analysis: {query_suite: extended}}}",
		"advanced_security: {code_security: {codeql_analysis: {query_suite: extended, query_suites: extended}}}",
		"advanced_security: {code_security: {codeql_analysis: {query_suites: [extended]}}}",
		"advanced_security: {code_security: {codeql_analysis: {query_suites: {}}}}",
	} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {" + body + "}}\n")); err == nil {
			t.Fatalf("accepted invalid public keys: %s", body)
		}
	}
}

func TestUIPluralKeysRoundTripScalarAndCommitMenus(t *testing.T) {
	for _, suite := range []string{"default", "extended"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {pull_requests: {allow_merge_commits: true, allow_squash_merging: true, merge_commits: {default_commit_message: default_message}, squash_merging: {default_commit_message: default_message}}}, advanced_security: {code_security: {codeql_analysis: {query_suites: " + suite + "}}}}}\n"
		ui, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil {
			t.Fatal(err)
		}
		if d.Security.CodeQLConfiguration.QuerySuite == nil || *d.Security.CodeQLConfiguration.QuerySuite != suite || d.Repository.MergeCommitTitle == nil || d.Repository.SquashMergeCommitTitle == nil {
			t.Fatal("native values lost")
		}
		data, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"merge_commits:", "squash_merging:", "query_suites: " + suite, "default_commit_message: default_message"} {
			if !strings.Contains(string(data), want) {
				t.Fatalf("missing %s", want)
			}
		}
		for _, old := range []string{"merge_commit:", "squash_merge:", "query_suite:"} {
			if strings.Contains(string(data), old) {
				t.Fatalf("export retained obsolete key %s", old)
			}
		}
	}
}
