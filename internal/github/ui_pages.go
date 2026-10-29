package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func PagesDeclaration(ui uimodel.Pages) (settings.PagesDeclaration, error) {
	var result settings.PagesDeclaration
	if b := ui.BuildAndDeployment; b != nil {
		if b.Source != nil {
			var mode string
			switch *b.Source {
			case uimodel.GitHubActions:
				mode = "workflow"
			case uimodel.DeployFromABranch:
				mode = "legacy"
			default:
				return settings.PagesDeclaration{}, fmt.Errorf("unsupported UI Pages source choice")
			}
			result.BuildType = &mode
		}
		if (b.Branch == nil) != (b.SelectFolder == nil) {
			return settings.PagesDeclaration{}, fmt.Errorf("Pages Branch and Select folder must be declared together")
		}
		if b.Branch != nil {
			var path string
			switch *b.SelectFolder {
			case uimodel.PagesRoot:
				path = "/"
			case uimodel.PagesDocs:
				path = "/docs"
			default:
				return settings.PagesDeclaration{}, fmt.Errorf("unsupported UI Pages folder choice")
			}
			result.Source = &settings.PagesSource{Branch: *b.Branch, Path: path}
		}
	}
	if ui.CustomDomain != nil {
		result.CNAME = &settings.PagesDomain{Value: copyUIPagesValue(ui.CustomDomain.Value)}
	}
	result.HTTPSEnforced = copyUIPagesValue(ui.EnforceHTTPS)
	if err := result.Validate(); err != nil {
		return settings.PagesDeclaration{}, err
	}
	return result, nil
}

func copyUIPagesValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

// PreparedPages keeps unowned site visibility outside desired UI controls.
type PreparedPages struct {
	Pages  uimodel.Pages
	Public bool
}

// PagesPolicyUI projects complete exported policy without site visibility metadata.
func PagesPolicyUI(policy settings.PagesDeclaration) (uimodel.Pages, error) {
	if policy.BuildType == nil || policy.CNAME == nil || policy.HTTPSEnforced == nil {
		return uimodel.Pages{}, fmt.Errorf("Pages policy requires complete build type, custom domain and HTTPS values")
	}
	if err := policy.Validate(); err != nil {
		return uimodel.Pages{}, err
	}
	var source uimodel.PagesSourceChoice
	switch *policy.BuildType {
	case "legacy":
		source = uimodel.DeployFromABranch
	case "workflow":
		source = uimodel.GitHubActions
	default:
		return uimodel.Pages{}, fmt.Errorf("Pages build type is outside UI choices")
	}
	b := &uimodel.PagesBuildAndDeployment{Source: &source}
	if *policy.BuildType == "legacy" {
		if policy.Source == nil {
			return uimodel.Pages{}, fmt.Errorf("Pages branch source was not acquired")
		}
		var folder uimodel.PagesFolderChoice
		switch policy.Source.Path {
		case "/":
			folder = uimodel.PagesRoot
		case "/docs":
			folder = uimodel.PagesDocs
		default:
			return uimodel.Pages{}, fmt.Errorf("Pages folder is outside UI choices")
		}
		b.Branch = copyUIPagesValue(&policy.Source.Branch)
		b.SelectFolder = &folder
	}
	return uimodel.Pages{BuildAndDeployment: b, CustomDomain: &uimodel.PagesCustomDomain{Value: copyUIPagesValue(policy.CNAME.Value)}, EnforceHTTPS: copyUIPagesValue(policy.HTTPSEnforced)}, nil

}

func PagesObservation(observation settings.PagesObservation) (PreparedPages, error) {
	state, err := observation.Values()
	if err != nil {
		return PreparedPages{}, err
	}
	ui, err := PagesPolicyUI(settings.PagesDeclaration{BuildType: &state.BuildType, Source: state.Source, CNAME: &settings.PagesDomain{Value: state.CNAME}, HTTPSEnforced: &state.HTTPSEnforced})
	if err != nil {
		return PreparedPages{}, err
	}
	return PreparedPages{Pages: ui, Public: state.Public}, nil
}
