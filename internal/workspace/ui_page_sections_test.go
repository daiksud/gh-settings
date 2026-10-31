package workspace

import (
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIPageSectionSchemaRoundTrip(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    general:
      releases:
        immutable_release: false
    actions:
      general:
        actions_permissions:
          policy: allow_owner_and_select_non_owner_actions_and_reusable_workflows
          require_actions_to_be_pinned_to_a_full_length_commit_sha: false
          allow_actions_created_by_github: true
          allow_actions_by_marketplace_verified_creators: false
          allow_or_block_specified_actions_and_reusable_workflows: []
        workflow_permissions:
          default_permissions: read_repository_contents_and_packages_permissions
          allow_github_actions_to_create_and_approve_pull_requests: false
    advanced_security:
      private_vulnerability_reporting: false
      secret_protection:
        enabled: false
        push_protection: false
`
	want := uimodel.Repository{Settings: &uimodel.RepositorySettings{
		General: &uimodel.General{Releases: &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: uiConfigPtr(false)}},
		Actions: &uimodel.ActionsSettings{General: &uimodel.ActionsGeneral{
			ActionsPermissions: uiConfigPtr(uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows),
			RequireActionsToBePinnedToAFullLengthCommitSHA: uiConfigPtr(false),
			AllowActionsCreatedByGitHub:                    uiConfigPtr(true), AllowActionsByMarketplaceVerifiedCreators: uiConfigPtr(false),
			AllowOrBlockSpecifiedActionsAndReusableWorkflows: uiConfigPtr([]string{}),
			WorkflowPermissions: uiConfigPtr(uimodel.ReadRepositoryContentsAndPackagesPermissions),
			AllowGitHubActionsToCreateAndApprovePullRequests: uiConfigPtr(false),
		}},
		SecurityAndQuality: &uimodel.SecurityAndQuality{AdvancedSecurity: &uimodel.AdvancedSecurity{
			PrivateVulnerabilityReporting: uiConfigPtr(false), SecretProtection: uiConfigPtr(false), PushProtection: uiConfigPtr(false),
		}},
	}}
	got, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("page/section decoding changed controls: %#v", got)
	}
	encoded, err := EncodeConfig(want)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"immutable_release: false", "policy: allow_owner_and_select_non_owner_actions_and_reusable_workflows", "default_permissions: read_repository_contents_and_packages_permissions", "enabled: false", "push_protection: false"} {
		if !strings.Contains(string(encoded), required) {
			t.Errorf("export missing %q: %s", required, encoded)
		}
	}
	for _, forbidden := range []string{"security_and_quality:", "enable_release_immutability:", "selection:"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Errorf("export contains obsolete key %q: %s", forbidden, encoded)
		}
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("page/section round trip changed intent: %s, %v", encoded, err)
	}
}

func TestUIPageSectionSchemaPreservesIndependentOmission(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        actions_permissions:
          require_actions_to_be_pinned_to_a_full_length_commit_sha: false
        workflow_permissions:
          allow_github_actions_to_create_and_approve_pull_requests: false
    advanced_security:
      secret_protection:
        push_protection: false
`
	got, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	actions := got.Settings.Actions.General
	security := got.Settings.SecurityAndQuality.AdvancedSecurity
	if actions.ActionsPermissions != nil || actions.WorkflowPermissions != nil || security.SecretProtection != nil {
		t.Fatal("a section companion populated an omitted parent choice")
	}
	if actions.RequireActionsToBePinnedToAFullLengthCommitSHA == nil || *actions.RequireActionsToBePinnedToAFullLengthCommitSHA ||
		actions.AllowGitHubActionsToCreateAndApprovePullRequests == nil || *actions.AllowGitHubActionsToCreateAndApprovePullRequests ||
		security.PushProtection == nil || *security.PushProtection {
		t.Fatal("explicit false companion intent was lost")
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"policy:", "default_permissions:", "enabled:"} {
		if strings.Contains(string(encoded), absent) {
			t.Errorf("export invented %q: %s", absent, encoded)
		}
	}
}

func TestUIPageSectionSchemaRejectsSupersededShapes(t *testing.T) {
	for name, body := range map[string]string{
		"navigation category":         "security_and_quality: {advanced_security: {secret_protection: true}}",
		"old release key":             "general: {releases: {enable_release_immutability: true}}",
		"scalar actions policy":       "actions: {general: {actions_permissions: allow_all_actions_and_reusable_workflows}}",
		"scalar workflow permissions": "actions: {general: {workflow_permissions: read_and_write_permissions}}",
		"flat SHA control":            "actions: {general: {require_actions_to_be_pinned_to_a_full_length_commit_sha: true}}",
		"flat GitHub allowlist":       "actions: {general: {allow_actions_created_by_github: true}}",
		"flat verified allowlist":     "actions: {general: {allow_actions_by_marketplace_verified_creators: true}}",
		"flat action patterns":        "actions: {general: {allow_or_block_specified_actions_and_reusable_workflows: []}}",
		"flat PR approval":            "actions: {general: {allow_github_actions_to_create_and_approve_pull_requests: true}}",
		"scalar secret protection":    "advanced_security: {secret_protection: true}",
		"flat push protection":        "advanced_security: {push_protection: true}",
		"generic selection":           "actions: {general: {actions_permissions: {selection: disable_actions}}}",
		"generic value":               "actions: {general: {workflow_permissions: {value: read_and_write_permissions}}}",
		"second actions enabled flag": "actions: {general: {actions_permissions: {enabled: false}}}",
	} {
		t.Run(name, func(t *testing.T) {
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {" + body + "}}\n"
			if _, err := DecodeConfig([]byte(input)); err == nil {
				t.Fatalf("accepted superseded or invented public shape: %s", input)
			}
		})
	}
}
