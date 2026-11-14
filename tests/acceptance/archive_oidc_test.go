//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestArchivedOIDCCLIReadOnly(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, mode := range []string{"matching", "immutable only", "order differs", "immutable missing", "owner drift", "malformed", "duplicate owner", "duplicate mode"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			repoReads, subjectReads, writes := 0, 0, 0
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
					repoReads++
					owner := 8
					if mode == "owner drift" && repoReads == 3 {
						owner = 9
					}
					if mode == "duplicate owner" && repoReads == 3 {
						fmt.Fprint(w, `{"id":42,"full_name":"owner/current","archived":true,"visibility":"public","owner":{"id":9,"id":8,"type":"User"}}`)
					} else {
						fmt.Fprintf(w, `{"id":42,"full_name":"owner/current","archived":true,"visibility":"public","owner":{"id":%d,"type":"User"}}`, owner)
					}
				case "/repos/owner/current/actions/oidc/customization/sub":
					subjectReads++
					body := `{"use_default":false,"include_claim_keys":["repo","context"],"use_immutable_subject":false}`
					if mode == "order differs" && subjectReads == 2 {
						body = `{"use_default":false,"include_claim_keys":["context","repo"],"use_immutable_subject":false}`
					}
					if mode == "immutable missing" {
						body = `{"use_default":false,"include_claim_keys":["repo","context"]}`
					}
					if mode == "duplicate mode" && subjectReads == 2 {
						body = `{"use_default":true,"use_default":false,"include_claim_keys":["repo","context"],"use_immutable_subject":false}`
					}
					if mode == "malformed" && subjectReads == 2 {
						body = `{"use_default":false}`
					}
					fmt.Fprint(w, body)
				default:
					t.Errorf("unexpected route: %s", r.URL.Path)
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
			selected := "use_default_template: false, subject_claim_template: [repo, context], use_immutable_subject_claim: false"
			if mode == "immutable only" {
				selected = "use_immutable_subject_claim: false"
			}
			input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {oidc: {subject_claim: {" + selected + "}}}}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			want := mode == "matching" || mode == "immutable only"
			if err != nil || (result.runErr == nil) != want {
				t.Fatalf("outcome %v %v %s", err, result.runErr, result.stderr.String())
			}
			expectedReads := 2
			if mode == "duplicate owner" {
				expectedReads = 1
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input || writes != 0 || subjectReads != expectedReads {
				t.Fatalf("declaration/writes/reads: %v %d %d", err, writes, subjectReads)
			}
		})
	}
}
