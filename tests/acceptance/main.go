// Acceptance checks launch an already-built CLI against a local Unix-socket API fixture.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"go.yaml.in/yaml/v4"
)

type scenario struct {
	name, body, errorText, input                                              string
	expected                                                                  map[string]any
	noRepo, noRead, outputFailure, rewriteOrigin, aliasOrigin, worktreeOrigin bool
	userlessOrigin, success                                                   bool
	args                                                                      []string
	status                                                                    int
}

func main() {
	binary := flag.String("binary", "", "path to the already-built gh-settings executable")
	flag.Parse()
	if *binary == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: acceptance --binary PATH")
		os.Exit(2)
	}
	path, err := filepath.Abs(*binary)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ordinary := map[string]any{"schemaVersion": 1, "settings": map[string]any{"repository": map[string]any{"description": "Example repository", "homepage": "https://example.com", "has_issues": true, "has_projects": false, "has_wiki": true}}}
	empty := map[string]any{"schemaVersion": 1, "settings": map[string]any{"repository": map[string]any{"description": "", "homepage": "", "has_issues": false, "has_projects": false, "has_wiki": false}}}
	cases := []scenario{
		{name: "five-settings", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"id":123}`, expected: ordinary},
		{name: "zero-values", body: `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty},
		{name: "unset-strings", body: `{"description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty},
		{name: "origin-override", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary},
		{name: "origin-scp-without-user", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, userlessOrigin: true},
		{name: "origin-insteadof", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, rewriteOrigin: true},
		{name: "origin-alias", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, aliasOrigin: true},
		{name: "origin-worktree", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, worktreeOrigin: true},
		{name: "read-failure", body: `{"message":"Fixture access denied"}`, status: 403, errorText: "403"},
		{name: "repository-failure", noRepo: true, noRead: true, errorText: "origin"},
		{name: "incomplete-response", body: `{"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, errorText: "missing description"},
		{name: "output-failure", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, outputFailure: true, errorText: "write output"},
		{name: "unsupported-command", args: []string{"apply"}, noRead: true, errorText: "unsupported"},
		{name: "validate-all-fields", args: []string{"validate"}, input: repositoryInput("description: Example\nhomepage: https://example.com\nhas_issues: true\nhas_projects: false\nhas_wiki: true\n"), noRead: true, success: true},
		{name: "validate-partial-zero-values", args: []string{"validate"}, input: repositoryInput("description: ''\nhas_issues: false\n"), noRead: true, success: true},
		{name: "validate-unknown-key", args: []string{"validate"}, input: repositoryInput("visibility: private\n"), noRead: true, errorText: "unknown setting"},
		{name: "validate-duplicate-key", args: []string{"validate"}, input: repositoryInput("has_issues: true\nhas_issues: false\n"), noRead: true, errorText: "duplicate key"},
		{name: "validate-alias", args: []string{"validate"}, input: repositoryInput("description: &text Example\nhomepage: *text\n"), noRead: true, success: true},
		{name: "validate-merge", args: []string{"validate"}, input: repositoryInput("<<: {has_issues: false}\n"), noRead: true, success: true},
		{name: "validate-quoted-null-string", args: []string{"validate"}, input: repositoryInput("description: 'null'\n"), noRead: true, success: true},
		{name: "validate-wrong-type", args: []string{"validate"}, input: repositoryInput("has_issues: 'false'\n"), noRead: true, errorText: "must be a boolean"},
		{name: "validate-null-string", args: []string{"validate"}, input: repositoryInput("description: null\n"), noRead: true, errorText: "must be a string"},
		{name: "validate-null-boolean", args: []string{"validate"}, input: repositoryInput("has_issues: null\n"), noRead: true, errorText: "must be a boolean"},
		{name: "validate-malformed", args: []string{"validate"}, input: repositoryInput("has_issues: [\n"), noRead: true, errorText: "decode YAML"},
		{name: "validate-multiple-documents", args: []string{"validate"}, input: "has_issues: true\n---\nhas_wiki: false\n", noRead: true, errorText: "expected one YAML document"},
		{name: "validate-old-flat-format", args: []string{"validate"}, input: "schemaVersion: 1\nhas_issues: false\n", noRead: true, errorText: "unknown root key"},
		{name: "validate-missing-schema-version", args: []string{"validate"}, input: "settings: {}\n", noRead: true, errorText: "missing schemaVersion"},
		{name: "validate-unsupported-schema-version", args: []string{"validate"}, input: "schemaVersion: 2\n", noRead: true, errorText: "unsupported schemaVersion"},
		{name: "validate-null-key", args: []string{"validate"}, input: "schemaVersion: 1\nnull: ignored\n", noRead: true, errorText: "mapping keys must be strings"},
		{name: "validate-alias-duplicate-key", args: []string{"validate"}, input: repositoryInput("&field has_issues: true\n*field: false\n"), noRead: true, errorText: "duplicate key"},
		{name: "validate-alias-merge-key", args: []string{"validate"}, input: repositoryInput("&merge <<: {has_issues: true}\n*merge: {visibility: private}\n"), noRead: true, errorText: "merge key cannot be an alias"},
		{name: "validate-binary-string", args: []string{"validate"}, input: repositoryInput("description: !!binary /w==\n"), noRead: true, errorText: "must be a string"},
		{name: "validate-legacy-tagged-boolean", args: []string{"validate"}, input: repositoryInput("has_issues: !!bool yes\n"), noRead: true, errorText: "spelled true or false"},
		{name: "validate-mistagged-mapping", args: []string{"validate"}, input: "schemaVersion: 1\nsettings: !!str {}\n", noRead: true, errorText: "mapping must have YAML map tag"},
		{name: "validate-mistagged-merge-source", args: []string{"validate"}, input: repositoryInput("<<: !!str {has_issues: true}\n"), noRead: true, errorText: "mapping must have YAML map tag"},
		{name: "validate-missing-file", args: []string{"validate"}, noRead: true, errorText: "settings.yml"},
	}
	for _, test := range cases {
		if err := check(path, test); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", test.name, err)
			os.Exit(1)
		}
		fmt.Printf("PASS %s\n", test.name)
	}
}

func repositoryInput(fields string) string {
	var result strings.Builder
	result.WriteString("schemaVersion: 1\nsettings:\n  repository:\n")
	for _, line := range strings.SplitAfter(fields, "\n") {
		if line != "" {
			result.WriteString("    ")
			result.WriteString(line)
		}
	}
	return result.String()
}

func check(binary string, test scenario) error {
	// A short /tmp path stays within Unix socket path limits on both macOS and Linux.
	root, err := os.MkdirTemp("/tmp", "ghs-accept-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	work := filepath.Join(root, "repo")
	cfg := filepath.Join(root, "config")
	for _, dir := range []string{work, cfg} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return err
		}
	}
	socket := filepath.Join(root, "api.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return err
	}
	var reads atomic.Int32
	var invalid atomic.Bool
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/repos/owner/current" {
			invalid.Store(true)
			http.Error(w, "unexpected fixture request", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		status := test.status
		if status == 0 {
			status = 200
		}
		w.WriteHeader(status)
		fmt.Fprint(w, test.body)
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()
	if err := os.WriteFile(filepath.Join(cfg, "config.yml"), []byte("http_unix_socket: "+socket+"\n"), 0600); err != nil {
		return err
	}
	env := fixtureEnv(cfg)
	if !test.noRepo {
		gitCommands := [][]string{{"init", "--quiet"}}
		if !test.worktreeOrigin {
			origin := "https://github.com/owner/current.git"
			if test.userlessOrigin {
				origin = "github.com:owner/current.git"
			}
			gitCommands = append(gitCommands, []string{"remote", "add", "origin", origin})
		} else {
			gitCommands = append(gitCommands,
				[]string{"config", "--local", "extensions.worktreeConfig", "true"},
				[]string{"config", "--worktree", "remote.origin.url", "https://github.com/owner/current.git"})
		}
		for _, args := range gitCommands {
			cmd := exec.Command("git", args...)
			cmd.Dir = work
			cmd.Env = env
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("git: %w: %s", err, output)
			}
		}
		if test.rewriteOrigin {
			cmd := exec.Command("git", "config", "--local", "url.https://mirror.example/.insteadOf", "https://github.com/")
			cmd.Dir = work
			cmd.Env = env
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("git rewrite: %w: %s", err, output)
			}
		}
		if test.aliasOrigin {
			for _, args := range [][]string{
				{"remote", "set-url", "origin", "gh:owner/current.git"},
				{"config", "--local", "url.https://github.com/.insteadOf", "gh:"},
			} {
				cmd := exec.Command("git", args...)
				cmd.Dir = work
				cmd.Env = env
				if output, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("git alias: %w: %s", err, output)
				}
			}
		}
	}
	if test.input != "" {
		if err := os.Mkdir(filepath.Join(work, ".github"), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(work, ".github", "settings.yml"), []byte(test.input), 0600); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	args := test.args
	if args == nil {
		args = []string{"export"}
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = work
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if test.outputFailure {
		file := filepath.Join(root, "read-only-output")
		if err := os.WriteFile(file, nil, 0600); err != nil {
			return err
		}
		destination, err := os.Open(file)
		if err != nil {
			return err
		}
		defer destination.Close()
		cmd.Stdout = destination
	}
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if invalid.Load() {
		return fmt.Errorf("request was not a read of the current repository")
	}
	if test.noRead {
		if reads.Load() != 0 {
			return fmt.Errorf("unexpected API request")
		}
	} else if reads.Load() == 0 {
		return fmt.Errorf("CLI did not use the local API fixture")
	}
	if test.success {
		if runErr != nil || stdout.Len() != 0 || stderr.Len() != 0 {
			return fmt.Errorf("execution: %v, stdout=%q stderr=%q", runErr, stdout.String(), stderr.String())
		}
	} else if test.expected != nil {
		if runErr != nil || stderr.Len() != 0 {
			return fmt.Errorf("execution: %v, stderr=%q", runErr, stderr.String())
		}
		var actual map[string]any
		if err := yaml.Unmarshal(stdout.Bytes(), &actual); err != nil {
			return err
		}
		if !reflect.DeepEqual(actual, test.expected) {
			return fmt.Errorf("YAML=%#v, want %#v", actual, test.expected)
		}
	} else {
		exit, ok := runErr.(*exec.ExitError)
		if !ok || exit.ExitCode() == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.errorText) {
			return fmt.Errorf("execution: %v, stdout=%q stderr=%q", runErr, stdout.String(), stderr.String())
		}
	}
	if test.input != "" {
		unchanged, err := os.ReadFile(filepath.Join(work, ".github", "settings.yml"))
		if err != nil || string(unchanged) != test.input {
			return fmt.Errorf("configuration changed: %q, error=%v", unchanged, err)
		}
	}
	return nil
}

func fixtureEnv(config string) []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "GH_") || strings.HasPrefix(key, "GIT_") || key == "GITHUB_TOKEN" {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GH_CONFIG_DIR="+config, "GH_TOKEN=fixture-only-token", "GH_REPO=other/repository", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
}
