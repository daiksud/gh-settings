package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func refreshCollaboratorPrune(repo repository.Repository, current settings.CollaboratorPruneObservation, reconciled *settings.CollaboratorWriteResult) (settings.CollaboratorPruneObservation, error) {
	client, err := github.NewCollaboratorsClient(repo.Host)
	if err != nil {
		return settings.CollaboratorPruneObservation{}, err
	}
	return client.RefreshCollaboratorPrune(repo, current, reconciled)
}
func deleteCollaborator(repo repository.Repository, current settings.CollaboratorPruneObservation, deletion settings.CollaboratorDeletion) (settings.CollaboratorPruneObservation, settings.CollaboratorDeleteResult, error) {
	client, err := github.NewCollaboratorsClient(repo.Host)
	if err != nil {
		return current, settings.CollaboratorDeleteResult{Deletion: deletion}, err
	}
	return client.DeleteCollaborator(repo, current, deletion)
}
