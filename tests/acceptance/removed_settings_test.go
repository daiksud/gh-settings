//go:build acceptance

package acceptance

import (
	"strings"
	"testing"
)

func TestRemovedOrganizationSettingsRejectBeforeRemoteWork(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture(planResponse, 200)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"custom_properties: {}", "custom_properties: null", "custom_properties: {service: api}", "general: {allow_forking: false}", "general: {allow_forking: true}"} {
		config := repositoryInput("settings:\n  " + body + "\n")
		unknownKey := "custom_properties"
		if strings.HasPrefix(body, "general:") {
			unknownKey = "allow_forking"
		}
		if err := writeSettingsFile(fixture.work, config); err != nil {
			t.Fatal(err)
		}
		before := fixture.reads.Load()
		for _, command := range []string{"validate", "plan", "apply"} {
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || r.runErr == nil || r.stdout.Len() != 0 || (!strings.Contains(withoutExportInfo(r.stderr.String()), "unknown") || !strings.Contains(withoutExportInfo(r.stderr.String()), unknownKey)) {
				t.Fatalf("%s %s: error=%v stderr=%s", command, body, err, r.stderr.String())
			}
		}
		if err := checkSettingsUnchanged(fixture.work, config); err != nil || fixture.reads.Load() != before {
			t.Fatalf("removed settings touched file or API: %v", err)
		}
	}
}
