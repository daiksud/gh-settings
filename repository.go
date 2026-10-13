package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func gitRoot() (string, error) {
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(root), "\n"), nil
}

func currentRepository() (repository.Repository, error) {
	if _, err := gitRoot(); err != nil {
		return repository.Repository{}, fmt.Errorf("resolve current repository: a GitHub origin remote is required: %w", err)
	}
	// Git remote get-url applies insteadOf transport rewrites, which can hide
	// the configured GitHub repository behind a mirror URL.
	remote, err := exec.Command("git", "config", "--get-all", "remote.origin.url").Output()
	if err != nil {
		return repository.Repository{}, fmt.Errorf("resolve current repository: a GitHub origin remote is required: %w", err)
	}
	// Match git remote get-url's first fetch URL when origin has multiple URLs.
	firstURL, _, _ := strings.Cut(string(remote), "\n")
	remoteURL := strings.TrimSpace(firstURL)
	if repo, err := parseGitHubOrigin(remoteURL); err == nil {
		return repo, nil
	}
	// Aliases may rewrite in the other direction, from a short local name to
	// a GitHub URL. Preserve that previously supported form as a fallback.
	rewritten, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return repository.Repository{}, fmt.Errorf("resolve current repository: a GitHub origin remote is required: %w", err)
	}
	return parseGitHubOrigin(strings.TrimSpace(string(rewritten)))
}

func parseGitHubOrigin(remoteURL string) (repository.Repository, error) {
	// go-gh recognizes full SSH URLs but not SCP syntax without a username.
	if strings.HasPrefix(remoteURL, "github.com:") {
		remoteURL = "ssh://github.com/" + strings.TrimPrefix(remoteURL, "github.com:")
	}
	if !strings.HasPrefix(remoteURL, "https://") && !strings.HasPrefix(remoteURL, "ssh://") && !strings.HasPrefix(remoteURL, "git@github.com:") {
		return repository.Repository{}, fmt.Errorf("origin must be a GitHub HTTPS or SSH URL")
	}
	repo, err := repository.Parse(remoteURL)
	if err != nil {
		return repository.Repository{}, fmt.Errorf("parse origin remote: %w", err)
	}
	if repo.Host != "github.com" {
		return repository.Repository{}, fmt.Errorf("origin must point to github.com, got %q", repo.Host)
	}
	return repo, nil
}
