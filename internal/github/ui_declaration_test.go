package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestDeclarationFromUIAboutAndGeneral(t *testing.T) {
	empty, website, branch := "", "https://example.com", "main"
	yes, no := true, false
	choice := uimodel.PullRequestTitle
	topics := []string{"settings", "github"}
	ui := uimodel.Repository{
		About: &uimodel.About{Description: &empty, Website: &website, Topics: &topics},
		Settings: &uimodel.RepositorySettings{General: &uimodel.General{
			TemplateRepository: &no, DefaultBranch: &branch,
			Features:     uimodel.Features{Issues: &yes},
			PullRequests: uimodel.PullRequests{AllowMergeCommits: &yes, MergeCommit: uimodel.CommitMessage{DefaultCommitMessage: &choice}},
		}},
	}
	title, message := "PR_TITLE", "BLANK"
	wantTopics := []string{"github", "settings"}
	want := settings.Declaration{
		Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &website, IsTemplate: &no, DefaultBranch: &branch, HasIssues: &yes, AllowMergeCommit: &yes, MergeCommitTitle: &title, MergeCommitMessage: &message},
		Topics:     settings.TopicsDeclaration{Names: &wantTopics},
	}
	got, err := DeclarationFromUI(ui)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("declaration %#v, error %v; want %#v", got, err, want)
	}
}

func TestDeclarationFromUIEmptyGroups(t *testing.T) {
	for _, ui := range []uimodel.Repository{{}, {About: &uimodel.About{}}, {Settings: &uimodel.RepositorySettings{General: &uimodel.General{}}}} {
		got, err := DeclarationFromUI(ui)
		if err != nil || !got.Empty() {
			t.Fatalf("omitted controls became intent: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUIRejectsInvalidChoiceAndDependencies(t *testing.T) {
	validAbout := "preserve only on success"
	apiChoice := uimodel.CommitMessageChoice("PR_TITLE")
	creation := uimodel.AllUsers
	no := false
	for _, general := range []uimodel.General{
		{PullRequests: uimodel.PullRequests{MergeCommit: uimodel.CommitMessage{DefaultCommitMessage: &apiChoice}}},
		{Features: uimodel.Features{Issues: &no, IssuePermissions: uimodel.CreationPermissions{CreationAllowedBy: &creation}}},
	} {
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &validAbout}, Settings: &uimodel.RepositorySettings{General: &general}})
		if err == nil || !got.Empty() {
			t.Fatalf("accepted invalid UI intent or returned partial declaration: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUILabelIdentityAndColor(t *testing.T) {
	id, selectedColor, newColor, empty := int64(17), "#ABCDEF", "#123456", ""
	labels := []uimodel.Label{{ID: &id, Name: "renamed", Color: &selectedColor, Description: &empty}, {Name: "new", Color: &newColor}}
	got, err := DeclarationFromUI(uimodel.Repository{Labels: &labels})
	if err != nil || got.Labels == nil {
		t.Fatalf("missing converted labels: %v", err)
	}
	values := got.Labels.Values()
	if len(values) != 2 || values[0].ID == nil || *values[0].ID != 17 || values[0].Name != "renamed" || values[0].Color == nil || *values[0].Color != "abcdef" || values[0].Description == nil || *values[0].Description != "" || values[1].ID != nil || values[1].Color == nil || *values[1].Color != "123456" || values[1].Description != nil {
		t.Fatalf("label conversion changed identity/presence: %#v", values)
	}
	if *labels[0].Color != "#ABCDEF" {
		t.Fatal("conversion modified UI intent")
	}
}

func TestDeclarationFromUILabelCollectionPresence(t *testing.T) {
	empty := []uimodel.Label{}
	for _, labels := range []*[]uimodel.Label{nil, &empty} {
		got, err := DeclarationFromUI(uimodel.Repository{Labels: labels})
		if err != nil || (got.Labels != nil) != (labels != nil) || (got.Labels != nil && got.Labels.Len() != 0) {
			t.Fatalf("collection presence changed: %#v, %v", got.Labels, err)
		}
	}
}

func TestDeclarationFromUILabelInvalidIntentIsAtomic(t *testing.T) {
	id, color, noPrefix, description := int64(17), "#abcdef", "abcdef", "valid About"
	for _, labels := range [][]uimodel.Label{
		{{Name: "bad-color", Color: &noPrefix}},
		{{ID: &id, Name: "one", Color: &color}, {ID: &id, Name: "two", Color: &color}},
	} {
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Labels: &labels})
		if err == nil || !got.Empty() {
			t.Fatalf("accepted invalid labels or returned partial intent: empty=%v, error=%v", got.Empty(), err)
		}
	}
}

func TestDeclarationFromUIRulesetWholeObject(t *testing.T) {
	entries := []uimodel.Ruleset{{ID: rulesetPtr(int64(7)), RulesetName: "main", EnforcementStatus: rulesetPtr(uimodel.Active), TargetBranches: &uimodel.BranchTargets{IncludeDefaultBranch: rulesetPtr(true), IncludeByPattern: rulesetPtr([]string{"release/*"})}, Rules: uimodel.RulesetRules{RestrictDeletions: rulesetPtr(false), BlockForcePushes: rulesetPtr(true)}}, {RulesetName: "tags", EnforcementStatus: rulesetPtr(uimodel.Disabled), TargetTags: &uimodel.TagTargets{IncludeAllTags: rulesetPtr(true)}}}
	got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Rulesets: &entries}})
	if err != nil || got.Rulesets == nil {
		t.Fatalf("missing rulesets: %v", err)
	}
	want := []settings.RulesetDeclaration{
		{ID: rulesetPtr(int64(7)), Name: "main", Target: rulesetPtr("branch"), Enforcement: rulesetPtr("active"), Conditions: &settings.RulesetConditions{RefName: &settings.RefNameCondition{Include: rulesetPtr([]string{"refs/heads/release/*", "~DEFAULT_BRANCH"}), Exclude: rulesetPtr([]string{})}}, BypassActors: rulesetPtr([]settings.BypassActor{}), Rules: rulesetPtr([]settings.RulesetRule{{Type: "non_fast_forward"}})},
		{Name: "tags", Target: rulesetPtr("tag"), Enforcement: rulesetPtr("disabled"), Conditions: &settings.RulesetConditions{RefName: &settings.RefNameCondition{Include: rulesetPtr([]string{"~ALL"}), Exclude: rulesetPtr([]string{})}}, BypassActors: rulesetPtr([]settings.BypassActor{}), Rules: rulesetPtr([]settings.RulesetRule{})},
	}
	if !reflect.DeepEqual(got.Rulesets.Values(), want) {
		t.Fatalf("rulesets %#v; want %#v", got.Rulesets.Values(), want)
	}
}

func TestDeclarationFromUIRulesetCollectionPresence(t *testing.T) {
	empty := []uimodel.Ruleset{}
	for _, entries := range []*[]uimodel.Ruleset{nil, &empty} {
		got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Rulesets: entries}})
		if err != nil || (got.Rulesets != nil) != (entries != nil) || (got.Rulesets != nil && len(got.Rulesets.Values()) != 0) {
			t.Fatalf("rulesets presence changed: %#v, %v", got.Rulesets, err)
		}
	}
}

func TestDeclarationFromUIRulesetInvalidIntentIsAtomic(t *testing.T) {
	description := "valid About"
	for _, entry := range []uimodel.Ruleset{
		{RulesetName: "missing enforcement", TargetBranches: &uimodel.BranchTargets{}},
		{RulesetName: "missing target", EnforcementStatus: rulesetPtr(uimodel.Active)},
	} {
		entries := []uimodel.Ruleset{entry}
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Settings: &uimodel.RepositorySettings{Rulesets: &entries}})
		if err == nil || !got.Empty() {
			t.Fatalf("accepted invalid rulesets or partial intent: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUIActions(t *testing.T) {
	ui := uimodel.ActionsGeneral{
		ActionsPermissions: rulesetPtr(uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows),
		RequireActionsToBePinnedToAFullLengthCommitSHA: rulesetPtr(true), AllowActionsCreatedByGitHub: rulesetPtr(false), AllowActionsByMarketplaceVerifiedCreators: rulesetPtr(true),
		AllowOrBlockSpecifiedActionsAndReusableWorkflows: rulesetPtr([]string{"example/a@v1", "!example/b@*", "example/a@v1"}),
		WorkflowPermissions: rulesetPtr(uimodel.ReadAndWritePermissions), AllowGitHubActionsToCreateAndApprovePullRequests: rulesetPtr(false),
		CheckWorkflowRunStatusArtifactAndLogRetention: rulesetPtr(30), ApprovalForRunningForkPullRequestWorkflowsFromContributors: rulesetPtr(uimodel.RequireApprovalForAllExternalContributors),
		ForkPullRequestWorkflows: &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: rulesetPtr(true), SendWriteTokensToWorkflowsFromForkPullRequests: rulesetPtr(false)},
	}
	want := settings.ActionsPolicyDeclaration{Enabled: rulesetPtr(true), AllowedActions: rulesetPtr("selected"), SHAPinningRequired: rulesetPtr(true), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: rulesetPtr(false), VerifiedAllowed: rulesetPtr(true), PatternsAllowed: rulesetPtr([]string{"example/a@v1", "!example/b@*", "example/a@v1"})}, DefaultWorkflowPermissions: rulesetPtr("write"), CanApprovePullRequestReviews: rulesetPtr(false), RetentionDays: rulesetPtr(30), ForkApprovalPolicy: rulesetPtr("all_external_contributors"), PrivateFork: settings.PrivateForkDeclaration{Run: rulesetPtr(true), WriteTokens: rulesetPtr(false)}}
	got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Actions: &uimodel.ActionsSettings{General: &ui}}})
	if err != nil || !reflect.DeepEqual(got.Actions, want) {
		t.Fatalf("Actions %#v; want %#v; error=%v", got.Actions, want, err)
	}
	for _, actions := range []*uimodel.ActionsSettings{nil, {}, {General: &uimodel.ActionsGeneral{}}} {
		got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Actions: actions}})
		if err != nil || !got.Empty() {
			t.Fatalf("empty Actions became intent: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUIActionsInvalidIntentIsAtomic(t *testing.T) {
	description := "valid About"
	for _, actions := range []uimodel.ActionsGeneral{
		{WorkflowPermissions: rulesetPtr(uimodel.WorkflowPermissionChoice("write"))},
		{CheckWorkflowRunStatusArtifactAndLogRetention: rulesetPtr(0)},
		{CheckWorkflowRunStatusArtifactAndLogRetention: rulesetPtr(401)},
		{ActionsPermissions: rulesetPtr(uimodel.AllowAllActionsAndReusableWorkflows), AllowActionsCreatedByGitHub: rulesetPtr(false)},
	} {
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Settings: &uimodel.RepositorySettings{Actions: &uimodel.ActionsSettings{General: &actions}}})
		if err == nil || !got.Empty() {
			t.Fatalf("invalid Actions accepted or partial intent returned: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUISecurityComposition(t *testing.T) {
	description, branch := "About retained", "main"
	advanced := &uimodel.AdvancedSecurity{SecretProtection: rulesetPtr(true), PushProtection: rulesetPtr(false), PrivateVulnerabilityReporting: rulesetPtr(true)}
	releases := &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: rulesetPtr(false)}
	for _, test := range []struct {
		name     string
		general  *uimodel.General
		security *uimodel.SecurityAndQuality
		want     settings.SecurityDeclaration
	}{
		{"both", &uimodel.General{DefaultBranch: &branch, Releases: releases}, &uimodel.SecurityAndQuality{AdvancedSecurity: advanced}, settings.SecurityDeclaration{ImmutableReleases: rulesetPtr(false), SecretScanning: rulesetPtr(true), PushProtection: rulesetPtr(false), PrivateVulnerabilityReporting: rulesetPtr(true)}},
		{"release only", &uimodel.General{Releases: releases}, nil, settings.SecurityDeclaration{ImmutableReleases: rulesetPtr(false)}},
		{"security only", nil, &uimodel.SecurityAndQuality{AdvancedSecurity: advanced}, settings.SecurityDeclaration{SecretScanning: rulesetPtr(true), PushProtection: rulesetPtr(false), PrivateVulnerabilityReporting: rulesetPtr(true)}},
		{"empty", &uimodel.General{Releases: &uimodel.SecurityGeneralReleases{}}, &uimodel.SecurityAndQuality{}, settings.SecurityDeclaration{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Settings: &uimodel.RepositorySettings{General: test.general, SecurityAndQuality: test.security}})
			if err != nil || !reflect.DeepEqual(got.Security, test.want) || got.Repository.Description == nil || *got.Repository.Description != description {
				t.Fatalf("Security composition changed: %#v, %v", got, err)
			}
			if test.name == "both" && (got.Repository.DefaultBranch == nil || *got.Repository.DefaultBranch != branch) {
				t.Fatal("Security composition dropped General")
			}
		})
	}
}

func TestDeclarationFromUISecurityInvalidIntentIsAtomic(t *testing.T) {
	description := "valid About"
	got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Settings: &uimodel.RepositorySettings{SecurityAndQuality: &uimodel.SecurityAndQuality{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: rulesetPtr(false), PushProtection: rulesetPtr(true)}}}})
	if err == nil || !got.Empty() {
		t.Fatalf("invalid Security accepted or partial intent returned: %#v, %v", got, err)
	}
}

func TestDeclarationFromUIPages(t *testing.T) {
	for _, domain := range []*string{nil, rulesetPtr("docs.example.com")} {
		pages := &uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: rulesetPtr(uimodel.DeployFromABranch), Branch: rulesetPtr("pages"), SelectFolder: rulesetPtr(uimodel.PagesDocs)}, CustomDomain: &uimodel.PagesCustomDomain{Value: domain}, EnforceHTTPS: rulesetPtr(false)}
		want := settings.PagesDeclaration{BuildType: rulesetPtr("legacy"), Source: &settings.PagesSource{Branch: "pages", Path: "/docs"}, CNAME: &settings.PagesDomain{Value: domain}, HTTPSEnforced: rulesetPtr(false)}
		got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Pages: pages}})
		if err != nil || !reflect.DeepEqual(got.Pages, want) {
			t.Fatalf("Pages %#v; want %#v; error=%v", got.Pages, want, err)
		}
	}
	for _, pages := range []*uimodel.Pages{nil, {}} {
		got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Pages: pages}})
		if err != nil || !got.Empty() {
			t.Fatalf("empty Pages became intent: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUIPagesInvalidIntentIsAtomic(t *testing.T) {
	description := "valid About"
	for _, pages := range []uimodel.Pages{
		{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: rulesetPtr(uimodel.PagesSourceChoice("workflow"))}},
		{CustomDomain: &uimodel.PagesCustomDomain{Value: rulesetPtr("")}},
		{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Branch: rulesetPtr("main")}},
		{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: rulesetPtr(uimodel.GitHubActions), Branch: rulesetPtr("main"), SelectFolder: rulesetPtr(uimodel.PagesRoot)}},
	} {
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Settings: &uimodel.RepositorySettings{Pages: &pages}})
		if err == nil || !got.Empty() {
			t.Fatalf("invalid Pages accepted or partial intent returned: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUIEnvironments(t *testing.T) {
	entries := []uimodel.Environment{
		{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: rulesetPtr(0), RequiredReviewers: rulesetPtr([]uimodel.EnvironmentReviewer{{Person: rulesetPtr(int64(11))}, {Team: rulesetPtr(int64(12))}}), PreventSelfReview: rulesetPtr(false), AllowAdministratorsToBypassConfiguredProtectionRules: rulesetPtr(true)}, DeploymentBranchesAndTags: rulesetPtr(uimodel.SelectedBranchesAndTags), DeploymentBranchOrTagRules: rulesetPtr([]uimodel.EnvironmentDeploymentRule{{RefType: uimodel.EnvironmentBranch, NamePattern: "Release/*"}, {RefType: uimodel.EnvironmentTag, NamePattern: "Release/*"}})},
		{Name: "Open", DeploymentBranchesAndTags: rulesetPtr(uimodel.NoRestriction)},
	}
	want := []settings.EnvironmentDeclaration{
		{Name: "Open", DeploymentBranchPolicyDeclared: true},
		{Name: "Production", WaitTimer: rulesetPtr(0), Reviewers: rulesetPtr([]settings.EnvironmentReviewer{{Type: "Team", ID: 12}, {Type: "User", ID: 11}}), PreventSelfReview: rulesetPtr(false), CanAdminsBypass: rulesetPtr(true), DeploymentBranchPolicyDeclared: true, DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, DeploymentBranchPatterns: rulesetPtr([]settings.EnvironmentPattern{{Type: "branch", Name: "Release/*"}, {Type: "tag", Name: "Release/*"}})},
	}
	got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Environments: &entries}})
	if err != nil || got.Environments == nil || !reflect.DeepEqual(got.Environments.Values(), want) {
		t.Fatalf("Environment composition changed: %#v, %v", got.Environments, err)
	}
	empty := []uimodel.Environment{}
	for _, collection := range []*[]uimodel.Environment{nil, &empty} {
		got, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{Environments: collection}})
		if err != nil || (got.Environments != nil) != (collection != nil) || (got.Environments != nil && got.Environments.Len() != 0) {
			t.Fatalf("Environment collection presence changed: %#v, %v", got.Environments, err)
		}
	}
}

func TestDeclarationFromUIEnvironmentsInvalidIntentIsAtomic(t *testing.T) {
	description := "valid About"
	for _, entries := range [][]uimodel.Environment{
		{{Name: "one", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{RequiredReviewers: rulesetPtr([]uimodel.EnvironmentReviewer{{Person: rulesetPtr(int64(11)), Team: rulesetPtr(int64(12))}})}}},
		{{Name: "Production"}, {Name: "production"}},
		{{Name: "one", DeploymentBranchesAndTags: rulesetPtr(uimodel.NoRestriction), DeploymentBranchOrTagRules: rulesetPtr([]uimodel.EnvironmentDeploymentRule{})}},
	} {
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Settings: &uimodel.RepositorySettings{Environments: &entries}})
		if err == nil || !got.Empty() {
			t.Fatalf("invalid Environment accepted or partial intent returned: %#v, %v", got, err)
		}
	}
}
