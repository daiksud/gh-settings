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
	name, body, errorText                        string
	expected                                     map[string]any
	noRepo, noRead, outputFailure, rewriteOrigin bool
	args                                         []string
	status                                       int
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
	ordinary := map[string]any{"description": "Example repository", "homepage": "https://example.com", "has_issues": true, "has_projects": false, "has_wiki": true}
	empty := map[string]any{"description": "", "homepage": "", "has_issues": false, "has_projects": false, "has_wiki": false}
	cases := []scenario{
		{name: "five-settings", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"id":123}`, expected: ordinary},
		{name: "zero-values", body: `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty},
		{name: "unset-strings", body: `{"description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty},
		{name: "origin-override", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary},
		{name: "origin-insteadof", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, rewriteOrigin: true},
		{name: "read-failure", body: `{"message":"Fixture access denied"}`, status: 403, errorText: "403"},
		{name: "repository-failure", noRepo: true, noRead: true, errorText: "origin"},
		{name: "incomplete-response", body: `{"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, errorText: "missing description"},
		{name: "output-failure", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, outputFailure: true, errorText: "write output"},
		{name: "unsupported-command", args: []string{"apply"}, noRead: true, errorText: "unsupported"},
	}
	for _, test := range cases {
		if err := check(path, test); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", test.name, err)
			os.Exit(1)
		}
		fmt.Printf("PASS %s\n", test.name)
	}
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
		for _, args := range [][]string{{"init", "--quiet"}, {"remote", "add", "origin", "https://github.com/owner/current.git"}} {
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
	if test.expected != nil {
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
