package githubactions

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRExamplePreparesOnlyVerifiedSettingsData(t *testing.T) {
	w, _ := readWorkflow(t, "settings-pr.yml")
	testCandidatePreparation(t, w, false)
}

func TestOperationalPRPreparesOnlyPublicSettingsData(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/settings-pr.yml")
	testCandidatePreparation(t, w, true)
}

func testCandidatePreparation(t *testing.T, w workflow, public bool) {
	t.Helper()
	prepare := findStep(t, w.Jobs["plan"], "candidate")
	repo := "owner/config"
	if public {
		repo = "daiksud/gh-settings"
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	modes := []string{"valid", "symlink", "missing", "moved", "wrong-blob", "oversize", "null-size", "api403", "redirect", "unsafe-root", "unfiltered"}
	if public {
		modes = append(modes, "private", "wrong-repository")
	}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			source, consumer, tools := filepath.Join(root, "source"), filepath.Join(root, "consumer"), filepath.Join(root, "tools")
			for _, path := range []string{source, consumer, tools, filepath.Join(source, ".github")} {
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			base := []string{}
			for _, value := range os.Environ() {
				name, _, _ := strings.Cut(value, "=")
				if !strings.HasPrefix(name, "GIT_") && name != "GH_TOKEN" && name != "GITHUB_TOKEN" && name != "HOME" && name != "XDG_CONFIG_HOME" && name != "PYTHONPATH" {
					base = append(base, value)
				}
			}
			base = append(base, "HOME="+root, "XDG_CONFIG_HOME="+root, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
			git := func(args ...string) string {
				cmd := exec.Command(realGit, args...)
				cmd.Dir = source
				cmd.Env = base
				data, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %s %v", args, data, err)
				}
				return strings.TrimSpace(string(data))
			}
			git("init", "--quiet", "--template=")
			git("config", "uploadpack.allowFilter", "true")
			if mode == "unfiltered" {
				git("config", "uploadpack.allowFilter", "false")
			}
			payload := []byte("schemaVersion: 1\nsettings: {repository: {description: '$(touch PWNED)'}}\n")
			if err := os.WriteFile(filepath.Join(source, "pwn.sh"), []byte("touch PWNED\n"), 0700); err != nil {
				t.Fatal(err)
			}
			if mode == "symlink" {
				if err := os.Symlink("/proc/self/environ", filepath.Join(source, ".github/settings.yml")); err != nil {
					t.Fatal(err)
				}
			} else if mode != "missing" {
				if err := os.WriteFile(filepath.Join(source, ".github/settings.yml"), payload, 0600); err != nil {
					t.Fatal(err)
				}
			}
			git("add", ".")
			git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "candidate data")
			head := git("rev-parse", "HEAD")
			blob := strings.Repeat("b", 40)
			if mode != "missing" {
				blob = git("rev-parse", "HEAD:.github/settings.yml")
			}
			if mode == "moved" {
				head = strings.Repeat("a", 40)
			}
			body := map[string]any{"sha": blob, "encoding": "base64", "size": len(payload), "content": base64.StdEncoding.EncodeToString(payload)}
			if mode == "wrong-blob" {
				body["content"] = base64.StdEncoding.EncodeToString([]byte("not the committed blob"))
			}
			if mode == "oversize" {
				body["size"] = 1024*1024 + 1
			}
			if mode == "null-size" {
				body["size"] = nil
			}
			data, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			response := filepath.Join(root, "response.json")
			if err := os.WriteFile(response, data, 0600); err != nil {
				t.Fatal(err)
			}
			// Only replace network transport: real Git must still produce the
			// filtered commit/tree objects, mode checks and resulting metadata.
			wrapper := `#!/usr/bin/env python3
import os, subprocess, sys
args = sys.argv[1:]
if os.environ['FIXTURE_PUBLIC'] == 'true':
    if any(k in os.environ for k in ('GH_TOKEN', 'GITHUB_TOKEN', 'SSH_ASKPASS', 'GIT_CONFIG_COUNT')) or any('extraheader' in a.lower() for a in args):
        sys.exit(92)
if 'fetch' in args:
    at = args.index('origin')
    args[at:at+2] = [os.environ['FIXTURE_REMOTE'], 'HEAD']
    args = ['-c', 'protocol.file.allow=always'] + args
sys.exit(subprocess.call([os.environ['REAL_GIT'], *args]))
`
			if err := os.WriteFile(filepath.Join(tools, "git"), []byte(wrapper), 0700); err != nil {
				t.Fatal(err)
			}
			pythonTransport := `import io, json, os, urllib.error, urllib.request
class FixtureOpener:
    def open(self, request, timeout):
        public = os.environ['FIXTURE_PUBLIC'] == 'true'
        base = 'https://api.github.com/repos/' + os.environ['GITHUB_REPOSITORY']
        if timeout != 30 or request.get_header('Authorization') != (None if public else 'Bearer dummy-app-token'):
            raise RuntimeError('unexpected request')
        if public and request.full_url == base:
            value = {'id': 1414158260, 'full_name': 'daiksud/gh-settings', 'private': False, 'visibility': 'public'}
            if os.environ['FIXTURE_MODE'] == 'private':
                value.update(private=True, visibility='private')
            if os.environ['FIXTURE_MODE'] == 'wrong-repository':
                value['id'] = 1
            return io.BytesIO(json.dumps(value).encode())
        if request.full_url != base + '/git/blobs/' + os.environ['FIXTURE_BLOB']:
            raise RuntimeError('unexpected URL')
        if os.environ['FIXTURE_MODE'] in ('api403', 'redirect'):
            code = 302 if os.environ['FIXTURE_MODE'] == 'redirect' else 403
            raise urllib.error.HTTPError(request.full_url, code, 'PRIVATE_SERVER_VALUE', {}, None)
        return io.BytesIO(open(os.environ['FIXTURE_RESPONSE'], 'rb').read())
urllib.request.build_opener = lambda *args: FixtureOpener()
`
			if err := os.WriteFile(filepath.Join(tools, "sitecustomize.py"), []byte(pythonTransport), 0600); err != nil {
				t.Fatal(err)
			}
			if mode == "unsafe-root" {
				if err := os.Symlink(source, filepath.Join(consumer, ".github")); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("bash", "--noprofile", "--norc", "-euo", "pipefail", "-c", prepare.Run)
			cmd.Dir = consumer
			publicMode := "false"
			if public {
				publicMode = "true"
			}
			cmd.Env = append(base, "PATH="+tools+":"+os.Getenv("PATH"), "PYTHONPATH="+tools, "REAL_GIT="+realGit, "FIXTURE_PUBLIC="+publicMode, "FIXTURE_REMOTE=file://"+source, "FIXTURE_BLOB="+blob, "FIXTURE_RESPONSE="+response, "FIXTURE_MODE="+mode, "GH_TOKEN=dummy-app-token", "GITHUB_TOKEN=must-not-be-used", "GITHUB_REPOSITORY="+repo, "GITHUB_WORKSPACE="+consumer, "EXPECTED_HEAD="+head, "PR_NUMBER=7")
			output, runErr := cmd.CombinedOutput()
			if strings.Contains(string(output), "dummy-app-token") || strings.Contains(string(output), "PRIVATE_SERVER_VALUE") || strings.Contains(string(output), "touch PWNED") {
				t.Fatal("credential/data leaked", string(output))
			}
			if mode != "valid" {
				if runErr == nil || !strings.Contains(string(output), "candidate settings preparation failed") {
					t.Fatal(mode, runErr, string(output))
				}
				return
			}
			if runErr != nil || len(output) != 0 {
				t.Fatal(runErr, string(output))
			}
			prepared, err := os.ReadFile(filepath.Join(consumer, ".github/settings.yml"))
			if err != nil || !bytes.Equal(prepared, payload) {
				t.Fatal(string(prepared), err)
			}
			for _, path := range []string{"pwn.sh", "PWNED"} {
				if _, err := os.Lstat(filepath.Join(consumer, path)); !os.IsNotExist(err) {
					t.Fatal("candidate code checked out or executed", path, err)
				}
			}
			check := exec.Command(realGit, "rev-parse", "HEAD")
			check.Dir = consumer
			check.Env = base
			actual, err := check.Output()
			if err != nil || strings.TrimSpace(string(actual)) != head {
				t.Fatal("prepared SHA not candidate", string(actual), err)
			}
			config, err := os.ReadFile(filepath.Join(consumer, ".git/config"))
			if err != nil || strings.Contains(string(config), "dummy-app-token") || strings.Contains(string(config), "AUTHORIZATION") {
				t.Fatal("persisted token", err)
			}
		})
	}
}
