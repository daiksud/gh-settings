package settings

import (
	"slices"
)

func isPushRule(kind string) bool {
	return slices.Contains([]string{"file_path_restriction", "max_file_path_length", "file_extension_restriction", "max_file_size"}, kind)
}
func (p *RuleParameters) hasPush() bool {
	return p != nil && (p.RestrictedFilePaths != nil || p.IgnoredFilePaths != nil || p.RestrictedFileExtensions != nil || p.MaxFilePathLength != nil || p.MaxFileSize != nil)
}
