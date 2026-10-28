// Package uimodel prepares UI-oriented values without activating public YAML.
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
	DeploymentBranchesAndTags  *EnvironmentDeploymentChoice
	DeploymentBranchOrTagRules *[]EnvironmentDeploymentRule
	Name                       string
	DeploymentProtectionRules  *EnvironmentDeploymentProtectionRules
}
type EnvironmentDeploymentProtectionRules struct {
	// Zero represents the unchecked Wait timer group; positive values are minutes.
	WaitTimer                                            *int
	RequiredReviewers                                    *[]EnvironmentReviewer
	PreventSelfReview                                    *bool
	AllowAdministratorsToBypassConfiguredProtectionRules *bool
}

// The UI selects people or teams; transport principal types remain in adapters.
type EnvironmentReviewer struct {
	Person *int64
	Team   *int64
}
