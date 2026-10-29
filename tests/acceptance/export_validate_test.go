//go:build acceptance

package acceptance

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestExportThenValidate(t *testing.T) {
	path := acceptanceBinary(t)
	for _, test := range []struct {
		name, response string
		want           map[string]any
	}{
		{
			name:     "ordinary-values",
			response: `{"default_branch":"release/v2","description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":true,"is_template":true,"id":123}`,
			want: map[string]any{
				"version": "daiksud/gh-settings@v0",
				"repository": map[string]any{
					"about":  map[string]any{"description": "Example repository", "website": "https://example.com", "topics": []any{}},
					"labels": []any{},
					"settings": map[string]any{"general": map[string]any{
						"template_repository": true, "default_branch": "release/v2",
						"features":      map[string]any{"wikis": true, "issues": true, "sponsorships": false, "discussions": false, "projects": false, "pull_requests": false, "issue_permissions": map[string]any{"creation_allowed_by": "all_users"}},
						"pull_requests": map[string]any{"allow_merge_commits": false, "allow_squash_merging": false, "allow_rebase_merging": false, "always_suggest_updating_pull_request_branches": false, "allow_auto_merge": false, "automatically_delete_head_branches": false},
						"commits":       map[string]any{"require_contributors_to_sign_off_on_web_based_commits": true},
					}},
				},
			},
		},
		{
			name:     "zero-and-null-strings",
			response: `{"default_branch":"main","description":null,"homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`,
			want: map[string]any{
				"version": "daiksud/gh-settings@v0",
				"repository": map[string]any{
					"about":  map[string]any{"description": "", "website": "", "topics": []any{}},
					"labels": []any{},
					"settings": map[string]any{"general": map[string]any{
						"template_repository": false, "default_branch": "main",
						"features":      map[string]any{"wikis": false, "issues": false, "sponsorships": false, "discussions": false, "projects": false, "pull_requests": false},
						"pull_requests": map[string]any{"allow_merge_commits": false, "allow_squash_merging": false, "allow_rebase_merging": false, "always_suggest_updating_pull_request_branches": false, "allow_auto_merge": false, "automatically_delete_head_branches": false},
						"commits":       map[string]any{"require_contributors_to_sign_off_on_web_based_commits": false},
					}},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, err := startAPIFixture(test.response, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()

			exportEnv := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, exportEnv); err != nil {
				t.Fatal(err)
			}
			exported, err := executeCLI(path, fixture.work, fixture.root, exportEnv, []string{"export"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if exported.runErr != nil || exported.stdout.Len() != 0 || exported.stderr.Len() != 0 {
				t.Fatalf("export: %v, stderr=%q", exported.runErr, exported.stderr.String())
			}
			if err := fixture.checkRequests(false); err != nil {
				t.Fatal(err)
			}
			if reads := fixture.reads.Load(); reads != 7 {
				t.Fatalf("export made %d API requests, want 7", reads)
			}

			var actual map[string]any
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(data, &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, test.want) {
				t.Fatalf("exported YAML=%#v, want %#v", actual, test.want)
			}

			original := string(data)
			validated, err := executeCLI(path, fixture.work, fixture.root, fixtureEnv(fixture.cfg), []string{"validate"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if validated.runErr != nil || validated.stdout.Len() != 0 || validated.stderr.Len() != 0 {
				t.Fatalf("validate: %v, stdout=%q stderr=%q", validated.runErr, validated.stdout.String(), validated.stderr.String())
			}
			if err := fixture.checkRequests(false); err != nil {
				t.Fatal(err)
			}
			if reads := fixture.reads.Load(); reads != 7 {
				t.Fatalf("validate made %d additional API requests, want 0", reads-7)
			}
			if err := checkSettingsUnchanged(fixture.work, original); err != nil {
				t.Fatal(err)
			}
		})
	}
}
