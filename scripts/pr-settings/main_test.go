package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct {
	o                       observer
	root, temporary, source string
	s                       snapshot
	routes                  map[string]any
	requests                []string
	body                    []byte
}

func command(t *testing.T, dir string, input []byte, args ...string) []byte {
	t.Helper()
	options := []string{"-c", "core.hooksPath=/dev/null", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}
	cmd := exec.Command("git", append(options, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	cmd.Stdin = strings.NewReader(string(input))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture Git %v: %v %s", args, err, output)
	}
	return output
}

func newFixture(t *testing.T, kind string) *fixture {
	t.Helper()
	f := &fixture{root: t.TempDir(), temporary: t.TempDir(), source: t.TempDir(), routes: map[string]any{}}
	g := func(args ...string) string { return strings.TrimSpace(string(command(t, f.source, nil, args...))) }
	write := func(name, content string) {
		p := filepath.Join(f.source, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	g("init", "--quiet", "--initial-branch=main")
	write("README.md", "base")
	write(".github/workflows/poison.yml", "run: touch CANARY-executed")
	write("poison.sh", "#!/bin/sh\ntouch CANARY-executed\n")
	write(".gitattributes", "* filter=CANARY")
	g("add", ".")
	g("commit", "--quiet", "-m", "fixture base")
	g("checkout", "--quiet", "-b", "candidate")
	f.body = []byte("schemaVersion: 1\nsettings: {repository: {description: 'CANARY-$(touch injected)'}}\n")
	switch kind {
	case "directory symlink":
		g("rm", "-r", "--quiet", ".github")
		if err := os.Symlink("../outside", filepath.Join(f.source, ".github")); err != nil {
			t.Fatal(err)
		}
	case "file symlink":
		if err := os.Symlink("../../outside", filepath.Join(f.source, ".github/settings.yml")); err != nil {
			t.Fatal(err)
		}
	case "missing":
	case "oversize":
		f.body = []byte(strings.Repeat("x", maxConfig+1))
		write(".github/settings.yml", string(f.body))
	default:
		write(".github/settings.yml", string(f.body))
	}
	g("add", ".")
	if kind == "submodule" {
		g("update-index", "--add", "--cacheinfo", "160000,"+g("rev-parse", "main")+",.github/settings.yml")
	}
	g("commit", "--quiet", "--allow-empty", "-m", "fixture candidate")
	head := g("rev-parse", "HEAD")
	tree := g("rev-parse", "HEAD^{tree}")
	g("checkout", "--quiet", "main")
	g("commit", "--quiet", "--allow-empty", "-m", "base advanced")
	base := g("rev-parse", "HEAD")
	merge := strings.TrimSpace(string(command(t, f.source, []byte("fixture merge\n"), "commit-tree", tree, "-p", base, "-p", head)))
	f.s = snapshot{RepositoryID: targetID, Number: 12, Head: head, Base: base, Merge: merge}
	f.routes[""] = map[string]any{"id": targetID, "full_name": target, "visibility": "public"}
	f.routes["/pulls/12"] = map[string]any{"number": 12, "state": "open", "mergeable": true, "merge_commit_sha": merge, "base": map[string]any{"ref": "main", "sha": base, "repo": map[string]any{"id": targetID, "full_name": target}}, "head": map[string]any{"sha": head, "repo": map[string]any{"id": targetID, "full_name": target}}}
	f.routes["/git/ref/heads/main"] = map[string]any{"object": map[string]any{"sha": base, "type": "commit"}}
	if kind != "file symlink" && kind != "directory symlink" && kind != "missing" && kind != "submodule" {
		oid := blobID(f.body)
		f.routes["/git/blobs/"+oid] = map[string]any{"sha": oid, "size": len(f.body), "encoding": "base64", "content": base64.StdEncoding.EncodeToString(f.body)}
	}
	f.o.get = func(_ context.Context, route string, destination any) error {
		f.requests = append(f.requests, route)
		body, ok := f.routes[route]
		if !ok {
			return fmt.Errorf("CANARY-response-unavailable")
		}
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, destination)
	}
	f.o.fetch = func(ctx context.Context, stage, sha string) error {
		if sha != merge {
			return fmt.Errorf("wrong fetched SHA")
		}
		objects := g("cat-file", "--batch-all-objects", "--batch-check=%(objectname) %(objecttype)")
		for line := range strings.SplitSeq(objects, "\n") {
			parts := strings.Fields(line)
			if len(parts) != 2 {
				t.Fatal(line)
			}
			if parts[1] == "blob" {
				continue
			}
			data := command(t, f.source, nil, "cat-file", parts[1], parts[0])
			actual := strings.TrimSpace(string(command(t, stage, data, "hash-object", "-w", "--literally", "-t", parts[1], "--stdin")))
			if actual != parts[0] {
				t.Fatalf("fixture object identity changed: type=%s expected=%s actual=%q", parts[1], parts[0], actual)
			}
		}
		return nil
	}
	return f
}

func TestPrepareOnlyRegularSettingsAndRealMetadata(t *testing.T) {
	f := newFixture(t, "regular")
	if err := os.Mkdir(filepath.Join(f.root, ".trusted"), 0700); err != nil {
		t.Fatal(err)
	}
	r, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head)
	if err != nil {
		t.Fatal(err)
	}
	if r.snapshot != f.s || r.Blob != blobID(f.body) {
		t.Fatal("wrong provenance", r)
	}
	actual, err := git(context.Background(), f.root, "rev-parse", "HEAD")
	if err != nil || actual != f.s.Merge {
		t.Fatal(actual, err)
	}
	data, err := os.ReadFile(filepath.Join(f.root, ".github/settings.yml"))
	if err != nil || string(data) != string(f.body) {
		t.Fatal(err)
	}
	for _, name := range []string{".github/settings.yml", ".git/" + receiptName} {
		info, err := os.Stat(filepath.Join(f.root, name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("private file mode", name, err)
		}
	}
	for _, name := range []string{"poison.sh", "CANARY-executed", "injected", ".github/workflows", ".gitattributes"} {
		if _, err := os.Lstat(filepath.Join(f.root, name)); !os.IsNotExist(err) {
			t.Fatal("candidate code materialized or ran", name)
		}
	}
	if _, err = f.o.verify(context.Background(), f.root, 12, f.s.Head); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(f.temporary)
	if err != nil || len(entries) != 0 {
		t.Fatal("staging cleanup", err)
	}
	for _, route := range f.requests {
		if route != "" && route != "/pulls/12" && route != "/git/ref/heads/main" && !strings.HasPrefix(route, "/git/blobs/") {
			t.Fatal("unexpected API route", route)
		}
	}
}

func TestPrepareRejectsUnsafeGitPathsAndData(t *testing.T) {
	for _, kind := range []string{"directory symlink", "file symlink", "submodule", "missing", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t, kind)
			if _, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head); err == nil {
				t.Fatal("unsafe candidate accepted")
			}
			assertClean(t, f)
		})
	}
}

func assertClean(t *testing.T, f *fixture) {
	t.Helper()
	for _, name := range []string{".git", ".github"} {
		if _, err := os.Lstat(filepath.Join(f.root, name)); !os.IsNotExist(err) {
			t.Fatal("failed preparation left owned destination", name)
		}
	}
	entries, err := os.ReadDir(f.temporary)
	if err != nil || len(entries) != 0 {
		t.Fatal("staging not removed", err)
	}
}

func TestPrepareRejectsWrongPRAndStaleRevisions(t *testing.T) {
	for _, kind := range []string{"repository", "private target", "fork", "closed", "base branch", "head", "pending merge", "merge parents", "base moved", "end head moved", "end base moved", "end merge moved", "read refused"} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t, "regular")
			pr := f.routes["/pulls/12"].(map[string]any)
			switch kind {
			case "repository":
				f.routes[""] = map[string]any{"id": 1, "full_name": target}
			case "private target":
				f.routes[""].(map[string]any)["visibility"] = "private"
			case "fork":
				pr["head"].(map[string]any)["repo"] = map[string]any{"id": 1, "full_name": "attacker/fork"}
			case "closed":
				pr["state"] = "closed"
			case "base branch":
				pr["base"].(map[string]any)["ref"] = "other"
			case "head":
				pr["head"].(map[string]any)["sha"] = strings.Repeat("a", 40)
			case "pending merge":
				pr["mergeable"] = nil
			case "merge parents":
				pr["merge_commit_sha"] = f.s.Head
			case "base moved":
				f.routes["/git/ref/heads/main"] = map[string]any{"object": map[string]any{"sha": strings.Repeat("a", 40), "type": "commit"}}
			case "read refused":
				delete(f.routes, "/pulls/12")
			}
			if strings.HasPrefix(kind, "end ") {
				fetch := f.o.fetch
				f.o.fetch = func(ctx context.Context, root, sha string) error {
					if err := fetch(ctx, root, sha); err != nil {
						return err
					}
					switch kind {
					case "end head moved":
						pr["head"].(map[string]any)["sha"] = strings.Repeat("a", 40)
					case "end base moved":
						pr["base"].(map[string]any)["sha"] = strings.Repeat("a", 40)
					case "end merge moved":
						pr["merge_commit_sha"] = strings.Repeat("a", 40)
					}
					return nil
				}
			}
			if _, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head); err == nil {
				t.Fatal("mismatch accepted")
			}
			assertClean(t, f)
		})
	}
}

func TestPrepareRejectsBlobForgery(t *testing.T) {
	for _, key := range []string{"sha", "size", "encoding", "content"} {
		t.Run(key, func(t *testing.T) {
			f := newFixture(t, "regular")
			blob := f.routes["/git/blobs/"+blobID(f.body)].(map[string]any)
			if key == "size" {
				blob[key] = nil
			} else {
				blob[key] = "CANARY-malicious"
			}
			if _, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head); err == nil {
				t.Fatal("forged blob accepted")
			}
			assertClean(t, f)
		})
	}
}

func TestPreparePreservesExistingDestinations(t *testing.T) {
	for _, name := range []string{".git", ".github"} {
		f := newFixture(t, "regular")
		path := filepath.Join(f.root, name)
		if err := os.WriteFile(path, []byte("CANARY-original"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head); err == nil {
			t.Fatal("overwrote existing path")
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "CANARY-original" {
			t.Fatal("original changed", err)
		}
	}
	f := newFixture(t, "regular")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(f.root, ".github")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head); err == nil {
		t.Fatal("symlink destination accepted")
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("escaped workspace")
	}
}

func TestPrepareRefusesInsideAnExistingCheckout(t *testing.T) {
	f := newFixture(t, "regular")
	f.root = filepath.Join(f.source, "consumer")
	if err := os.Mkdir(f.root, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head); err == nil {
		t.Fatal("nested checkout accepted")
	}
	if len(f.requests) != 0 {
		t.Fatal("nested checkout reached remote work")
	}
	if _, err := os.Stat(filepath.Join(f.source, ".git")); err != nil {
		t.Fatal("parent checkout changed", err)
	}
	assertClean(t, f)
}

func TestVerifyRejectsTamperingAndStaleResults(t *testing.T) {
	for _, kind := range []string{"content", "forged digest", "receipt blob", "origin", "HEAD", "symlink file", "symlink directory", "head moved", "base moved", "merge moved"} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t, "regular")
			r, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(f.root, ".github/settings.yml")
			switch kind {
			case "content", "forged digest":
				if err := os.WriteFile(path, []byte("CANARY-tampered"), 0600); err != nil {
					t.Fatal(err)
				}
				if kind == "forged digest" {
					d := sha256.Sum256([]byte("CANARY-tampered"))
					r.Digest = hex.EncodeToString(d[:])
					data, _ := json.Marshal(r)
					if err := os.WriteFile(filepath.Join(f.root, ".git", receiptName), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
			case "receipt blob":
				r.Blob = strings.Repeat("a", 40)
				data, _ := json.Marshal(r)
				if err := os.WriteFile(filepath.Join(f.root, ".git", receiptName), data, 0600); err != nil {
					t.Fatal(err)
				}
			case "origin":
				if _, err := git(context.Background(), f.root, "remote", "set-url", "origin", "https://github.com/attacker/repo.git"); err != nil {
					t.Fatal(err)
				}
			case "HEAD":
				if _, err := git(context.Background(), f.root, "update-ref", "--no-deref", "HEAD", f.s.Base); err != nil {
					t.Fatal(err)
				}
			case "symlink file":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(f.source, "README.md"), path); err != nil {
					t.Fatal(err)
				}
			case "symlink directory":
				if err := os.RemoveAll(filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(f.source, filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
			case "head moved":
				f.routes["/pulls/12"].(map[string]any)["head"].(map[string]any)["sha"] = strings.Repeat("a", 40)
			case "base moved":
				f.routes["/git/ref/heads/main"] = map[string]any{"object": map[string]any{"sha": strings.Repeat("a", 40), "type": "commit"}}
			case "merge moved":
				f.routes["/pulls/12"].(map[string]any)["merge_commit_sha"] = strings.Repeat("a", 40)
			}
			if _, err := f.o.verify(context.Background(), f.root, 12, f.s.Head); err == nil {
				t.Fatal("tampered or stale result accepted")
			}
		})
	}
}

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHTTPReadsAreBoundedAndDoNotFollowRedirects(t *testing.T) {
	for _, kind := range []string{"success", "redirect", "403", "oversize", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: transport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.String() != "https://api.github.com/repos/"+target+"/pulls/12" {
					t.Fatal("unexpected request", req.Method, req.URL)
				}
				status, body := 200, `{"number":12}`
				headers := http.Header{}
				switch kind {
				case "redirect":
					status = 302
					headers.Set("Location", "https://attacker.invalid/CANARY")
				case "403":
					status = 403
					body = "CANARY-private-error"
				case "oversize":
					body = strings.Repeat("x", maxResponse+1)
				case "malformed":
					body = "CANARY-invalid-json"
				}
				return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})}
			var value struct{ Number int }
			err := getter(client)(context.Background(), "/pulls/12", &value)
			if (err == nil) != (kind == "success") || calls != 1 {
				t.Fatal("read boundary", err, calls)
			}
		})
	}
}

func TestArgumentBoundary(t *testing.T) {
	for _, args := range [][]string{nil, {"prepare", "0", strings.Repeat("a", 40)}, {"prepare", "01", strings.Repeat("a", 40)}, {"prepare", "12; touch CANARY", strings.Repeat("a", 40)}, {"prepare", "12", "--upload-pack=CANARY"}, {"apply", "12", strings.Repeat("a", 40)}, {"prepare", "12", strings.Repeat("a", 40), "../../escape"}} {
		if _, _, _, err := arguments(args); err == nil {
			t.Fatal("unsafe arguments accepted")
		}
	}
}

func TestPublicFailuresNeverEchoCandidateData(t *testing.T) {
	for _, args := range [][]string{{"prepare", "CANARY-PR", strings.Repeat("a", 40)}, {"verify", "12", "CANARY-SHA"}} {
		var output, diagnostic bytes.Buffer
		if status := execute(args, &output, &diagnostic); status != 1 || output.Len() != 0 || diagnostic.String() != "pr-settings: preparation or freshness verification failed\n" {
			t.Fatalf("unsafe public failure: %d %q %q", status, output.String(), diagnostic.String())
		}
	}
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("GITHUB_WORKSPACE", root)
	t.Setenv("GH_TOKEN", "")
	var output, diagnostic bytes.Buffer
	if status := execute([]string{"prepare", "12", strings.Repeat("a", 40)}, &output, &diagnostic); status != 1 || output.Len() != 0 || strings.Contains(diagnostic.String(), root) {
		t.Fatal("missing-token boundary failed")
	}
}

func TestUnexpectedFetchedBlobsAreRejected(t *testing.T) {
	f := newFixture(t, "regular")
	fetch := f.o.fetch
	f.o.fetch = func(ctx context.Context, stage, sha string) error {
		if err := fetch(ctx, stage, sha); err != nil {
			return err
		}
		command(t, stage, []byte("CANARY-unrequested-content"), "hash-object", "-w", "--stdin")
		return nil
	}
	if _, err := f.o.prepare(context.Background(), f.root, f.temporary, 12, f.s.Head); err == nil {
		t.Fatal("unfiltered candidate content accepted")
	}
	assertClean(t, f)
}

func TestMetadataFetchScopeAndGitEnvironment(t *testing.T) {
	bin, root := t.TempDir(), t.TempDir()
	argumentsPath, environmentPath := filepath.Join(root, "arguments"), filepath.Join(root, "environment")
	// Capture only synthetic test variables, never the process environment.
	script := `#!/bin/sh
printf '%s\n' "$@" > "$CAPTURE_ARGUMENTS"
printf '%s|%s|%s|%s|%s|%s' "${GIT_CONFIG_COUNT-}" "${GH_TOKEN-}" "${GITHUB_TOKEN-}" "${SSH_ASKPASS-}" "${GIT_ALLOW_PROTOCOL-}" "${GIT_TERMINAL_PROMPT-}" > "$CAPTURE_ENVIRONMENT"
`
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CAPTURE_ARGUMENTS", argumentsPath)
	t.Setenv("CAPTURE_ENVIRONMENT", environmentPath)
	t.Setenv("GIT_CONFIG_COUNT", "CANARY-config")
	t.Setenv("GH_TOKEN", "CANARY-owner-token")
	t.Setenv("GITHUB_TOKEN", "CANARY-workflow-token")
	t.Setenv("SSH_ASKPASS", "CANARY-program")
	sha := strings.Repeat("a", 40)
	if err := fetchMetadata(context.Background(), root, sha); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(argumentsPath)
	if err != nil {
		t.Fatal(err)
	}
	args := string(data)
	for _, value := range []string{"core.hooksPath=/dev/null", "credential.helper=", "http.followRedirects=false", "protocol.allow=never", "protocol.https.allow=always", "--filter=blob:none", "--depth=1", "--no-tags", "--no-recurse-submodules", "origin\n" + sha + "\n"} {
		if !strings.Contains(args, value) {
			t.Fatal("unsafe metadata fetch arguments", value)
		}
	}
	data, err = os.ReadFile(environmentPath)
	if err != nil || string(data) != "||||https|0" {
		t.Fatal("inherited Git programs or credentials", err)
	}
}
