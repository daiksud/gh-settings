//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var candidateBinary = flag.String("binary", "", "already-built release-version gh-settings executable for the real composite")

func suppliedCompositeProduct(t *testing.T) ([]byte, string) {
	t.Helper()
	if *candidateBinary == "" {
		t.Fatal("composite acceptance requires -binary PATH with a release-version binary")
	}
	product, err := os.ReadFile(*candidateBinary)
	if err != nil {
		t.Fatal(err)
	}
	// Artifact transfer can omit executable bits. Only execute our owned copy;
	// the upstream candidate gate has already checked the supplied bytes.
	copyPath := filepath.Join(t.TempDir(), "gh-settings")
	if err := os.WriteFile(copyPath, product, 0700); err != nil {
		t.Fatal(err)
	}
	versionOutput, err := exec.Command(copyPath, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	return product, strings.TrimSpace(strings.TrimPrefix(string(versionOutput), "gh-settings "))
}

// The real product is supplied, not rebuilt or replaced by the shell mock. Only
// its download is bounded; the actual installer verifies its checksum/version.
func TestBuiltCompositeExport(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"first and repeated export", "invalid file", "settings symlink", "directory symlink", "concurrent edit", "destination write failure", "output failure after save"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			path := filepath.Join(f.caller, ".github/settings.yml")
			const edited = "version: daiksud/gh-settings@v0\nrepository: {about: {description: concurrent}}\n"
			var beforeRead func()
			switch scenario {
			case "first and repeated export", "destination write failure":
				f.git("rm", "--quiet", ".github/settings.yml")
				f.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: first export checkout")
				f.sha = f.git("rev-parse", "HEAD")
				if err := os.RemoveAll(filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
				if scenario == "destination write failure" {
					beforeRead = func() { f.write(filepath.Dir(path), "blocked destination") }
				}
			case "invalid file":
				f.write(path, "invalid: [")
			case "settings symlink", "directory symlink":
				target := path
				outside := t.TempDir()
				if scenario == "settings symlink" {
					outside = filepath.Join(outside, "settings.yml")
					f.write(outside, "version: daiksud/gh-settings@v0\n")
				} else {
					target = filepath.Dir(path)
				}
				if err := os.RemoveAll(target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, target); err != nil {
					t.Fatal(err)
				}
			case "concurrent edit":
				beforeRead = func() { f.write(path, edited) }
			}
			initial, _ := os.ReadFile(path)
			reads := startCompositeExportAPI(t, f, beforeRead)
			extra := map[string]string{"github.action_ref": version, "GH_DEBUG": "api", "GITHUB_TOKEN": "ambient-canary"}
			if scenario == "output failure after save" {
				extra["OUTPUT_DIRECTORY"] = "1"
			}
			result := f.run(t, "export", "export-fixture-read", extra)
			success := scenario == "first and repeated export"
			if success {
				want := map[string]string{"config-sha": f.sha, "target": "owner/config", "cli-version": version, "cli-sha256": f.digest, "config-path": ".github/settings.yml"}
				if result.status != 0 || !reflect.DeepEqual(result.outputs, want) {
					t.Fatalf("real export failed: %+v", result)
				}
			} else if result.status == 0 || len(result.outputs) != 0 {
				t.Fatalf("file/output failure published success: %+v", result)
			}
			if result.summary != "" || len(result.remainingReports) != 0 || strings.Contains(result.log+fmt.Sprint(result.outputs), "CANARY-local-export-value") || strings.Contains(result.log, "ambient-canary") {
				t.Fatal("real export leaked values or rendered a report")
			}
			data, _ := os.ReadFile(path)
			switch scenario {
			case "first and repeated export", "output failure after save":
				if !strings.Contains(string(data), "CANARY-local-export-value") || !strings.Contains(string(data), "Exported repository") {
					t.Fatal("real export did not save independently acquired settings")
				}
			case "invalid file", "settings symlink", "directory symlink":
				if reads.Load() != 0 || string(data) != string(initial) {
					t.Fatal("file guard read the API or changed destination bytes")
				}
			case "concurrent edit":
				if string(data) != edited || reads.Load() == 0 {
					t.Fatal("concurrent consumer edit was overwritten")
				}
			case "destination write failure":
				block, _ := os.ReadFile(filepath.Dir(path))
				if string(block) != "blocked destination" || reads.Load() == 0 {
					t.Fatal("failed write altered the blocking destination")
				}
			}
			if success {
				priorReads := reads.Load()
				again := f.run(t, "export", "export-fixture-read", extra)
				repeated, _ := os.ReadFile(path)
				if again.status != 0 || !reflect.DeepEqual(again.outputs, result.outputs) || string(repeated) != string(data) || reads.Load() <= priorReads {
					t.Fatal("re-export did not reuse the same file and acquisition contract")
				}
			}
			if f.git("rev-parse", "HEAD") != f.sha || f.git("diff", "--cached", "--name-only") != "" {
				t.Fatal("Action committed or staged the exported file")
			}
		})
	}
}

func startCompositeExportAPI(t *testing.T, f *compositeFixture, beforeRead func()) *atomic.Int64 {
	t.Helper()
	state, err := os.MkdirTemp("/tmp", "ghs-export-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(state) })
	listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var reads atomic.Int64
	var once sync.Once
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token export-fixture-read" {
			t.Error("export escaped its explicit target/token boundary")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		query := r.Method == http.MethodPost && r.URL.Path == "/graphql" && strings.Contains(string(body), `"query":"query(`) && !strings.Contains(string(body), "mutation")
		if !query && (r.Method != http.MethodGet || r.URL.Path != "/repos/owner/config" && !strings.HasPrefix(r.URL.Path, "/repos/owner/config/")) {
			t.Error("export attempted a mutation or another target")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		reads.Add(1)
		once.Do(func() {
			if beforeRead != nil {
				beforeRead()
			}
		})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.RequestURI() {
		case "/repos/owner/config":
			fmt.Fprint(w, `{"default_branch":"main","description":"Exported repository","homepage":"","has_issues":true,"has_projects":true,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`)
		case "/repos/owner/config/actions/variables?per_page=30&page=1":
			fmt.Fprint(w, `{"total_count":1,"variables":[{"name":"EXPORT_CANARY","value":"CANARY-local-export-value"}]}`)
		default:
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"message":"CANARY-local-export-value"}`)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	f.write(filepath.Join(state, "config.yml"), "http_unix_socket: "+filepath.Join(state, "api.sock")+"\n")
	f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = state, "1"
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		f.base[key] = "http://127.0.0.1:1"
	}
	f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
	return &reads
}

// Execute the candidate's actual preparation/API/assertion scripts locally with
// the supplied product. Hosted uses expansion remains a separate CI observation.
func TestBuiltCandidateExportFixture(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.git("remote", "set-url", "origin", "https://github.com/example-owner/settings-fixture.git")
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	if err := os.Symlink(root(), filepath.Join(f.caller, ".action")); err != nil {
		t.Fatal(err)
	}
	apiBinary := filepath.Join(t.TempDir(), "plan-fixture")
	build := exec.Command("go", "build", "-o", apiBinary, "./tests/github-actions/plan-fixture")
	build.Dir = root()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture API: %v\n%s", err, output)
	}
	f.base["GITHUB_ENV"] = filepath.Join(t.TempDir(), "environment")
	f.write(f.base["GITHUB_ENV"], "")
	f.base["GITHUB_STEP_SUMMARY"] = filepath.Join(t.TempDir(), "step_summary_fixture")
	run := func(args ...string) string {
		t.Helper()
		command := exec.Command("bash", append([]string{filepath.Join(root(), "tests/github-actions/candidate-plan.sh")}, args...)...)
		command.Dir, command.Env = f.caller, fixtureEnvironment(f.base)
		output, err := command.CombinedOutput()
		if err != nil {
			requests, _ := os.ReadFile(filepath.Join(f.base["PLAN_FIXTURE"], "requests"))
			t.Fatalf("candidate %v failed: %v\n%s\nrequests:\n%s", args, err, output, requests)
		}
		return string(output)
	}
	run("start", apiBinary)
	environment, err := os.ReadFile(f.base["GITHUB_ENV"])
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(environment)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatal("invalid candidate environment")
		}
		f.base[key] = value
	}
	t.Cleanup(func() { run("stop") })
	f.base["ACTION_SHA"] = f.git("-C", root(), "rev-parse", "HEAD")
	f.base["EXPECTED_CLI_VERSION"], f.base["EXPECTED_CLI_SHA256"] = version, f.digest
	for _, scenario := range []string{"first", "repeat", "invalid", "symlink"} {
		run("prepare-export", scenario)
		f.sha = f.git("rev-parse", "HEAD")
		result := f.run(t, "export", "export-fixture-read", map[string]string{"github.action_ref": version, "GH_DEBUG": "api"})
		if strings.Contains(result.log+result.summary+fmt.Sprint(result.outputs), "CANARY-local-export-value") {
			t.Fatal("candidate export leaked its fixture variable")
		}
		f.base["PLAN_OUTCOME"] = "success"
		if result.status != 0 {
			f.base["PLAN_OUTCOME"] = "failure"
		}
		for variable, key := range map[string]string{"CONFIG_PATH": "config-path", "CONFIG_SHA": "config-sha", "TARGET": "target", "CLI_VERSION": "cli-version", "CLI_SHA256": "cli-sha256", "PRIVATE_PLAN_OUTCOME": "plan-outcome"} {
			f.base[variable] = result.outputs[key]
		}
		if output := run("assert-export", scenario); strings.Contains(output, "CANARY-local-export-value") {
			t.Fatal("candidate assertion printed its fixture variable")
		}
	}
}
