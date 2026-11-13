package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readCollaborators(repo repository.Repository) (settings.CollaboratorObservation, error) {
	client, err := github.NewCollaboratorsClient(repo.Host)
	if err != nil {
		return settings.CollaboratorObservation{}, err
	}
	return client.ReadCollaborators(repo)
}
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
