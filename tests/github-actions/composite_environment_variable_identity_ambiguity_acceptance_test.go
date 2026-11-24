//go:build acceptance

package githubactions

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeEnvironmentVariableParentIdentityAmbiguity(t *testing.T) {
	product := suppliedCompositeProduct(t)
	const value = "composite-identity-private-canary"
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"matching", `{"id":7,"name":"production"}`, true},
		{"duplicate ID", `{"id":8,"id":7,"name":"production"}`, false},
		{"name alias", `{"id":7,"Name":"other","name":"production"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := product.newFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"

			input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    environments:\n      - name: production\n        variables: [{name: FLAG, value: " + value + "}]\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-var-parent-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			parentReads, variableReads, writes := 0, 0, 0
			created := false
			const base = "/repos/owner/config/environments/production"
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token identity-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
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
				w.Header().Set("Content-Type", "application/json")
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
					t.Error("unexpected API route", r.URL.Path)
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
			result := f.run(t, "apply", "identity-fixture", map[string]string{"GH_DEBUG": "api"})
			if (result.status == 0) != tc.valid {
				t.Error("unexpected composite success status")
			}
			if !tc.valid && len(result.outputs) != 0 {
				t.Error("failed composite emitted outputs")
			}
			if strings.Contains(result.log+result.summary+result.api+result.calls+fmt.Sprint(result.outputs), value) {
				t.Error("variable value leaked in composite output")
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
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("configuration or commit changed", err)
			}
		})
	}
}
