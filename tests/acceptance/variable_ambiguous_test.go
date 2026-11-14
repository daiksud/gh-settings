//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestVariableAmbiguousOrdinaryReadFailsWithoutWrite(t *testing.T) {
	const desired = "PRIVATE_VARIABLE_DESIRED_CANARY"
	const remote = "PRIVATE_VARIABLE_REMOTE_CANARY"
	for _, family := range []string{"actions", "agents"} {
		for _, command := range []string{"plan", "apply"} {
			t.Run(family+"/"+command, func(t *testing.T) {
				var mu sync.Mutex
				reads, writes := 0, 0
				fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					if r.Method != http.MethodGet {
						writes++
						w.WriteHeader(400)
						return
					}
					if r.URL.Path != "/repos/owner/current/"+family+"/variables/FLAG" {
						t.Errorf("unexpected path %s", r.URL.Path)
						w.WriteHeader(400)
						return
					}
					reads++
					fmt.Fprintf(w, `{"name":"FLAG","value":%q,"Value":%q}`, remote, desired)
				}, false)
				if err != nil {
					t.Fatal(err)
				}
				defer fixture.close()
				env := append(exportFixtureEnv(fixture.cfg), "GH_DEBUG=api")
				if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
					t.Fatal(err)
				}
				input := repositoryInput("settings: {" + family + ": {variables: [{name: FLAG, value: " + desired + "}]}}\n")
				if err := writeSettingsFile(fixture.work, input); err != nil {
					t.Fatal(err)
				}
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{command}, false)
				if err != nil || result.runErr == nil {
					t.Fatalf("unexpected result %v %v", err, result.runErr)
				}
				if strings.Contains(result.stdout.String(), desired) || strings.Contains(result.stderr.String(), desired) || strings.Contains(result.stderr.String(), remote) {
					t.Fatal("variable value leaked")
				}
				mu.Lock()
				defer mu.Unlock()
				if reads != 1 || writes != 0 {
					t.Fatalf("reads=%d writes=%d", reads, writes)
				}
			})
		}
	}
}
