package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIConfigEnvironmentsControls(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: Production
        deployment_protection_rules:
          wait_timer: 0
          required_reviewers: [{person: 11}, {team: 12}]
          prevent_self_review: false
          allow_administrators_to_bypass_configured_protection_rules: true
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules:
          - {ref_type: branch, name_pattern: 'Release/*'}
          - {ref_type: tag, name_pattern: 'Release/*'}
      - name: Staging
        deployment_branches_and_tags: protected_branches_only
      - name: Open
        deployment_branches_and_tags: no_restriction
        deployment_protection_rules: {required_reviewers: []}
`
	want := []uimodel.Environment{
		{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: uiConfigPtr(0), RequiredReviewers: uiConfigPtr([]uimodel.EnvironmentReviewer{{Person: uiConfigPtr(int64(11))}, {Team: uiConfigPtr(int64(12))}}), PreventSelfReview: uiConfigPtr(false), AllowAdministratorsToBypassConfiguredProtectionRules: uiConfigPtr(true)}, DeploymentBranchesAndTags: uiConfigPtr(uimodel.SelectedBranchesAndTags), DeploymentBranchOrTagRules: uiConfigPtr([]uimodel.EnvironmentDeploymentRule{{RefType: uimodel.EnvironmentBranch, NamePattern: "Release/*"}, {RefType: uimodel.EnvironmentTag, NamePattern: "Release/*"}})},
		{Name: "Staging", DeploymentBranchesAndTags: uiConfigPtr(uimodel.ProtectedBranchesOnly)},
		{Name: "Open", DeploymentBranchesAndTags: uiConfigPtr(uimodel.NoRestriction), DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{RequiredReviewers: uiConfigPtr([]uimodel.EnvironmentReviewer{})}},
	}
	got, err := DecodeConfig([]byte(input))
	if err != nil || !reflect.DeepEqual(*got.Settings.Environments, want) {
		t.Fatalf("Environment controls changed: %#v, %v", got, err)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("Environment round trip: %s, %v", encoded, err)
	}
}

func TestUIConfigEnvironmentCollectionPresence(t *testing.T) {
	for _, input := range []string{"{}", "{environments: []}"} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + input + "}\n"))
		if err != nil || (got.Settings.Environments != nil) != (input != "{}") {
			t.Fatalf("Environment collection presence changed: %#v, %v", got, err)
		}
	}
}

func TestUIConfigEnvironmentsRejectsInvalidSyntax(t *testing.T) {
	for _, entry := range []string{
		"id: 7", "name: null", "wait_timer: 0", "reviewers: []", "deployment_branch_policy: null",
		"deployment_protection_rules: {wait_timer: '0'}", "deployment_protection_rules: {required_reviewers: [{type: User, id: 11}]}",
		"deployment_protection_rules: {required_reviewers: null}", "deployment_branch_or_tag_rules: [{type: branch, name: main}]",
	} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {environments: [{" + entry + "}]}}\n")); err == nil {
			t.Fatalf("accepted invalid Environment syntax: %s", entry)
		}
	}
}
