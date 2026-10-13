// Acceptance checks launch an already-built CLI against a local Unix-socket API fixture.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v4"
)

type scenario struct {
	name, body, errorText                                                     string
	expected                                                                  map[string]any
	noRepo, noRead, outputFailure, rewriteOrigin, aliasOrigin, worktreeOrigin bool
	userlessOrigin                                                            bool
	args                                                                      []string
	status                                                                    int
}

func check(binary string, test scenario) error {
	fixture, err := startAPIFixture(test.body, test.status)
	if err != nil {
		return err
	}
	defer fixture.close()
	root, work, cfg := fixture.root, fixture.work, fixture.cfg
	env := exportFixtureEnv(cfg)
	if err := setupGitCheckout(work, env, test); err != nil {
		return err
	}
	result, err := executeCLI(binary, work, root, env, test.args, test.outputFailure)
	if err != nil {
		return err
	}
	if err := fixture.checkRequests(test.noRead); err != nil {
		return err
	}
	if test.expected != nil {
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
	return append(env, "GH_CONFIG_DIR="+config, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
}

func exportFixtureEnv(config string) []string {
	return append(fixtureEnv(config), "GH_TOKEN=fixture-only-token", "GH_REPO=other/repository")
}
