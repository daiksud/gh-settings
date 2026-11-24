//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeVariableInventoryAmbiguity(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	const canary = "PRIVATE_VARIABLE_INVENTORY_CANARY"
	for _, family := range []string{"actions", "agents", "environment"} {
		for _, stage := range []string{"preflight", "readback"} {
			for _, ambiguity := range []string{"duplicate", "alias", "next", "prev-only"} {
				t.Run(family+"/"+stage+"/"+ambiguity, func(t *testing.T) {
					f := newCompositeFixture(t)
					f.base["GITHUB_ACTIONS"] = "true"
					f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
					f.write(f.base["MOCK_ASSET"], string(product))
					f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
					base, scope := "/repos/owner/config/"+family+"/variables", family+"-variables"
					settings := family + ": {variables: []}"
					if family == "environment" {
						base, scope = "/repos/owner/config/environments/production/variables", "environment-variables"
						settings = "environments: [{name: production, variables: []}]"
					}
					input := "version: daiksud/gh-settings@v0\nrepository: {settings: {" + settings + "}}\n"
					config := filepath.Join(f.caller, ".github/settings.yml")
					f.write(config, input)
					dir, err := os.MkdirTemp("/tmp", "ghs-inventory-")
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = os.RemoveAll(dir) })
					socket := filepath.Join(dir, "api.sock")
					listener, err := net.Listen("unix", socket)
					if err != nil {
						t.Fatal(err)
					}
					var mu sync.Mutex
					lists, reads, deletes, unexpected := 0, 0, 0, 0
					server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mu.Lock()
						defer mu.Unlock()
						if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token variable-fixture" {
							unexpected++
							w.WriteHeader(400)
							return
						}
						if family == "environment" && r.Method == http.MethodGet && r.URL.Path == "/repos/owner/config/environments/production" {
							fmt.Fprint(w, `{"id":42,"name":"production"}`)
							return
						}
						switch {
						case r.Method == http.MethodGet && r.URL.Path == base:
							lists++
							if stage == "preflight" || deletes > 0 {
								if ambiguity == "duplicate" {
									fmt.Fprintf(w, `{"total_count":1,"total_count":0,"variables":[{"name":"OLD","value":%q}],"variables":[]}`, canary)
								} else if ambiguity == "alias" {
									fmt.Fprintf(w, `{"total_count":1,"TOTAL_COUNT":0,"variables":[{"name":"OLD","value":%q}],"VARIABLES":[]}`, canary)
								} else if ambiguity == "next" {
									w.Header().Add("Link", `<https://example.invalid/next>; rel="next"`)
									fmt.Fprint(w, `{"total_count":0,"variables":[]}`)
								} else {
									w.Header().Add("Link", `<https://example.invalid/prev>; rel="prev"`)
									entries := make([]string, 30)
									for i := range entries {
										entries[i] = fmt.Sprintf(`{"name":"ITEM_%d","value":%q}`, i, canary)
									}
									fmt.Fprint(w, `{"total_count":31,"variables":[`+strings.Join(entries, ",")+`]}`)
								}
							} else {
								fmt.Fprintf(w, `{"total_count":1,"variables":[{"name":"OLD","value":%q}]}`, canary)
							}
						case r.Method == http.MethodGet && r.URL.Path == base+"/OLD":
							reads++
							fmt.Fprintf(w, `{"name":"OLD","value":%q}`, canary)
						case r.Method == http.MethodDelete && r.URL.Path == base+"/OLD":
							deletes++
							w.WriteHeader(http.StatusNoContent)
						default:
							unexpected++
							w.WriteHeader(http.StatusBadRequest)
						}
					})}
					go func() { _ = server.Serve(listener) }()
					t.Cleanup(func() { _ = server.Close() })
					f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+socket+"\n")
					f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
					for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
						f.base[key] = "http://127.0.0.1:1"
					}
					f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
					result := f.run(t, "apply", "variable-fixture", map[string]string{"github.action_ref": version, "inputs.prune": scope, "GH_DEBUG": "api"})
					output := result.log + result.summary + result.api + result.calls + fmt.Sprint(result.outputs)
					if strings.Contains(output, canary) {
						t.Fatal("variable value leaked")
					}
					if result.status == 0 {
						t.Fatal("ambiguous inventory accepted")
					}
					if stage == "readback" && !strings.Contains(output, "write may have applied") {
						t.Fatal("post-delete uncertainty not reported")
					}
					after, err := os.ReadFile(config)
					if err != nil || string(after) != input {
						t.Fatalf("configuration changed: %v", err)
					}
					mu.Lock()
					defer mu.Unlock()
					if unexpected != 0 || stage == "preflight" && (lists != 1 || reads != 0 || deletes != 0) || stage == "readback" && (lists < 3 || reads != 1 || deletes != 1) {
						t.Fatalf("fixture did not reach expected boundary: lists=%d reads=%d deletes=%d unexpected=%d", lists, reads, deletes, unexpected)
					}
				})
			}
		}
	}
}
