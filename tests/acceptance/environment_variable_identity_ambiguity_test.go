//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestEnvironmentVariableParentIdentityAmbiguity(t *testing.T) {
	const value = "parent-identity-private-canary"
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"matching", `{"id":7,"name":"production"}`, true},
		{"duplicate ID", `{"id":8,"id":7,"name":"production"}`, false},
		{"ID alias", `{"ID":8,"id":7,"name":"production"}`, false},
		{"duplicate name", `{"id":7,"name":"other","name":"production"}`, false},
		{"name alias", `{"id":7,"Name":"other","name":"production"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const base = "/repos/owner/current/environments/production"
			var mu sync.Mutex
			parentReads, variableReads, writes := 0, 0, 0
			created := false
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "POST" && r.URL.Path == base+"/variables" && tc.name == "duplicate ID" {
					writes++
					created = true
					w.WriteHeader(http.StatusCreated)
					return
				}
				if r.Method != "GET" {
					writes++
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case base:
					parentReads++
					fmt.Fprint(w, tc.body)
				case base + "/variables/FLAG":
					variableReads++
					if tc.name == "duplicate ID" && !created {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"message":"not found"}`)
						return
					}
					fmt.Fprintf(w, `{"name":"FLAG","value":%q}`, value)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(400)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := append(exportFixtureEnv(fixture.cfg), "GH_DEBUG=api")
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    environments:\n      - name: production\n        variables: [{name: FLAG, value: " + value + "}]\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if (result.runErr == nil) != tc.valid {
				t.Error("unexpected command success status")
			}
			if strings.Contains(result.stdout.String()+result.stderr.String(), value) {
				t.Error("variable value leaked in command output")
			}
			mu.Lock()
			if parentReads == 0 {
				t.Error("parent identity endpoint was not reached")
			}
			if tc.valid && variableReads == 0 {
				t.Error("matching parent did not reach variable read")
			}
			if !tc.valid && variableReads != 0 {
				t.Error("ambiguous parent reached variable endpoint")
			}
			if writes != 0 {
				t.Error("unexpected API write")
			}
			mu.Unlock()
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input {
				t.Fatal("settings changed", err)
			}
		})
	}
}
