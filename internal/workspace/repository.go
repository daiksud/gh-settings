package workspace

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// GitRoot returns the checkout root containing the current working directory.
func GitRoot() (string, error) {
	return gitRoot("")
}

func gitRoot(directory string) (string, error) {
	command := exec.Command("git", "rev-parse", "--show-toplevel")
	command.Dir = directory
	root, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(root), "\n"), nil
}

// CurrentCommit returns the consumer checkout's resolved commit identity.
func CurrentCommit() (string, error) {
	commit, err := exec.Command("git", "rev-parse", "--verify", "HEAD^{commit}").Output()
	if err != nil {
		return "", fmt.Errorf("resolve checkout commit: %w", err)
	}
	return strings.TrimSuffix(string(commit), "\n"), nil
}

// CurrentRepository resolves the GitHub target from a checkout-scoped origin.
func CurrentRepository() (repository.Repository, error) {
	if _, err := GitRoot(); err != nil {
		return repository.Repository{}, fmt.Errorf("resolve current repository: a GitHub origin remote is required: %w", err)
	}
	// Git remote get-url applies insteadOf transport rewrites, which can hide
	// the configured GitHub repository behind a mirror URL.
	remote, err := exec.Command("git", "config", "--null", "--show-scope", "--get-all", "remote.origin.url").Output()
	if err != nil {
		return repository.Repository{}, fmt.Errorf("resolve current repository: a GitHub origin remote is required: %w", err)
	}
	// Ignore system/global/command origins, including before a local value.
	// Preserve the first checkout-scoped fetch URL and included configuration.
	entries := strings.Split(string(remote), "\x00")
	var remoteURL string
	for i := 0; i+1 < len(entries); i += 2 {
		if entries[i] == "local" || entries[i] == "worktree" {
			remoteURL = strings.TrimSpace(entries[i+1])
			break
		}
	}
	if remoteURL == "" {
		return repository.Repository{}, fmt.Errorf("resolve current repository: a checkout-scoped GitHub origin remote is required")
	}
	return parseGitHubOrigin(remoteURL)
}

func parseGitHubOrigin(remoteURL string) (repository.Repository, error) {
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
