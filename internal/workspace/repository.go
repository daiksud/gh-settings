package workspace

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// GitRoot returns the checkout root containing the current working directory.
func GitRoot() (string, error) {
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(root), "\n"), nil
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
	if repo, err := parseGitHubOrigin(remoteURL); err == nil {
		return repo, nil
	}
	// Aliases may rewrite in the other direction, from a short local name to
	// a GitHub URL. Preserve that previously supported form as a fallback.
	rewritten, err := rewriteGitURL(remoteURL)
	if err != nil {
		return repository.Repository{}, fmt.Errorf("resolve current repository: %w", err)
	}
	return parseGitHubOrigin(rewritten)
}

// Apply insteadOf to the selected URL itself. Resolving the remote name again
// would let an earlier global remote.origin.url replace the checkout's alias.
func rewriteGitURL(remoteURL string) (string, error) {
	rules, err := exec.Command("git", "config", "--null", "--get-regexp", "^url\\..*\\.insteadof$").Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return remoteURL, nil // No rewrite rules.
		}
		return "", fmt.Errorf("read Git URL rewrite rules: %w", err)
	}
	rewritten, longest := remoteURL, -1
	for _, entry := range strings.Split(string(rules), "\x00") {
		key, prefix, ok := strings.Cut(entry, "\n")
		if !ok || !strings.HasPrefix(remoteURL, prefix) || len(prefix) <= longest {
			continue
		}
		base := strings.TrimSuffix(strings.TrimPrefix(key, "url."), ".insteadof")
		rewritten, longest = base+strings.TrimPrefix(remoteURL, prefix), len(prefix)
	}
	return rewritten, nil
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
