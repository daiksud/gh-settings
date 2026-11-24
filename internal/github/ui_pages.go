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
			mode, valid := settings.PagesModeValueForChoice(*b.Source)
			if !valid {
				return settings.PagesDeclaration{}, fmt.Errorf("unsupported UI Pages source choice")
			}
			result.BuildType = &mode
		}
		if b.Branch != nil && *b.Branch == "none" && b.SelectFolder == nil {
			result.Source = &settings.PagesSource{Branch: "none"}
		} else {
			if (b.Branch == nil) != (b.SelectFolder == nil) {
				return settings.PagesDeclaration{}, fmt.Errorf("Pages Branch and Select folder must be declared together")
			}
			if b.Branch != nil {
				path, valid := settings.PagesFolderValueForChoice(*b.SelectFolder)
				if !valid {
					return settings.PagesDeclaration{}, fmt.Errorf("unsupported UI Pages folder choice")
				}
				result.Source = &settings.PagesSource{Branch: *b.Branch, Path: path}
			}
		}
	}
	if ui.CustomDomain != nil {
		result.CNAME = &settings.PagesDomain{Value: copyUIValue(ui.CustomDomain.Value)}
	}
	result.HTTPSEnforced = copyUIValue(ui.EnforceHTTPS)
	if err := result.Validate(); err != nil {
		return settings.PagesDeclaration{}, err
	}
	return result, nil
}
