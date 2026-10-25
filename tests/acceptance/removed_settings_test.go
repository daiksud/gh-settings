//go:build acceptance

package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemovedOrganizationSettingsRejectBeforeRemoteWork(t *testing.T) {
	fixture, err := startAPIFixture(planResponse, 200)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"custom_properties: {}", "custom_properties: null", "custom_properties: {service: api}", "repository: {allow_forking: false}", "repository: {allow_forking: true}"} {
		config := "schemaVersion: 1\nsettings:\n  " + body + "\n"
		if err := writeSettingsFile(fixture.work, config); err != nil {
			t.Fatal(err)
		}
		before := fixture.reads.Load()
		for _, command := range []string{"validate", "plan", "apply"} {
			r, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || r.runErr == nil || r.stdout.Len() != 0 || !strings.Contains(r.stderr.String(), "unknown") {
				t.Fatalf("%s %s: error=%v stderr=%s", command, body, err, r.stderr.String())
			}
		}
		for _, option := range []string{"--forking", "--custom-properties", "--custom-property"} {
			r, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export", option}, false)
			if err != nil || r.runErr == nil || r.stdout.Len() != 0 || r.stderr.Len() == 0 {
				t.Fatalf("%s: error=%v stderr=%s", option, err, r.stderr.String())
			}
		}
		after, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
		if err != nil || string(after) != config || fixture.reads.Load() != before {
			t.Fatalf("removed settings touched file or API: %v", err)
		}
	}
}
