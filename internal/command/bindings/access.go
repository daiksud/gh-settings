package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func resolveCollaborator(repo repository.Repository, person settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error) {
	client, err := github.NewCollaboratorsClient(repo.Host)
	if err != nil {
		return settings.CollaboratorIdentity{}, err
	}
	return client.ResolveCollaborator(person)
}

func reconcileCollaborator(repo repository.Repository, person settings.CollaboratorSelection) (settings.CollaboratorWriteResult, error) {
	client, err := github.NewCollaboratorsClient(repo.Host)
	if err != nil {
		return settings.CollaboratorWriteResult{Person: person.Person}, err
	}
	return client.ReconcileCollaborator(repo, person)
}

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

func createDeployKey(repo repository.Repository, expected settings.DeployKeyObservation, desired settings.DeployKeyDeclaration) (settings.DeployKeyWriteResult, error) {
	client, err := github.NewDeployKeysClient(repo.Host)
	if err != nil {
		return settings.DeployKeyWriteResult{}, err
	}
	return client.CreateDeployKey(repo, expected, desired)
}

func deleteDeployKey(repo repository.Repository, expected settings.DeployKeyObservation, id int64) (settings.DeployKeyObservation, error) {
	client, err := github.NewDeployKeysClient(repo.Host)
	if err != nil {
		return settings.DeployKeyObservation{}, err
	}
	return client.DeleteDeployKey(repo, expected, id)
}
