// Package uimodel contains UI-oriented repository settings.
package uimodel

type EnvironmentDeploymentChoice string
type EnvironmentRefType string

const (
	NoRestriction                      EnvironmentDeploymentChoice = "no_restriction"
	ProtectedBranchesOnly              EnvironmentDeploymentChoice = "protected_branches_only"
	SelectedBranchesAndTags            EnvironmentDeploymentChoice = "selected_branches_and_tags"
	EnvironmentBranch                  EnvironmentRefType          = "branch"
	EnvironmentTag                     EnvironmentRefType          = "tag"
	DefaultEnvironmentDeploymentChoice                             = NoRestriction
	DefaultEnvironmentRefType                                      = EnvironmentBranch
)

type EnvironmentDeploymentRule struct {
	RefType     EnvironmentRefType
	NamePattern string
}

type Environment struct {
	Name                       string
	DeploymentProtectionRules  *EnvironmentDeploymentProtectionRules
	DeploymentBranchesAndTags  *EnvironmentDeploymentChoice
	DeploymentBranchOrTagRules *[]EnvironmentDeploymentRule
	Variables                  *[]RepositoryVariable
}
type EnvironmentDeploymentProtectionRules struct {
	RequiredReviewers *EnvironmentRequiredReviewers
	// Zero represents the unchecked Wait timer group; positive values are minutes.
	WaitTimer                                            *int
	EnableCustomRulesWithGitHubApps                      *[]int64
	AllowAdministratorsToBypassConfiguredProtectionRules *bool
}

type EnvironmentRequiredReviewers struct {
	Reviewers         *[]EnvironmentReviewer
	PreventSelfReview *bool
}

// The UI selects people or teams; transport principal types remain in adapters.
type EnvironmentReviewer struct {
	Person *int64
	Team   *int64
}
