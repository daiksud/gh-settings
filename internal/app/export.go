package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// Export saves all observed fields only after a complete read and serialization.
func Export(readRepository func(repository.Repository) (settings.RepositoryState, error)) error {
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return err
	}
	root, err := workspace.GitRoot()
	if err != nil {
		return fmt.Errorf("locate %s: %w", workspace.SettingsPath, err)
	}
	current, err := readRepository(repo)
	if err != nil {
		return err
	}
	declaration := settings.RepositoryDeclaration{
		Description: &current.Description, Homepage: &current.Homepage,
		HasIssues: &current.HasIssues, HasProjects: &current.HasProjects, HasWiki: &current.HasWiki,
	}
	data, err := workspace.EncodeConfig(declaration)
	if err != nil {
		return err
	}
	return workspace.WriteConfig(root, data)
}
