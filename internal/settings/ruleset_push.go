package settings

import (
	"fmt"
	"slices"
	"unicode/utf8"
)

func isPushRule(kind string) bool {
	return slices.Contains([]string{"file_path_restriction", "max_file_path_length", "file_extension_restriction", "max_file_size"}, kind)
}
func (p *RuleParameters) hasPush() bool {
	return p != nil && (p.RestrictedFilePaths != nil || p.IgnoredFilePaths != nil || p.RestrictedFileExtensions != nil || p.MaxFilePathLength != nil || p.MaxFileSize != nil)
}
func validatePushRule(r RulesetRule) error {
	p := r.Parameters
	if p == nil || p.hasPR() || p.hasChecks() || p.hasAnalysis() {
		return fmt.Errorf("%s requires its own complete parameters", r.Type)
	}
	if (p.RestrictedFilePaths != nil && r.Type != "file_path_restriction") || (p.IgnoredFilePaths != nil && r.Type != "file_path_restriction" && r.Type != "max_file_size") || (p.RestrictedFileExtensions != nil && r.Type != "file_extension_restriction") || (p.MaxFilePathLength != nil && r.Type != "max_file_path_length") || (p.MaxFileSize != nil && r.Type != "max_file_size") {
		return fmt.Errorf("parameters do not belong to %s", r.Type)
	}
	switch r.Type {
	case "file_path_restriction":
		if p.RestrictedFilePaths == nil || p.IgnoredFilePaths == nil {
			return fmt.Errorf("file_path_restriction requires restricted_file_paths and ignored_file_paths")
		}
	case "file_extension_restriction":
		if p.RestrictedFileExtensions == nil {
			return fmt.Errorf("file_extension_restriction requires restricted_file_extensions")
		}
	case "max_file_path_length":
		if p.MaxFilePathLength == nil || *p.MaxFilePathLength < 1 || *p.MaxFilePathLength > 32767 {
			return fmt.Errorf("max_file_path_length requires a value from 1 to 32767")
		}
	case "max_file_size":
		if p.IgnoredFilePaths == nil {
			return fmt.Errorf("max_file_size requires ignored_file_paths")
		}
		if p.MaxFileSize == nil || *p.MaxFileSize < 1 || *p.MaxFileSize > 100 {
			return fmt.Errorf("max_file_size requires a value from 1 to 100 MB")
		}
	}
	for _, field := range []struct {
		list    *[]string
		limited bool
	}{{p.RestrictedFilePaths, true}, {p.IgnoredFilePaths, false}, {p.RestrictedFileExtensions, true}} {
		list := field.list
		if list == nil {
			continue
		}
		if field.limited && len(*list) > 200 {
			return fmt.Errorf("push restriction has more than 200 entries")
		}
		seen := map[string]bool{}
		for _, value := range *list {
			if !validRulesetText(value) || (field.limited && utf8.RuneCountInString(value) > 200) || seen[value] {
				return fmt.Errorf("invalid or duplicate push restriction entry")
			}
			seen[value] = true
		}
	}
	return nil
}
