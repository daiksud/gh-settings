// Package uimodel contains UI-oriented repository settings.
package uimodel

type PagesSourceChoice string
type PagesFolderChoice string

const (
	GitHubActions      PagesSourceChoice = "github_actions"
	DeployFromABranch  PagesSourceChoice = "deploy_from_a_branch"
	PagesRoot          PagesFolderChoice = "root"
	PagesDocs          PagesFolderChoice = "docs"
	DefaultPagesSource                   = DeployFromABranch
	DefaultPagesFolder                   = PagesRoot
)

// PagesCustomDomain is an internal nullable scalar, not a public value submapping.
type PagesCustomDomain struct{ Value *string }

type Pages struct {
	CustomDomain       *PagesCustomDomain
	EnforceHTTPS       *bool
	BuildAndDeployment *PagesBuildAndDeployment
}
type PagesBuildAndDeployment struct {
	Source       *PagesSourceChoice
	Branch       *string
	SelectFolder *PagesFolderChoice
}
