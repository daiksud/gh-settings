package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func copyPushList(p *[]string) *[]string {
	if p == nil {
		return nil
	}
	values := append([]string{}, (*p)...)
	return &values
}
func declareUIRulesetPushRules(ui uimodel.RulesetRules) ([]settings.RulesetRule, error) {
	result := []settings.RulesetRule{}
	if rule := ui.RestrictFilePaths; rule != nil {
		if rule.RestrictedFilePaths == nil {
			return nil, fmt.Errorf("restrict_file_paths requires restricted_file_paths")
		}
		result = append(result, settings.RulesetRule{Type: "file_path_restriction", Parameters: &settings.RuleParameters{RestrictedFilePaths: copyPushList(rule.RestrictedFilePaths), IgnoredFilePaths: copyPushList(rule.AllowedExceptions)}})
	}
	if rule := ui.RestrictFilePathLength; rule != nil {
		if rule.MaximumFilePathLength == nil {
			return nil, fmt.Errorf("restrict_file_path_length requires maximum_file_path_length")
		}
		result = append(result, settings.RulesetRule{Type: "max_file_path_length", Parameters: &settings.RuleParameters{MaxFilePathLength: copyUIRulesetValue(rule.MaximumFilePathLength)}})
	}
	if rule := ui.RestrictFileExtensions; rule != nil {
		if rule.RestrictedFileExtensions == nil {
			return nil, fmt.Errorf("restrict_file_extensions requires restricted_file_extensions")
		}
		result = append(result, settings.RulesetRule{Type: "file_extension_restriction", Parameters: &settings.RuleParameters{RestrictedFileExtensions: copyPushList(rule.RestrictedFileExtensions)}})
	}
	if rule := ui.RestrictFileSize; rule != nil {
		if rule.MaximumFileSize == nil {
			return nil, fmt.Errorf("restrict_file_size requires maximum_file_size")
		}
		result = append(result, settings.RulesetRule{Type: "max_file_size", Parameters: &settings.RuleParameters{MaxFileSize: copyUIRulesetValue(rule.MaximumFileSize), IgnoredFilePaths: copyPushList(rule.AllowedExceptions)}})
	}
	return result, nil
}
func observeUIRulesetPushRule(ui *uimodel.RulesetRules, rule settings.RulesetRule) {
	p := rule.Parameters
	switch rule.Type {
	case "file_path_restriction":
		ui.RestrictFilePaths = &uimodel.FilePathsRule{RestrictedFilePaths: copyPushList(p.RestrictedFilePaths), AllowedExceptions: copyPushList(p.IgnoredFilePaths)}
	case "max_file_path_length":
		ui.RestrictFilePathLength = &uimodel.FilePathLengthRule{MaximumFilePathLength: copyUIRulesetValue(p.MaxFilePathLength)}
	case "file_extension_restriction":
		ui.RestrictFileExtensions = &uimodel.FileExtensionsRule{RestrictedFileExtensions: copyPushList(p.RestrictedFileExtensions)}
	case "max_file_size":
		ui.RestrictFileSize = &uimodel.FileSizeRule{MaximumFileSize: copyUIRulesetValue(p.MaxFileSize), AllowedExceptions: copyPushList(p.IgnoredFilePaths)}
	}
}
