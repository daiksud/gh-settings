//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Exercise the shipped composite action and its binary against a fake GitHub API.
func TestBuiltCompositeVariablesTransportPrivacy(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.base["GITHUB_ACTIONS"] = "true"
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	f.write(filepath.Join(f.caller, ".github/settings.yml"), `version: daiksud/gh-settings@v0
repository:
  settings:
    actions: {variables: [{name: KEEP, value: private-actions-new}, {name: CREATE, value: private-actions-create}]}
    agents: {variables: [{name: KEEP, value: private-agents-new}, {name: CREATE, value: private-agents-create}]}
    environments:
      - name: production
        variables: [{name: KEEP, value: private-environment-new}, {name: CREATE, value: private-environment-create}]
`)
	dir, err := os.MkdirTemp("/tmp", "ghs-var-privacy-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "/repos/owner/config"
	bases := map[string]string{prefix + "/actions/variables": "actions", prefix + "/agents/variables": "agents", prefix + "/environments/production/variables": "environment"}
	state := map[string]map[string]string{}
	seen := map[string]map[string]int{}
	for _, family := range bases {
		state[family] = map[string]string{"KEEP": "private-" + family + "-old", "DELETE": "private-" + family + "-delete"}
		seen[family] = map[string]int{}
	}
	var mu sync.Mutex
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token variable-fixture" {
			t.Error("unexpected target or credential")
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == prefix+"/environments/production" && r.Method == "GET" {
			fmt.Fprint(w, `{"id":42,"name":"production"}`)
			return
		}
		base, name := r.URL.Path, ""
		for candidate := range bases {
			if strings.HasPrefix(r.URL.Path, candidate+"/") {
				base = candidate
				name = strings.TrimPrefix(r.URL.Path, candidate+"/")
				break
			}
		}
		family, ok := bases[base]
		if !ok {
			t.Error("unexpected API route")
			w.WriteHeader(400)
			return
		}
		values := state[family]
		kind := r.Method
		if kind == "GET" {
			if name == "" {
				kind = "LIST"
			} else {
				kind = "READ"
			}
		}
		seen[family][kind]++
		switch r.Method {
		case "GET":
			if name != "" {
				if value, present := values[name]; present {
					_ = json.NewEncoder(w).Encode(map[string]string{"name": name, "value": value})
				} else {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"not found"}`)
				}
				return
			}
			names := make([]string, 0, len(values))
			for n := range values {
				names = append(names, n)
			}
			sort.Strings(names)
			items := make([]map[string]string, 0, len(names))
			for _, n := range names {
				items = append(items, map[string]string{"name": n, "value": values[n]})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"total_count": len(items), "variables": items})
		case "POST", "PATCH":
			var payload map[string]string
			if json.NewDecoder(r.Body).Decode(&payload) != nil {
				t.Error("invalid write payload")
				w.WriteHeader(400)
				return
			}
			if r.Method == "POST" {
				name = payload["name"]
				w.WriteHeader(201)
			} else {
				w.WriteHeader(204)
			}
			values[name] = payload["value"]
		case "DELETE":
			delete(values, name)
			w.WriteHeader(204)
		default:
			t.Error("unexpected variable operation")
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
	options := map[string]string{"github.action_ref": version, "inputs.prune": "actions-variables,agents-variables,environment-variables", "inputs.plan-reporting": "status-only", "GH_DEBUG": "api"}
	check := func(r compositeResult) {
		t.Helper()
		if r.status != 0 {
			t.Fatalf("composite command failed with status %d: %s", r.status, regexp.MustCompile(`private-[a-z-]+`).ReplaceAllString(r.log, "[redacted]"))
		}
		output := r.log + r.summary + r.api + r.calls + fmt.Sprint(r.outputs)
		if strings.Contains(output, "private-") {
			t.Fatal("variable value leaked in composite output")
		}
	}
	check(f.run(t, "plan", "variable-fixture", options))
	delete(options, "inputs.plan-reporting")
	check(f.run(t, "apply", "variable-fixture", options))
	mu.Lock()
	defer mu.Unlock()
	for _, family := range bases {
		for _, method := range []string{"LIST", "READ", "POST", "PATCH", "DELETE"} {
			if seen[family][method] == 0 {
				t.Errorf("%s missing %s", family, method)
			}
		}
		if len(state[family]) != 2 || state[family]["KEEP"] != "private-"+family+"-new" || state[family]["CREATE"] != "private-"+family+"-create" {
			t.Errorf("%s final state incorrect", family)
		}
	}
}
