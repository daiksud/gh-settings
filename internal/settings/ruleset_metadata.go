package settings

import "slices"

// IsMetadataPatternRule identifies the five repository metadata restrictions.
func IsMetadataPatternRule(kind string) bool {
	return slices.Contains([]string{"commit_message_pattern", "commit_author_email_pattern", "committer_email_pattern", "branch_name_pattern", "tag_name_pattern"}, kind)
}
func (p *RuleParameters) hasMetadata() bool {
	return p != nil && (p.Name != nil || p.Negate != nil || p.Operator != nil || p.Pattern != nil)
}
