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

func PagesObservation(observation settings.PagesObservation) (PreparedPages, error) {
	state, err := observation.Values()
	if err != nil {
		return PreparedPages{}, err
	}
	var source uimodel.PagesSourceChoice
	switch state.BuildType {
	case "legacy":
		source = uimodel.DeployFromABranch
	case "workflow":
		source = uimodel.GitHubActions
	default:
		return PreparedPages{}, fmt.Errorf("Pages build type is outside UI choices")
	}
	b := &uimodel.PagesBuildAndDeployment{Source: &source}
	if state.BuildType == "legacy" {
		if state.Source == nil {
			return PreparedPages{}, fmt.Errorf("Pages branch source was not acquired")
		}
		var folder uimodel.PagesFolderChoice
		switch state.Source.Path {
		case "/":
			folder = uimodel.PagesRoot
		case "/docs":
			folder = uimodel.PagesDocs
		default:
			return PreparedPages{}, fmt.Errorf("Pages folder is outside UI choices")
		}
		b.Branch = copyUIPagesValue(&state.Source.Branch)
		b.SelectFolder = &folder
	}
	return PreparedPages{Pages: uimodel.Pages{BuildAndDeployment: b, CustomDomain: &uimodel.PagesCustomDomain{Value: copyUIPagesValue(state.CNAME)}, EnforceHTTPS: copyUIPagesValue(&state.HTTPSEnforced)}, Public: state.Public}, nil
}
