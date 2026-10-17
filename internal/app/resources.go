package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Reader defers each API client until the operation needs that resource.
type Reader struct {
	Repository func(repository.Repository) (settings.RepositoryState, error)
	Topics     func(repository.Repository) ([]string, error)
}

// Writer sends each resource independently; successful earlier writes are not rolled back.
type Writer struct {
	Repository func(repository.Repository, settings.RepositoryDeclaration) error
	Topics     func(repository.Repository, settings.TopicsDeclaration) error
}
