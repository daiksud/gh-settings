package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func uiExportBase() settings.Declaration {
	return settings.Declaration{Repository: settings.RepositoryDeclaration{Description: rulesetPtr(""), Homepage: rulesetPtr("https://example.com")}, Topics: settings.TopicsDeclaration{Names: rulesetPtr([]string{"settings", "github"})}}
}

func TestUIFromExportDeclarationAboutGeneralLabels(t *testing.T) {
	source := uiExportBase()
	source.Repository.HasIssues, source.Repository.AllowMergeCommit = rulesetPtr(false), rulesetPtr(false)
	source.Repository.IssueCreationPolicy = rulesetPtr("COLLABORATORS_ONLY")
	source.Repository.MergeCommitTitle, source.Repository.MergeCommitMessage = rulesetPtr("PR_TITLE"), rulesetPtr("BLANK")
	labels, err := settings.NewLabelDeclarations([]settings.LabelDeclaration{{ID: rulesetPtr(int64(17)), Name: "bug", Color: rulesetPtr("abcdef"), Description: rulesetPtr("")}})
	if err != nil {
		t.Fatal(err)
	}
	source.Labels = &labels
	want := uimodel.Repository{About: &uimodel.About{Description: rulesetPtr(""), Website: rulesetPtr("https://example.com"), Topics: rulesetPtr([]string{"github", "settings"})}, Labels: rulesetPtr([]uimodel.Label{{ID: rulesetPtr(int64(17)), Name: "bug", Color: rulesetPtr("#abcdef"), Description: rulesetPtr("")}}), Settings: &uimodel.RepositorySettings{General: &uimodel.General{Features: uimodel.Features{Issues: rulesetPtr(false)}, PullRequests: uimodel.PullRequests{AllowMergeCommits: rulesetPtr(false)}}}}
	got, err := UIFromExportDeclaration(source)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("export UI %#v; want %#v; error=%v", got, want, err)
	}
	*got.About.Description = "changed"
	(*got.About.Topics)[0] = "changed"
	*got.Settings.General.Features.Issues = true
	if *source.Repository.Description != "" || (*source.Topics.Names)[0] != "settings" || *source.Repository.HasIssues {
		t.Fatal("export composition aliases source")
	}
}

func TestUIFromExportDeclarationLabelPresence(t *testing.T) {
	empty, err := settings.NewLabelDeclarations([]settings.LabelDeclaration{})
	if err != nil {
		t.Fatal(err)
	}
	for _, labels := range []*settings.LabelDeclarations{nil, &empty} {
		source := uiExportBase()
		source.Labels = labels
		got, err := UIFromExportDeclaration(source)
		if err != nil || (got.Labels != nil) != (labels != nil) || (got.Labels != nil && len(*got.Labels) != 0) {
			t.Fatalf("export label presence changed: %#v, %v", got, err)
		}
	}
}

func TestUIFromExportDeclarationRejectsIncompleteAndInvalidSources(t *testing.T) {
	for _, mutate := range []func(*settings.Declaration){
		func(d *settings.Declaration) { d.Repository.Description = nil },
		func(d *settings.Declaration) { d.Repository.Homepage = nil },
		func(d *settings.Declaration) { d.Topics.Names = nil },
		func(d *settings.Declaration) {
			d.Repository.MergeCommitTitle, d.Repository.MergeCommitMessage = rulesetPtr("UNKNOWN"), rulesetPtr("BLANK")
		},
		func(d *settings.Declaration) {
			labels, err := settings.NewLabelDeclarations([]settings.LabelDeclaration{{ID: rulesetPtr(int64(17)), Name: "bug", Color: rulesetPtr("abcdef")}})
			if err != nil {
				t.Fatal(err)
			}
			d.Labels = &labels
		},
	} {
		source := uiExportBase()
		mutate(&source)
		got, err := UIFromExportDeclaration(source)
		if err == nil || !reflect.DeepEqual(got, uimodel.Repository{}) {
			t.Fatalf("invalid export returned partial UI: %#v, %v", got, err)
		}
	}
}

func exportRulesetPolicies(t *testing.T, states []settings.RulesetState) *settings.RulesetDeclarations {
	t.Helper()
	observed, err := settings.NewRulesetObservation(states, 0)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := observed.ExportDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	return &exported
}

func TestUIFromExportDeclarationRulesets(t *testing.T) {
	policy := apiRulesetBase()
	*policy.Conditions.RefName.Include = []string{"~DEFAULT_BRANCH"}
	source := uiExportBase()
	source.Rulesets = exportRulesetPolicies(t, []settings.RulesetState{{ID: 7, Policy: policy}})
	want := []uimodel.Ruleset{{ID: rulesetPtr(int64(7)), RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.Active), TargetBranches: &uimodel.BranchTargets{IncludeDefaultBranch: rulesetPtr(true), IncludeAllBranches: rulesetPtr(false), IncludeByPattern: rulesetPtr([]string{}), ExcludeByPattern: rulesetPtr([]string{})}, BypassList: rulesetPtr([]uimodel.BypassEntry{}), Rules: uimodel.RulesetRules{RestrictDeletions: rulesetPtr(false), BlockForcePushes: rulesetPtr(false), RequireLinearHistory: rulesetPtr(false), RequireSignedCommits: rulesetPtr(false)}}}
	got, err := UIFromExportDeclaration(source)
	if err != nil || got.Settings == nil || got.Settings.Rulesets == nil || !reflect.DeepEqual(*got.Settings.Rulesets, want) {
		t.Fatalf("export Ruleset UI changed: %#v, %v", got, err)
	}
	for _, rulesets := range []*settings.RulesetDeclarations{nil, exportRulesetPolicies(t, []settings.RulesetState{})} {
		source.Rulesets = rulesets
		got, err := UIFromExportDeclaration(source)
		present := got.Settings != nil && got.Settings.Rulesets != nil
		if err != nil || present != (rulesets != nil) || (present && len(*got.Settings.Rulesets) != 0) {
			t.Fatalf("Ruleset export presence changed: %#v, %v", got, err)
		}
	}
}

func TestUIFromExportDeclarationRulesetWriteOnlyAndUnsupportedValues(t *testing.T) {
	policy := apiRulesetBase()
	parameters := completeUIAPIPR()
	parameters.DismissalRestriction = &settings.ReviewDismissalRestriction{Enabled: rulesetPtr(true), AllowedActors: rulesetPtr([]settings.ReviewActor{{Type: "Team", ID: 12}})}
	policy.Rules = rulesetPtr([]settings.RulesetRule{{Type: "pull_request", Parameters: parameters}})
	source := uiExportBase()
	source.Rulesets = exportRulesetPolicies(t, []settings.RulesetState{{ID: 7, Policy: policy}})
	if source.Rulesets.Values()[0].HasWriteOnlyDismissal() {
		t.Fatal("existing export boundary retained write-only dismissal")
	}
	got, err := UIFromExportDeclaration(source)
	if err != nil || got.Settings == nil || got.Settings.Rulesets == nil {
		t.Fatalf("missing exported Ruleset: %#v, %v", got, err)
	}
	pr := (*got.Settings.Rulesets)[0].Rules.RequireAPullRequestBeforeMerging
	if pr == nil || pr.RestrictWhoCanDismissPullRequestReviews != nil || pr.PeopleTeamsOrAppsThatCanDismissReviews != nil {
		t.Fatalf("write-only dismissal appeared in UI export: %#v", pr)
	}
	for _, mutate := range []func(*settings.RulesetDeclaration){
		func(p *settings.RulesetDeclaration) { p.Enforcement = rulesetPtr("evaluate") },
		func(p *settings.RulesetDeclaration) {
			p.BypassActors = rulesetPtr([]settings.BypassActor{{ActorType: "RepositoryRole", ActorID: rulesetPtr(int64(99)), BypassMode: rulesetPtr("always")}})
		},
	} {
		unsupported := apiRulesetBase()
		unsupported.Name = "Zulu"
		mutate(&unsupported)
		source.Rulesets = exportRulesetPolicies(t, []settings.RulesetState{{ID: 7, Policy: apiRulesetBase()}, {ID: 8, Policy: unsupported}})
		got, err := UIFromExportDeclaration(source)
		if err == nil || !reflect.DeepEqual(got, uimodel.Repository{}) {
			t.Fatalf("unsupported Ruleset returned partial export: %#v, %v", got, err)
		}
	}
}

func TestUIFromExportDeclarationActions(t *testing.T) {
	source := uiExportBase()
	source.Actions = settings.ActionsPolicyDeclaration{Enabled: rulesetPtr(true), AllowedActions: rulesetPtr("selected"), SHAPinningRequired: rulesetPtr(false), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: rulesetPtr(false), VerifiedAllowed: rulesetPtr(true), PatternsAllowed: rulesetPtr([]string{"example/a@v1", "!example/b@*", "example/a@v1"})}, DefaultWorkflowPermissions: rulesetPtr("read"), CanApprovePullRequestReviews: rulesetPtr(false), RetentionDays: rulesetPtr(30), ForkApprovalPolicy: rulesetPtr("first_time_contributors"), PrivateFork: settings.PrivateForkDeclaration{Run: rulesetPtr(true), WriteTokens: rulesetPtr(false), Secrets: rulesetPtr(true), Approval: rulesetPtr(false)}}
	want := &uimodel.ActionsGeneral{ActionsPermissions: rulesetPtr(uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows), RequireActionsToBePinnedToAFullLengthCommitSHA: rulesetPtr(false), AllowActionsCreatedByGitHub: rulesetPtr(false), AllowActionsByMarketplaceVerifiedCreators: rulesetPtr(true), AllowOrBlockSpecifiedActionsAndReusableWorkflows: rulesetPtr([]string{"example/a@v1", "!example/b@*", "example/a@v1"}), WorkflowPermissions: rulesetPtr(uimodel.ReadRepositoryContentsAndPackagesPermissions), AllowGitHubActionsToCreateAndApprovePullRequests: rulesetPtr(false), CheckWorkflowRunStatusArtifactAndLogRetention: rulesetPtr(30), ApprovalForRunningForkPullRequestWorkflowsFromContributors: rulesetPtr(uimodel.RequireApprovalForFirstTimeContributors), ForkPullRequestWorkflows: &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: rulesetPtr(true), SendWriteTokensToWorkflowsFromForkPullRequests: rulesetPtr(false), SendSecretsAndVariablesToWorkflowsFromForkPullRequests: rulesetPtr(true), RequireApprovalForForkPullRequestWorkflows: rulesetPtr(false)}}
	got, err := UIFromExportDeclaration(source)
	if err != nil || got.Settings == nil || got.Settings.Actions == nil || !reflect.DeepEqual(got.Settings.Actions.General, want) {
		t.Fatalf("Actions export changed: %#v, %v", got, err)
	}
	*source.Actions.Enabled, *source.Actions.PrivateFork.Run = false, false
	got, err = UIFromExportDeclaration(source)
	want.ActionsPermissions = rulesetPtr(uimodel.DisableActions)
	want.AllowActionsCreatedByGitHub, want.AllowActionsByMarketplaceVerifiedCreators, want.AllowOrBlockSpecifiedActionsAndReusableWorkflows = nil, nil, nil
	want.ForkPullRequestWorkflows = &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: rulesetPtr(false)}
	if err != nil || got.Settings == nil || got.Settings.Actions == nil || !reflect.DeepEqual(got.Settings.Actions.General, want) {
		t.Fatalf("disabled latent Actions leaked: %#v, %v", got, err)
	}
}

func TestUIFromExportDeclarationActionsScopesAndErrors(t *testing.T) {
	for _, test := range []struct {
		policy settings.ActionsPolicyDeclaration
		want   *uimodel.ActionsGeneral
	}{
		{settings.ActionsPolicyDeclaration{}, nil},
		{settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: rulesetPtr("write"), CanApprovePullRequestReviews: rulesetPtr(false)}, &uimodel.ActionsGeneral{WorkflowPermissions: rulesetPtr(uimodel.ReadAndWritePermissions), AllowGitHubActionsToCreateAndApprovePullRequests: rulesetPtr(false)}},
		{settings.ActionsPolicyDeclaration{RetentionDays: rulesetPtr(30)}, &uimodel.ActionsGeneral{CheckWorkflowRunStatusArtifactAndLogRetention: rulesetPtr(30)}},
	} {
		source := uiExportBase()
		source.Actions = test.policy
		got, err := UIFromExportDeclaration(source)
		var general *uimodel.ActionsGeneral
		if got.Settings != nil && got.Settings.Actions != nil {
			general = got.Settings.Actions.General
		}
		if err != nil || !reflect.DeepEqual(general, test.want) {
			t.Fatalf("Actions scope changed: %#v, %v", got, err)
		}
	}
	for _, policy := range []settings.ActionsPolicyDeclaration{
		{Enabled: rulesetPtr(true), AllowedActions: rulesetPtr("all"), SHAPinningRequired: rulesetPtr(false), RetentionDays: rulesetPtr(0)},
		{DefaultWorkflowPermissions: rulesetPtr("read"), CanApprovePullRequestReviews: rulesetPtr(false), ForkApprovalPolicy: rulesetPtr("unknown")},
		{PrivateFork: settings.PrivateForkDeclaration{Run: rulesetPtr(true)}},
	} {
		source := uiExportBase()
		source.Actions = policy
		got, err := UIFromExportDeclaration(source)
		if err == nil || !reflect.DeepEqual(got, uimodel.Repository{}) {
			t.Fatalf("invalid Actions returned partial export: %#v, %v", got, err)
		}
	}
}

func TestUIFromExportDeclarationSecurity(t *testing.T) {
	for _, test := range []struct {
		name     string
		policy   settings.SecurityDeclaration
		advanced *uimodel.AdvancedSecurity
		releases *uimodel.SecurityGeneralReleases
	}{
		{"both", settings.SecurityDeclaration{ImmutableReleases: rulesetPtr(false), SecretScanning: rulesetPtr(true), PushProtection: rulesetPtr(false), PrivateVulnerabilityReporting: rulesetPtr(true)}, &uimodel.AdvancedSecurity{SecretProtection: rulesetPtr(true), PushProtection: rulesetPtr(false), PrivateVulnerabilityReporting: rulesetPtr(true)}, &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: rulesetPtr(false)}},
		{"release only", settings.SecurityDeclaration{ImmutableReleases: rulesetPtr(false)}, nil, &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: rulesetPtr(false)}},
		{"security only", settings.SecurityDeclaration{PrivateVulnerabilityReporting: rulesetPtr(false)}, &uimodel.AdvancedSecurity{PrivateVulnerabilityReporting: rulesetPtr(false)}, nil},
		{"omitted", settings.SecurityDeclaration{}, nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := uiExportBase()
			source.Repository.DefaultBranch = rulesetPtr("main")
			source.Security = test.policy
			got, err := UIFromExportDeclaration(source)
			if err != nil || got.Settings == nil || got.Settings.General == nil || got.Settings.General.DefaultBranch == nil || *got.Settings.General.DefaultBranch != "main" || !reflect.DeepEqual(got.Settings.General.Releases, test.releases) {
				t.Fatalf("Security composition dropped General or Releases: %#v, %v", got, err)
			}
			var advanced *uimodel.AdvancedSecurity
			if got.Settings.SecurityAndQuality != nil {
				advanced = got.Settings.SecurityAndQuality.AdvancedSecurity
			}
			if !reflect.DeepEqual(advanced, test.advanced) {
				t.Fatalf("Advanced Security %#v; want %#v", advanced, test.advanced)
			}
			if test.name == "both" {
				*advanced.SecretProtection = false
				*got.Settings.General.Releases.EnableReleaseImmutability = true
				if !*source.Security.SecretScanning || *source.Security.ImmutableReleases {
					t.Fatal("Security export aliases source")
				}
			}
		})
	}
	source := uiExportBase()
	source.Security.ImmutableReleases = rulesetPtr(false)
	got, err := UIFromExportDeclaration(source)
	if err != nil || got.Settings == nil || got.Settings.General == nil || got.Settings.General.Releases == nil || got.Settings.General.Releases.EnableReleaseImmutability == nil || *got.Settings.General.Releases.EnableReleaseImmutability || got.Settings.General.DefaultBranch != nil || got.Settings.SecurityAndQuality != nil {
		t.Fatalf("release-only export invented General or Security controls: %#v, %v", got, err)
	}
	source.Security = settings.SecurityDeclaration{SecretScanning: rulesetPtr(false), PushProtection: rulesetPtr(true)}
	got, err = UIFromExportDeclaration(source)
	if err == nil || !reflect.DeepEqual(got, uimodel.Repository{}) {
		t.Fatalf("inconsistent Security returned partial export: %#v, %v", got, err)
	}
}

func TestUIFromExportDeclarationPages(t *testing.T) {
	source := uiExportBase()
	source.Repository.DefaultBranch = rulesetPtr("main")
	source.Pages = settings.PagesDeclaration{BuildType: rulesetPtr("legacy"), Source: &settings.PagesSource{Branch: "pages", Path: "/docs"}, CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(false)}
	want := &uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: rulesetPtr(uimodel.DeployFromABranch), Branch: rulesetPtr("pages"), SelectFolder: rulesetPtr(uimodel.PagesDocs)}, CustomDomain: &uimodel.PagesCustomDomain{}, EnforceHTTPS: rulesetPtr(false)}
	got, err := UIFromExportDeclaration(source)
	if err != nil || got.Settings == nil || !reflect.DeepEqual(got.Settings.Pages, want) || got.Settings.General == nil || got.Settings.General.DefaultBranch == nil || *got.Settings.General.DefaultBranch != "main" || got.About == nil || *got.About.Website != "https://example.com" {
		t.Fatalf("Pages export or earlier groups changed: %#v, %v", got, err)
	}
	source.Pages = settings.PagesDeclaration{}
	got, err = UIFromExportDeclaration(source)
	if err != nil || got.Settings.Pages != nil {
		t.Fatalf("omitted Pages became intent: %#v, %v", got, err)
	}
	source.Pages.BuildType = rulesetPtr("workflow")
	got, err = UIFromExportDeclaration(source)
	if err == nil || !reflect.DeepEqual(got, uimodel.Repository{}) {
		t.Fatalf("incomplete Pages returned partial export: %#v, %v", got, err)
	}
}

func TestUIFromExportDeclarationEnvironments(t *testing.T) {
	policy := settings.EnvironmentDeclaration{Name: "Production", WaitTimer: rulesetPtr(0), PreventSelfReview: rulesetPtr(false), CanAdminsBypass: rulesetPtr(true), Reviewers: rulesetPtr([]settings.EnvironmentReviewer{{Type: "User", ID: 11}, {Type: "Team", ID: 12}}), DeploymentBranchPolicyDeclared: true, DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, DeploymentBranchPatterns: rulesetPtr([]settings.EnvironmentPattern{{Type: "tag", Name: "Release/*"}, {Type: "branch", Name: "Release/*"}})}
	open := uiEnvironmentObservedPolicy("alpha")
	policies, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{policy, open})
	if err != nil {
		t.Fatal(err)
	}
	source := uiExportBase()
	source.Environments = &policies
	want := []uimodel.Environment{
		{Name: "alpha", DeploymentBranchesAndTags: rulesetPtr(uimodel.NoRestriction), DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: rulesetPtr(0), PreventSelfReview: rulesetPtr(false), AllowAdministratorsToBypassConfiguredProtectionRules: rulesetPtr(false), RequiredReviewers: rulesetPtr([]uimodel.EnvironmentReviewer{})}},
		{Name: "Production", DeploymentBranchesAndTags: rulesetPtr(uimodel.SelectedBranchesAndTags), DeploymentBranchOrTagRules: rulesetPtr([]uimodel.EnvironmentDeploymentRule{{RefType: uimodel.EnvironmentBranch, NamePattern: "Release/*"}, {RefType: uimodel.EnvironmentTag, NamePattern: "Release/*"}}), DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: rulesetPtr(0), PreventSelfReview: rulesetPtr(false), AllowAdministratorsToBypassConfiguredProtectionRules: rulesetPtr(true), RequiredReviewers: rulesetPtr([]uimodel.EnvironmentReviewer{{Team: rulesetPtr(int64(12))}, {Person: rulesetPtr(int64(11))}})}},
	}
	got, err := UIFromExportDeclaration(source)
	if err != nil || got.Settings == nil || got.Settings.Environments == nil || !reflect.DeepEqual(*got.Settings.Environments, want) {
		t.Fatalf("Environment export changed: %#v, %v", got, err)
	}
	empty, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{})
	if err != nil {
		t.Fatal(err)
	}
	for _, entries := range []*settings.EnvironmentDeclarations{nil, &empty} {
		source.Environments = entries
		got, err := UIFromExportDeclaration(source)
		present := got.Settings != nil && got.Settings.Environments != nil
		if err != nil || present != (entries != nil) || (present && len(*got.Settings.Environments) != 0) {
			t.Fatalf("Environment export presence changed: %#v, %v", got, err)
		}
	}
	partial := uiEnvironmentObservedPolicy("Zulu")
	partial.WaitTimer = nil
	policies, err = settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{open, partial})
	if err != nil {
		t.Fatal(err)
	}
	source.Environments = &policies
	got, err = UIFromExportDeclaration(source)
	if err == nil || !reflect.DeepEqual(got, uimodel.Repository{}) {
		t.Fatalf("incomplete Environment returned partial export: %#v, %v", got, err)
	}
}
