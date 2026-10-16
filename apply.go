package main

import (
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func applyCurrent() ([]byte, error) {
	desired, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	if desired == (settings.RepositoryDeclaration{}) {
		return []byte("No managed repository settings.\n"), nil
	}
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return nil, err
	}
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return nil, err
	}
	if err := client.ApplyRepository(repo, desired); err != nil {
		return nil, err
	}
	return []byte("Applied repository settings.\n"), nil
}
