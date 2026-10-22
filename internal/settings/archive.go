package settings

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// ArchiveState is one complete observation of the selected repository's identity and lifecycle state.
// The checked ID is kept across the transition; a name-addressed API is not an atomic ID lock.
type ArchiveState struct {
	ID       int64
	FullName string
	Archived bool
}

func (s ArchiveState) Validate(repo repository.Repository) error {
	if s.ID <= 0 || s.FullName == "" || !strings.EqualFold(s.FullName, repo.Owner+"/"+repo.Name) {
		return fmt.Errorf("repository archive identity does not match checkout target")
	}
	return nil
}

var archiveBoolFields = [...]repositoryField[ArchiveState, bool]{
	{name: "repository.archived", current: func(s ArchiveState) bool { return s.Archived },
		desired:    func(d RepositoryDeclaration) *bool { return d.Archived },
		setDesired: func(d *RepositoryDeclaration, v bool) { d.Archived = &v }},
}
