package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportUsesOrigin(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "--quiet", dir}, {"-C", dir, "remote", "add", "origin", "https://github.com/owner/current.git"}} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	t.Chdir(dir)
	t.Setenv("GH_REPO", "other/repository")
	repo, err := currentRepository()
	if err != nil {
		t.Fatal(err)
	}
	if repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
		t.Fatalf("selected %+v", repo)
	}
}

func TestExportUsesConfiguredOriginBeforeInsteadOf(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet", dir},
		{"-C", dir, "remote", "add", "origin", "https://github.com/owner/current.git"},
		{"-C", dir, "remote", "set-url", "--add", "origin", "https://github.com/owner/other.git"},
		{"-C", dir, "config", "--local", "url.https://mirror.example/.insteadOf", "https://github.com/"},
	} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	t.Chdir(dir)
	if output, err := exec.Command("git", "remote", "get-url", "origin").Output(); err != nil || string(output) != "https://mirror.example/owner/current.git\n" {
		t.Fatalf("rewritten origin=%q err=%v", output, err)
	}
	repo, err := currentRepository()
	if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
		t.Fatalf("selected %+v, err=%v", repo, err)
	}
}

func TestExportUsesAliasRewrittenToGitHub(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet", dir},
		{"-C", dir, "remote", "add", "origin", "gh:owner/current.git"},
		{"-C", dir, "config", "--local", "url.https://github.com/.insteadOf", "gh:"},
	} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	t.Chdir(dir)
	if output, err := exec.Command("git", "remote", "get-url", "origin").Output(); err != nil || string(output) != "https://github.com/owner/current.git\n" {
		t.Fatalf("rewritten origin=%q err=%v", output, err)
	}
	repo, err := currentRepository()
	if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
		t.Fatalf("selected %+v, err=%v", repo, err)
	}
}

func TestExportRejectsGlobalOriginOutsideCheckout(t *testing.T) {
	config := filepath.Join(t.TempDir(), "global-gitconfig")
	if err := os.WriteFile(config, []byte("[remote \"origin\"]\n\turl = https://github.com/owner/current.git\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Chdir(t.TempDir())
	if repo, err := currentRepository(); err == nil {
		t.Fatalf("selected global origin outside checkout: %+v", repo)
	}
}

func TestExportUsesRepositoryConfigSources(t *testing.T) {
	for _, source := range []string{"include", "worktree"} {
		t.Run(source, func(t *testing.T) {
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			dir := t.TempDir()
			if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
				t.Fatalf("git init: %v: %s", err, output)
			}
			var commands [][]string
			if source == "include" {
				included := filepath.Join(dir, "included.conf")
				if err := os.WriteFile(included, []byte("[remote \"origin\"]\n\turl = https://github.com/owner/current.git\n"), 0600); err != nil {
					t.Fatal(err)
				}
				commands = [][]string{{"config", "--local", "include.path", included}}
			} else {
				commands = [][]string{
					{"config", "--local", "extensions.worktreeConfig", "true"},
					{"config", "--worktree", "remote.origin.url", "https://github.com/owner/current.git"},
				}
			}
			for _, args := range commands {
				cmd := exec.Command("git", args...)
				cmd.Dir = dir
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %v: %v: %s", args, err, output)
				}
			}
			t.Chdir(dir)
			if output, err := exec.Command("git", "remote", "get-url", "origin").Output(); err != nil || string(output) != "https://github.com/owner/current.git\n" {
				t.Fatalf("origin=%q err=%v", output, err)
			}
			repo, err := currentRepository()
			if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
				t.Fatalf("selected %+v, err=%v", repo, err)
			}
		})
	}
}

func TestExportRepositoryFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	code := run([]string{"export"}, &stdout, &stderr, commandOperations{export: exportCurrent})
	if code == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRepositoryRemotes(t *testing.T) {
	for _, remote := range []string{"git@github.com:owner/current.git", "github.com:owner/current.git", "ssh://git@github.com/owner/current.git", "../owner/current", "owner/current", "https://gitlab.com/owner/current.git"} {
		t.Run(remote, func(t *testing.T) {
			dir := t.TempDir()
			for _, args := range [][]string{{"init", "--quiet", dir}, {"-C", dir, "remote", "add", "origin", remote}} {
				if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
					t.Fatalf("git: %v: %s", err, out)
				}
			}
			subdir := filepath.Join(dir, "subdirectory")
			if err := os.Mkdir(subdir, 0700); err != nil {
				t.Fatal(err)
			}
			t.Chdir(subdir)
			repo, err := currentRepository()
			if strings.Contains(remote, "github.com") {
				if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
					t.Fatalf("repo=%+v err=%v", repo, err)
				}
			} else if err == nil {
				t.Fatalf("non-GitHub remote selected: %+v", repo)
			}
		})
	}
}
