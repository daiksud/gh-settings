package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func checkValidate(binary string, test validateCase) error {
	fixture, err := startAPIFixture("", 0)
	if err != nil {
		return err
	}
	defer fixture.close()
	env := fixtureEnv(fixture.cfg)
	if err := initGitCheckout(fixture.work, env); err != nil {
		return err
	}
	if err := writeSettingsFile(fixture.work, test.input); err != nil {
		return err
	}
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"validate"}, false)
	if err != nil {
		return err
	}
	if err := fixture.checkRequests(true); err != nil {
		return err
	}
	if test.wantError == "" {
		if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
			return fmt.Errorf("execution: %v, stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
		}
	} else {
		exit, ok := result.runErr.(*exec.ExitError)
		if !ok || exit.ExitCode() == 0 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), test.wantError) {
			return fmt.Errorf("execution: %v, stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
		}
	}
	return checkSettingsUnchanged(fixture.work, test.input)
}
