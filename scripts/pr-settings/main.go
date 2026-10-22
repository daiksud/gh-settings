// Command pr-settings prepares one untrusted settings blob using trusted code.
package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

const (
	target      = "daiksud/gh-settings"
	targetID    = int64(1414158260)
	origin      = "https://github.com/" + target + ".git"
	maxConfig   = 1024 * 1024
	maxResponse = 2 * maxConfig
	receiptName = "gh-settings-plan.json"
)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

type snapshot struct {
	RepositoryID int64  `json:"repository_id"`
	Number       int    `json:"number"`
	Head         string `json:"head"`
	Base         string `json:"base"`
	Merge        string `json:"merge"`
}

type receipt struct {
	Version int    `json:"version"`
	Target  string `json:"target"`
	snapshot
	Tree   string `json:"tree"`
	Blob   string `json:"blob"`
	Digest string `json:"config_sha256"`
}

type observer struct {
	get   func(context.Context, string, any) error
	fetch func(context.Context, string, string) error
}

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}

func execute(args []string, output, diagnostic io.Writer) int {
	if err := run(args, output); err != nil {
		// API, Git, YAML and filesystem error values may contain private data.
		fmt.Fprintln(diagnostic, "pr-settings: preparation or freshness verification failed")
		return 1
	}
	return 0
}

func arguments(args []string) (string, int, string, error) {
	if len(args) != 3 || (args[0] != "prepare" && args[0] != "verify") || len(args[1]) > 10 || !shaPattern.MatchString(args[2]) {
		return "", 0, "", errors.New("invalid invocation")
	}
	n, err := strconv.Atoi(args[1])
	if err != nil || n <= 0 || strconv.Itoa(n) != args[1] {
		return "", 0, "", errors.New("invalid PR number")
	}
	return args[0], n, args[2], nil
}

func run(args []string, output io.Writer) error {
	mode, number, head, err := arguments(args)
	if err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" || !filepath.IsAbs(workspace) {
		return errors.New("workspace required")
	}
	info, err := os.Lstat(workspace)
	if err != nil || !info.IsDir() {
		return errors.New("invalid workspace")
	}
	canonical, err := filepath.EvalSymlinks(workspace)
	if err != nil || canonical != root {
		return errors.New("workspace mismatch")
	}
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		return errors.New("explicit ordinary read token required")
	}
	client, err := api.NewHTTPClient(api.ClientOptions{Host: "github.com", APIHost: "api.github.com", AuthToken: token, LogIgnoreEnv: true, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	o := observer{get: getter(client), fetch: fetchMetadata}
	var r receipt
	if mode == "prepare" {
		r, err = o.prepare(ctx, root, os.Getenv("RUNNER_TEMP"), number, head)
	} else {
		r, err = o.verify(ctx, root, number, head)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "target=%s\npr-number=%d\nhead-sha=%s\nbase-sha=%s\nconfig-sha=%s\nconfig-sha256=%s\n", target, r.Number, r.Head, r.Base, r.Merge, r.Digest)
	if err != nil && mode == "prepare" {
		_ = os.RemoveAll(filepath.Join(root, ".git"))
		_ = os.RemoveAll(filepath.Join(root, ".github"))
	}
	return err
}

func getter(client *http.Client) func(context.Context, string, any) error {
	return func(ctx context.Context, route string, destination any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+target+route, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Cache-Control", "no-cache")
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return errors.New("read refused")
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, maxResponse+1))
		if err != nil || len(data) > maxResponse {
			return errors.New("response limit")
		}
		return json.Unmarshal(data, destination)
	}
}

func (o observer) observe(ctx context.Context, number int, head string) (snapshot, error) {
	var repo struct {
		ID         int64  `json:"id"`
		Name       string `json:"full_name"`
		Visibility string `json:"visibility"`
	}
	if err := o.get(ctx, "", &repo); err != nil {
		return snapshot{}, err
	}
	if repo.ID != targetID || repo.Name != target || repo.Visibility != "public" {
		return snapshot{}, errors.New("wrong repository")
	}
	type reference struct {
		SHA  string `json:"sha"`
		Ref  string `json:"ref"`
		Repo struct {
			ID   int64  `json:"id"`
			Name string `json:"full_name"`
		} `json:"repo"`
	}
	var pr struct {
		Number    int       `json:"number"`
		State     string    `json:"state"`
		Base      reference `json:"base"`
		Head      reference `json:"head"`
		Merge     string    `json:"merge_commit_sha"`
		Mergeable *bool     `json:"mergeable"`
	}
	if err := o.get(ctx, "/pulls/"+strconv.Itoa(number), &pr); err != nil {
		return snapshot{}, err
	}
	if pr.Number != number || pr.State != "open" || pr.Base.Ref != "main" || pr.Head.SHA != head || pr.Base.Repo.ID != targetID || pr.Head.Repo.ID != targetID || pr.Base.Repo.Name != target || pr.Head.Repo.Name != target || pr.Mergeable == nil || !*pr.Mergeable || !shaPattern.MatchString(pr.Base.SHA) || !shaPattern.MatchString(pr.Merge) {
		return snapshot{}, errors.New("unavailable or mismatched PR")
	}
	var ref struct {
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err := o.get(ctx, "/git/ref/heads/main", &ref); err != nil {
		return snapshot{}, err
	}
	if ref.Object.SHA != pr.Base.SHA || ref.Object.Type != "commit" {
		return snapshot{}, errors.New("base moved")
	}
	return snapshot{RepositoryID: repo.ID, Number: number, Head: head, Base: pr.Base.SHA, Merge: pr.Merge}, nil
}

func cleanRoot(ctx context.Context, root string) error {
	for _, path := range []string{".git", ".github"} {
		if _, err := os.Lstat(filepath.Join(root, path)); !os.IsNotExist(err) {
			return errors.New("destination exists")
		}
	}
	if _, err := git(ctx, root, "rev-parse", "--git-dir"); err == nil {
		return errors.New("nested workspace")
	}
	return nil
}

func (o observer) prepare(ctx context.Context, root, temporary string, number int, head string) (r receipt, err error) {
	if !filepath.IsAbs(temporary) {
		return r, errors.New("runner temporary directory required")
	}
	if err = cleanRoot(ctx, root); err != nil {
		return r, err
	}
	s, err := o.observe(ctx, number, head)
	if err != nil {
		return r, err
	}
	stage, err := os.MkdirTemp(temporary, "gh-settings-pr-")
	if err != nil {
		return r, err
	}
	var owned []string
	defer func() {
		if cleanup := os.RemoveAll(stage); cleanup != nil && err == nil {
			err = errors.New("temporary cleanup failed")
		}
		if err != nil {
			for _, path := range owned {
				_ = os.RemoveAll(path)
			}
		}
	}()
	if _, err = git(ctx, stage, "init", "--quiet", "--template="); err != nil {
		return r, err
	}
	if _, err = git(ctx, stage, "remote", "add", "origin", origin); err != nil {
		return r, err
	}
	if err = o.fetch(ctx, stage, s.Merge); err != nil {
		return r, err
	}
	objects, err := git(ctx, stage, "cat-file", "--batch-all-objects", "--batch-check=%(objecttype)")
	if err != nil {
		return r, err
	}
	for _, kind := range strings.Fields(objects) {
		if kind != "commit" && kind != "tree" {
			return r, errors.New("unexpected candidate content")
		}
	}
	tree, blob, err := mergeBlob(ctx, stage, s)
	if err != nil {
		return r, err
	}
	var raw struct {
		SHA      string `json:"sha"`
		Encoding string `json:"encoding"`
		Size     *int64 `json:"size"`
		Content  string `json:"content"`
	}
	if err = o.get(ctx, "/git/blobs/"+blob, &raw); err != nil {
		return r, err
	}
	if raw.SHA != blob || raw.Encoding != "base64" || raw.Size == nil || *raw.Size < 0 || *raw.Size > maxConfig {
		return r, errors.New("invalid blob metadata")
	}
	data, err := base64.StdEncoding.DecodeString(raw.Content)
	if err != nil || len(data) > maxConfig || int64(len(data)) != *raw.Size {
		return r, errors.New("invalid blob content")
	}
	if blobID(data) != blob {
		return r, errors.New("blob mismatch")
	}
	if _, err = git(ctx, stage, "update-ref", "--no-deref", "HEAD", s.Merge); err != nil {
		return r, err
	}
	digest := sha256.Sum256(data)
	r = receipt{Version: 1, Target: target, snapshot: s, Tree: tree, Blob: blob, Digest: hex.EncodeToString(digest[:])}
	encoded, err := json.Marshal(r)
	if err != nil {
		return r, err
	}
	if err = os.WriteFile(filepath.Join(stage, ".git", receiptName), encoded, 0600); err != nil {
		return r, err
	}
	if err = cleanRoot(ctx, root); err != nil {
		return r, err
	}
	for _, name := range []string{".git", ".github"} {
		path := filepath.Join(root, name)
		if err = os.Mkdir(path, 0700); err != nil {
			return r, err
		}
		owned = append(owned, path)
	}
	entries, err := os.ReadDir(filepath.Join(stage, ".git"))
	if err != nil {
		return r, err
	}
	for _, entry := range entries {
		if err = os.Rename(filepath.Join(stage, ".git", entry.Name()), filepath.Join(root, ".git", entry.Name())); err != nil {
			return r, err
		}
	}
	if err = os.WriteFile(filepath.Join(root, ".github", "settings.yml"), data, 0600); err != nil {
		return r, err
	}
	// End-of-preparation check; the consumer must verify again after its plan.
	latest, err := o.observe(ctx, number, head)
	if err != nil || latest != s {
		return r, errors.New("stale preparation")
	}
	return r, nil
}

func (o observer) verify(ctx context.Context, root string, number int, head string) (receipt, error) {
	var r receipt
	for _, name := range []string{".git", ".github"} {
		info, err := os.Lstat(filepath.Join(root, name))
		if err != nil || !info.IsDir() {
			return r, errors.New("unsafe prepared directory")
		}
	}
	data, err := regularFile(filepath.Join(root, ".git", receiptName), 4096)
	if err != nil {
		return r, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&r); err != nil {
		return r, err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return r, errors.New("trailing receipt content")
	}
	if r.Version != 1 || r.Target != target || r.RepositoryID != targetID || r.Number != number || r.Head != head || !shaPattern.MatchString(r.Base) || !shaPattern.MatchString(r.Merge) || !shaPattern.MatchString(r.Tree) || !shaPattern.MatchString(r.Blob) {
		return r, errors.New("invalid receipt")
	}
	data, err = regularFile(filepath.Join(root, ".github", "settings.yml"), maxConfig)
	if err != nil {
		return r, err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != r.Digest {
		return r, errors.New("configuration changed")
	}
	if blobID(data) != r.Blob {
		return r, errors.New("configuration no longer matches candidate blob")
	}
	current, err := git(ctx, root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || current != r.Merge {
		return r, errors.New("checkout changed")
	}
	remote, err := git(ctx, root, "config", "--get-all", "remote.origin.url")
	if err != nil || remote != origin {
		return r, errors.New("origin changed")
	}
	tree, blob, err := mergeBlob(ctx, root, r.snapshot)
	if err != nil || tree != r.Tree || blob != r.Blob {
		return r, errors.New("receipt provenance changed")
	}
	latest, err := o.observe(ctx, number, head)
	if err != nil || latest != r.snapshot {
		return r, errors.New("stale result")
	}
	return r, nil
}

func blobID(data []byte) string {
	object := sha1.New()
	fmt.Fprintf(object, "blob %d\x00", len(data))
	_, _ = object.Write(data)
	return hex.EncodeToString(object.Sum(nil))
}

func regularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("invalid prepared file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("prepared file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("prepared file limit")
	}
	return data, nil
}

func mergeBlob(ctx context.Context, root string, s snapshot) (string, string, error) {
	kind, err := git(ctx, root, "cat-file", "-t", s.Merge)
	if err != nil || kind != "commit" {
		return "", "", errors.New("merge commit required")
	}
	metadata, err := git(ctx, root, "cat-file", "-p", s.Merge)
	if err != nil {
		return "", "", err
	}
	var tree string
	var parents []string
	for line := range strings.SplitSeq(strings.SplitN(metadata, "\n\n", 2)[0], "\n") {
		if strings.HasPrefix(line, "tree ") {
			tree = strings.TrimPrefix(line, "tree ")
		}
		if strings.HasPrefix(line, "parent ") {
			parents = append(parents, strings.TrimPrefix(line, "parent "))
		}
	}
	if !shaPattern.MatchString(tree) || len(parents) != 2 || parents[0] != s.Base || parents[1] != s.Head {
		return "", "", errors.New("wrong merge identity")
	}
	directory, err := treeEntry(ctx, root, tree, ".github", "040000", "tree")
	if err != nil {
		return "", "", err
	}
	blob, err := treeEntry(ctx, root, directory, "settings.yml", "100644", "blob")
	return tree, blob, err
}

func treeEntry(ctx context.Context, root, tree, name, mode, kind string) (string, error) {
	entry, err := git(ctx, root, "ls-tree", "-z", tree, "--", name)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(entry, "\t", 2)
	if len(parts) != 2 || parts[1] != name+"\x00" {
		return "", errors.New("missing or ambiguous configuration path")
	}
	fields := strings.Fields(parts[0])
	if len(fields) != 3 || fields[0] != mode || fields[1] != kind || !shaPattern.MatchString(fields[2]) {
		return "", errors.New("unsafe configuration path")
	}
	return fields[2], nil
}

func fetchMetadata(ctx context.Context, root, sha string) error {
	_, err := git(ctx, root, "fetch", "--quiet", "--filter=blob:none", "--depth=1", "--no-tags", "--no-recurse-submodules", "origin", sha)
	return err
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > maxResponse {
		return 0, errors.New("Git output limit")
	}
	return b.Buffer.Write(p)
}

func git(ctx context.Context, root string, args ...string) (string, error) {
	options := []string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "core.askPass=", "-c", "credential.helper=", "-c", "http.followRedirects=false", "-c", "protocol.allow=never", "-c", "protocol.https.allow=always", "-c", "submodule.recurse=false"}
	cmd := exec.CommandContext(ctx, "git", append(options, args...)...)
	cmd.Dir = root
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if !strings.HasPrefix(name, "GIT_") && name != "GH_TOKEN" && name != "GITHUB_TOKEN" && name != "SSH_ASKPASS" {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=https", "GIT_NO_LAZY_FETCH=1")
	var stdout, stderr boundedOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.New("Git operation failed")
	}
	return strings.TrimSuffix(stdout.String(), "\n"), nil
}
