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

func TestArchivedWebhookCLIReadOnly(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, mode := range []string{"matching", "config differs", "created drift", "missing config"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			hookReads, writes := 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "GET" {
					writes++
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, `{"id":42,"full_name":"owner/current","archived":true,"private":false,"visibility":"public","owner":{"id":8,"login":"owner","type":"User"}}`)
				case "/repos/owner/current/hooks/7":
					hookReads++
					created := "2026-11-14T00:00:00Z"
					if mode == "created drift" && hookReads == 2 {
						created = "2026-11-14T01:00:00Z"
					}
					fmt.Fprintf(w, `{"id":7,"name":"web","type":"Repository","url":"https://api.github.com/repos/owner/current/hooks/7","created_at":%q,"active":true,"events":["issues","push"],"config":{"url":"https://example.com/hook","content_type":"json","insecure_ssl":"0","secret":"SECRET_CANARY"}}`, created)
				case "/repos/owner/current/hooks/7/config":
					content := "json"
					if mode == "config differs" && hookReads == 2 {
						content = "form"
					}
					if mode == "missing config" && hookReads == 2 {
						fmt.Fprint(w, `{}`)
						return
					}
					fmt.Fprintf(w, `{"url":"https://example.com/hook","content_type":%q,"insecure_ssl":"0","secret":"SECRET_CANARY"}`, content)
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
					w.WriteHeader(400)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, webhooks: [{id: 7, content_type: application/json, active: true, events: [push, issues]}]}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || (result.runErr == nil) != (mode == "matching") {
				t.Fatalf("outcome %v %v %s", err, result.runErr, result.stderr.String())
			}
			if strings.Contains(result.stdout.String()+result.stderr.String(), "SECRET_CANARY") {
				t.Fatal("secret leaked")
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input || writes != 0 || hookReads != 2 {
				t.Fatalf("declaration/writes/reads: %v %d %d", err, writes, hookReads)
			}
		})
	}
}
