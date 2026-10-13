package main

import (
	"fmt"
	"os/exec"
)

func setupOrdinaryOrigin(work string, env []string) error {
	if err := initGitCheckout(work, env); err != nil {
		return err
	}
	return runGit(work, env, "remote", "add", "origin", "https://github.com/owner/current.git")
}

func setupUserlessSCPOrigin(work string, env []string) error {
	if err := initGitCheckout(work, env); err != nil {
		return err
	}
	return runGit(work, env, "remote", "add", "origin", "github.com:owner/current.git")
}

func setupRewrittenOrigin(work string, env []string) error {
	if err := setupOrdinaryOrigin(work, env); err != nil {
		return err
	}
	return runGit(work, env, "config", "--local", "url.https://mirror.example/.insteadOf", "https://github.com/")
}

func setupAliasedOrigin(work string, env []string) error {
	if err := setupOrdinaryOrigin(work, env); err != nil {
		return err
	}
	if err := runGit(work, env, "remote", "set-url", "origin", "gh:owner/current.git"); err != nil {
		return err
	}
	return runGit(work, env, "config", "--local", "url.https://github.com/.insteadOf", "gh:")
}

func setupWorktreeOrigin(work string, env []string) error {
	if err := initGitCheckout(work, env); err != nil {
		return err
	}
	if err := runGit(work, env, "config", "--local", "extensions.worktreeConfig", "true"); err != nil {
		return err
	}
	return runGit(work, env, "config", "--worktree", "remote.origin.url", "https://github.com/owner/current.git")
}

func initGitCheckout(work string, env []string) error {
	return runGit(work, env, "init", "--quiet")
}

func runGit(work string, env []string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = work
	cmd.Env = env
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git: %w: %s", err, output)
	}
	return nil
}
