package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Reader defers each API client until the operation needs that resource.
type Reader struct {
	ObserveRepository func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error)
	Topics            func(repository.Repository) ([]string, error)
	Labels            func(repository.Repository) (settings.LabelObservation, error)
}

// Writer sends each resource independently; successful earlier writes are not rolled back.
type Writer struct {
	Repository func(repository.Repository, settings.RepositoryDeclaration) error
	Topics     func(repository.Repository, settings.TopicsDeclaration) error
	ReadLabels func(repository.Repository) (settings.LabelObservation, error)
	Labels     func(repository.Repository, settings.LabelChanges) error
}
