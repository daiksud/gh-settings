package settings

import (
	"strings"
	"unicode/utf8"
)

// ValidDefaultBranch accepts literal branch names without resolving revisions or
// normalizing names. GitHub decides whether the branch exists during PATCH.
func ValidDefaultBranch(name string) bool {
	if name == "" || name == "HEAD" || !utf8.ValidString(name) || strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".") || strings.Contains(name, "..") || strings.Contains(name, "@{") {
		return false
	}
	for _, component := range strings.Split(name, "/") {
		if component == "" || strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".lock") {
			return false
		}
	}
	for _, value := range []byte(name) {
		if value <= ' ' || value == 127 || strings.ContainsRune("~^:?*[\\", rune(value)) {
			return false
		}
	}
	return true
}
