// Acceptance checks launch an already-built CLI against a local Unix-socket API fixture.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

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
	cases := append(exportScenarios(), validateScenarios()...)
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
	fixture, err := startAPIFixture(test.body, test.status)
	if err != nil {
		return err
	}
	defer fixture.close()
	root, work, cfg := fixture.root, fixture.work, fixture.cfg
	env := fixtureEnv(cfg)
	if err := setupGitCheckout(work, env, test); err != nil {
		return err
	}
	if test.input != "" {
		if err := os.Mkdir(filepath.Join(work, ".github"), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(work, ".github", "settings.yml"), []byte(test.input), 0600); err != nil {
			return err
		}
	}
	result, err := executeCLI(binary, work, root, env, test.args, test.outputFailure)
	if err != nil {
		return err
	}
	if err := fixture.checkRequests(test.noRead); err != nil {
		return err
	}
	if test.success {
		if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
			return fmt.Errorf("execution: %v, stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
		}
	} else if test.expected != nil {
		if result.runErr != nil || result.stderr.Len() != 0 {
			return fmt.Errorf("execution: %v, stderr=%q", result.runErr, result.stderr.String())
		}
		var actual map[string]any
		if err := yaml.Unmarshal(result.stdout.Bytes(), &actual); err != nil {
			return err
		}
		if !reflect.DeepEqual(actual, test.expected) {
			return fmt.Errorf("YAML=%#v, want %#v", actual, test.expected)
		}
	} else {
		exit, ok := result.runErr.(*exec.ExitError)
		if !ok || exit.ExitCode() == 0 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), test.errorText) {
			return fmt.Errorf("execution: %v, stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
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
