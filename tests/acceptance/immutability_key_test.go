//go:build acceptance

package acceptance

import (
	"net/http"
	"sync/atomic"
	"testing"
)

func TestImmutableReleaseObsoleteKeyFailsBeforeRemoteWork(t *testing.T) {
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
	for _, fields := range []string{"immutable_release: true", "immutable_release: true, enable_release_immutability: true"} {
		input := generalInput("releases: {" + fields + "}\n")
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"validate", "plan", "apply", "export"} {
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || result.runErr == nil || result.stdout.Len() != 0 {
				t.Fatalf("%s accepted obsolete input: %v %#v", command, err, result)
			}
			if requests.Load() != 0 {
				t.Fatalf("%s contacted an API before rejecting obsolete input", command)
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		}
	}
}
