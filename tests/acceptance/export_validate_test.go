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
			response: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"id":123}`,
			want: map[string]any{"schemaVersion": 1, "settings": map[string]any{"labels": []any{}, "topics": map[string]any{"names": []any{}}, "repository": map[string]any{
				"description": "Example repository", "homepage": "https://example.com",
				"has_issues": true, "has_projects": false, "has_wiki": true, "allow_merge_commit": false, "allow_squash_merge": false, "allow_rebase_merge": false, "delete_branch_on_merge": false, "allow_auto_merge": false, "allow_update_branch": false, "squash_merge_commit_title": "COMMIT_OR_PR_TITLE", "squash_merge_commit_message": "COMMIT_MESSAGES",
			}}},
		},
		{
			name:     "zero-and-null-strings",
			response: `{"description":null,"homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`,
			want: map[string]any{"schemaVersion": 1, "settings": map[string]any{"labels": []any{}, "topics": map[string]any{"names": []any{}}, "repository": map[string]any{
				"description": "", "homepage": "",
				"has_issues": false, "has_projects": false, "has_wiki": false, "allow_merge_commit": false, "allow_squash_merge": false, "allow_rebase_merge": false, "delete_branch_on_merge": false, "allow_auto_merge": false, "allow_update_branch": false, "squash_merge_commit_title": "COMMIT_OR_PR_TITLE", "squash_merge_commit_message": "COMMIT_MESSAGES",
			}}},
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
			if reads := fixture.reads.Load(); reads != 4 {
				t.Fatalf("export made %d API requests, want 4", reads)
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
			if reads := fixture.reads.Load(); reads != 4 {
				t.Fatalf("validate made %d additional API requests, want 0", reads-4)
			}
			if err := checkSettingsUnchanged(fixture.work, original); err != nil {
				t.Fatal(err)
			}
		})
	}
}
