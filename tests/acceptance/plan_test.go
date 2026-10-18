//go:build acceptance

package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const planResponse = `{"description":"old","homepage":"old-url","has_issues":true,"has_projects":true,"has_wiki":true}`

func TestPlanAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, input, body, want, wantError              string
		status                                          int
		noRead, outputFailure, nested, noOrigin, noAuth bool
	}{
		{name: "partial-zero-values", input: repositoryInput("description: ''\nhomepage: ''\nhas_issues: false\n"), want: "settings.repository.description: \"old\" → \"\"\nsettings.repository.homepage: \"old-url\" → \"\"\nsettings.repository.has_issues: true → false\n", nested: true},
		{name: "no-differences", input: repositoryInput("has_wiki: true\n"), want: "No differences for managed repository settings.\n"},
		{name: "all-fields", input: repositoryInput("has_wiki: false\nhas_projects: false\nhas_issues: false\nhomepage: new-url\ndescription: new\n"), want: "settings.repository.description: \"old\" → \"new\"\nsettings.repository.homepage: \"old-url\" → \"new-url\"\nsettings.repository.has_issues: true → false\nsettings.repository.has_projects: true → false\nsettings.repository.has_wiki: true → false\n"},
		{name: "no-managed-fields", input: "schemaVersion: 1\n", want: "No managed repository settings.\n", noRead: true, noOrigin: true, noAuth: true},
		{name: "empty-settings", input: "schemaVersion: 1\nsettings: {}\n", want: "No managed repository settings.\n", noRead: true, noOrigin: true, noAuth: true},
		{name: "empty-repository", input: "schemaVersion: 1\nsettings:\n  repository: {}\n", want: "No managed repository settings.\n", noRead: true, noOrigin: true, noAuth: true},
		{name: "null-strings", input: repositoryInput("description: ''\nhomepage: ''\n"), body: `{"description":null,"homepage":null,"has_issues":true,"has_projects":true,"has_wiki":true}`, want: "No differences for managed repository settings.\n"},
		{name: "escaped-strings", input: repositoryInput("description: \"new\\n\\\"\\\\\\u001b\"\n"), body: `{"description":"old\n\"\\\u001b","homepage":"","has_issues":true,"has_projects":true,"has_wiki":true}`, want: "settings.repository.description: \"old\\n\\\"\\\\^[\" → \"new\\n\\\"\\\\\\x1b\"\n"},
		{name: "malformed-before-api", input: repositoryInput("has_issues: [\n"), wantError: "validate .github/settings.yml: decode YAML", noRead: true},
		{name: "invalid-before-origin", input: repositoryInput("has_issues: 'false'\n"), wantError: "must be a boolean", noRead: true, noOrigin: true},
		{name: "missing-file", wantError: "read .github/settings.yml", noRead: true},
		{name: "missing-origin", input: repositoryInput("has_issues: false\n"), wantError: "a GitHub origin remote is required", noRead: true, noOrigin: true},
		{name: "api-failure", input: repositoryInput("has_issues: false\n"), body: `{"message":"Resource not accessible"}`, status: 403, wantError: "403"},
		{name: "decode-failure", input: repositoryInput("has_issues: false\n"), body: `{}`, wantError: "missing description"},
		{name: "output-failure", input: repositoryInput("has_issues: false\n"), outputFailure: true, wantError: "write output:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := test.body
			if body == "" {
				body = planResponse
			}
			fixture, err := startAPIFixture(body, test.status)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if test.noAuth {
				env = fixtureEnv(fixture.cfg)
			}
			if test.noOrigin {
				err = initGitCheckout(fixture.work, env)
			} else {
				err = setupOrdinaryOrigin(fixture.work, env)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := writeSettingsFile(fixture.work, test.input); err != nil {
				t.Fatal(err)
			}
			work := fixture.work
			if test.nested {
				work = filepath.Join(work, "nested")
				if err := os.Mkdir(work, 0700); err != nil {
					t.Fatal(err)
				}
			}
			result, err := executeCLI(binary, work, fixture.root, env, []string{"plan"}, test.outputFailure)
			if err != nil {
				t.Fatal(err)
			}
			if err := fixture.checkRequests(test.noRead); err != nil {
				t.Fatal(err)
			} // Reject every method except GET, including PATCH.
			wantReads := int32(1)
			if test.noRead {
				wantReads = 0
			}
			if reads := fixture.reads.Load(); reads != wantReads {
				t.Fatalf("reads=%d want=%d", reads, wantReads)
			}
			if test.wantError == "" {
				if result.runErr != nil || result.stdout.String() != test.want || result.stderr.Len() != 0 {
					t.Fatalf("execution=%v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
				}
			} else {
				exit, ok := result.runErr.(*exec.ExitError)
				if !ok || exit.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), test.wantError) {
					t.Fatalf("execution=%v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
				}
			}
			if err := checkSettingsUnchanged(fixture.work, test.input); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExportEditValidatePlan(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture(planResponse, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	exported, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if exported.runErr != nil || exported.stdout.Len() != 0 || exported.stderr.Len() != 0 {
		t.Fatalf("export: %v stderr=%q", exported.runErr, exported.stderr.String())
	}
	// Edit the exported YAML's issues value and preserve all other fields.
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(data)
	edited := strings.Replace(original, "has_issues: true", "has_issues: false", 1)
	if original == edited {
		t.Fatal("export did not contain expected issues field")
	}
	if err := writeSettingsFile(fixture.work, edited); err != nil {
		t.Fatal(err)
	}
	validated, err := executeCLI(binary, fixture.work, fixture.root, fixtureEnv(fixture.cfg), []string{"validate"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if validated.runErr != nil || validated.stdout.Len() != 0 || validated.stderr.Len() != 0 || fixture.reads.Load() != 4 {
		t.Fatalf("validate: %v stdout=%q stderr=%q reads=%d", validated.runErr, validated.stdout.String(), validated.stderr.String(), fixture.reads.Load())
	}
	planned, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"plan"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if planned.runErr != nil || planned.stdout.String() != "settings.repository.has_issues: true → false\n" || planned.stderr.Len() != 0 || fixture.reads.Load() != 8 {
		t.Fatalf("plan: %v stdout=%q stderr=%q reads=%d", planned.runErr, planned.stdout.String(), planned.stderr.String(), fixture.reads.Load())
	}
	if err := fixture.checkRequests(false); err != nil {
		t.Fatal(err)
	}
	if err := checkSettingsUnchanged(fixture.work, edited); err != nil {
		t.Fatal(err)
	}
}
