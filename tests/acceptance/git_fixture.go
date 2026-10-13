package main

import (
	"fmt"
	"os/exec"
)

func setupGitCheckout(work string, env []string, test scenario) error {
	if test.noRepo {
		return nil
	}
	if err := initGitCheckout(work, env); err != nil {
		return err
	}
	var gitCommands [][]string
	if !test.worktreeOrigin {
		origin := "https://github.com/owner/current.git"
		if test.userlessOrigin {
			origin = "github.com:owner/current.git"
		}
		gitCommands = append(gitCommands, []string{"remote", "add", "origin", origin})
	} else {
		gitCommands = append(gitCommands,
			[]string{"config", "--local", "extensions.worktreeConfig", "true"},
			[]string{"config", "--worktree", "remote.origin.url", "https://github.com/owner/current.git"})
	}
	for _, args := range gitCommands {
		cmd := exec.Command("git", args...)
		cmd.Dir = work
		cmd.Env = env
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git: %w: %s", err, output)
		}
	}
	if test.rewriteOrigin {
		cmd := exec.Command("git", "config", "--local", "url.https://mirror.example/.insteadOf", "https://github.com/")
		cmd.Dir = work
		cmd.Env = env
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git rewrite: %w: %s", err, output)
		}
	}
	if test.aliasOrigin {
		for _, args := range [][]string{
			{"remote", "set-url", "origin", "gh:owner/current.git"},
			{"config", "--local", "url.https://github.com/.insteadOf", "gh:"},
		} {
			cmd := exec.Command("git", args...)
			cmd.Dir = work
			cmd.Env = env
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("git alias: %w: %s", err, output)
			}
		}
	}
	return nil
}

func initGitCheckout(work string, env []string) error {
	cmd := exec.Command("git", "init", "--quiet")
	cmd.Dir = work
	cmd.Env = env
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git: %w: %s", err, output)
	}
	return nil
}
