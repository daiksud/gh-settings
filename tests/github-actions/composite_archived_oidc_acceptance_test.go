//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestBuiltCompositeArchivedOIDC(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, mode := range []string{"matching", "immutable only", "order differs", "immutable missing", "owner drift", "malformed", "duplicate owner", "duplicate mode"} {
		t.Run(mode, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			selected := "use_default_template: false, subject_claim_template: [repo, context], use_immutable_subject_claim: false"
			if mode == "immutable only" {
				selected = "use_immutable_subject_claim: false"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {oidc: {subject_claim: {" + selected + "}}}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-oidc-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			repoReads, subjectReads, writes := 0, 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				if r.Method != "GET" {
					writes++
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/config":
					repoReads++
					owner := 2
					if mode == "owner drift" && repoReads == 3 {
						owner = 3
					}
					if mode == "duplicate owner" && repoReads == 3 {
						fmt.Fprint(w, `{"id":1,"full_name":"owner/config","archived":true,"visibility":"public","owner":{"id":9,"id":2,"type":"User"}}`)
					} else {
						fmt.Fprintf(w, `{"id":1,"full_name":"owner/config","archived":true,"visibility":"public","owner":{"id":%d,"type":"User"}}`, owner)
					}
				case "/repos/owner/config/actions/oidc/customization/sub":
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
					t.Error("unexpected route", r.URL.Path)
					w.WriteHeader(400)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			result := f.run(t, "apply", "archive-fixture", map[string]string{"github.action_ref": version})
			want := mode == "matching" || mode == "immutable only"
			if (result.status == 0) != want || (result.status != 0 && len(result.outputs) != 0) {
				t.Fatalf("unexpected result: %+v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			expectedReads := 2
			if mode == "duplicate owner" {
				expectedReads = 1
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha || writes != 0 || subjectReads != expectedReads {
				t.Fatalf("declaration/writes/reads: %v %d %d", err, writes, subjectReads)
			}
		})
	}
}
