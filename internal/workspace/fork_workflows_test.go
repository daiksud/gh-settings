package workspace

import (
	"strings"
	"testing"
)

func TestForkPolicyYAMLTypesAndRoundtrip(t *testing.T) {
	for _, value := range []string{"{}", "{fork_pr_approval_policy: first_time_contributors}", "{private_fork_workflows: {}}", "{private_fork_workflows: {send_write_tokens_to_workflows: false}}", "{private_fork_workflows: {run_workflows_from_fork_pull_requests: false, send_secrets_and_variables: true, require_approval_for_fork_pr_workflows: false}}"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: " + value + "}"))
		if err != nil {
			t.Fatal(value, err)
		}
		data, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil {
			t.Fatal(err)
		}
		if d.Actions.PrivateFork.Empty() != again.Actions.PrivateFork.Empty() {
			t.Fatal("lost false", string(data))
		}
		if strings.Contains(value, "send_write_tokens_to_workflows: false") && !strings.Contains(string(data), "send_write_tokens_to_workflows: false") {
			t.Fatal(string(data))
		}
	}
	for _, value := range []string{"{fork_pr_approval_policy: null}", "{fork_pr_approval_policy: false}", "{fork_pr_approval_policy: never}", "{private_fork_workflows: null}", "{private_fork_workflows: []}", "{private_fork_workflows: {send_secrets_and_variables: null}}", "{private_fork_workflows: {send_secrets_and_variables: 'false'}}", "{private_fork_workflows: {send_secrets_and_variables: 0}}", "{private_fork_workflows: {unknown: false}}", "{private_fork_workflows: {send_secrets_and_variables: false, send_secrets_and_variables: true}}"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: " + value + "}")); err == nil {
			t.Fatal("invalid accepted", value)
		}
	}
}
