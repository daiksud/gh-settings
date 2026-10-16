package workspace

import (
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
	repo, err := CurrentRepository()
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
	repo, err := CurrentRepository()
	if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
		t.Fatalf("selected %+v, err=%v", repo, err)
	}
}

func TestExportRejectsAliasRewrittenToGitHub(t *testing.T) {
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
	repo, err := CurrentRepository()
	if err == nil || repo.Host != "" || repo.Owner != "" || repo.Name != "" {
		t.Fatalf("rewrite-only alias selected %+v, err=%v", repo, err)
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
	if repo, err := CurrentRepository(); err == nil {
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
			repo, err := CurrentRepository()
			if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
				t.Fatalf("selected %+v, err=%v", repo, err)
			}
		})
	}
}

func TestRepositoryRemotes(t *testing.T) {
	for _, test := range []struct {
		remote string
		valid  bool
	}{
		{"https://github.com/owner/current.git", true},
		{"git@github.com:owner/current.git", true},
		{"ssh://git@github.com/owner/current.git", true},
		{"github.com:owner/current.git", false},
		{"../owner/current", false},
		{"owner/current", false},
		{"https://gitlab.com/owner/current.git", false},
	} {
		t.Run(test.remote, func(t *testing.T) {
			dir := t.TempDir()
			for _, args := range [][]string{{"init", "--quiet", dir}, {"-C", dir, "remote", "add", "origin", test.remote}} {
				if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
					t.Fatalf("git: %v: %s", err, out)
				}
			}
			subdir := filepath.Join(dir, "subdirectory")
			if err := os.Mkdir(subdir, 0700); err != nil {
				t.Fatal(err)
			}
			t.Chdir(subdir)
			repo, err := CurrentRepository()
			if test.valid {
				if err != nil || repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "current" {
					t.Fatalf("repo=%+v err=%v", repo, err)
				}
			} else if err == nil || repo.Host != "" || repo.Owner != "" || repo.Name != "" {
				t.Fatalf("unsupported remote selected: %+v, err=%v", repo, err)
			}
		})
	}
}

func TestRepositoryIgnoresUnscopedOrigins(t *testing.T) {
	for _, source := range []string{"global", "system", "command"} {
		for _, checkout := range []string{"missing", "local", "include", "worktree", "alias"} {
			t.Run(source+"/"+checkout, func(t *testing.T) {
				dir := t.TempDir()
				config := filepath.Join(t.TempDir(), "unscoped.conf")
				if err := os.WriteFile(config, []byte("[remote \"origin\"]\n\turl = https://github.com/unrelated/wrong.git\n[url \"https://github.com/owner/\"]\n\tinsteadOf = gh:\n[url \"https://github.com/owner/current.git\"]\n\tinsteadOf = gh:current\n"), 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
				t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
				t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
				t.Setenv("GIT_CONFIG_COUNT", "0")
				if source == "global" {
					t.Setenv("GIT_CONFIG_GLOBAL", config)
				}
				if source == "system" {
					t.Setenv("GIT_CONFIG_SYSTEM", config)
				}
				if source == "command" {
					t.Setenv("GIT_CONFIG_COUNT", "1")
					t.Setenv("GIT_CONFIG_KEY_0", "remote.origin.url")
					t.Setenv("GIT_CONFIG_VALUE_0", "https://github.com/unrelated/wrong.git")
				}
				if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
					t.Fatalf("git init: %v: %s", err, output)
				}
				t.Chdir(dir)
				var commands [][]string
				switch checkout {
				case "local":
					commands = [][]string{{"config", "--local", "remote.origin.url", "https://github.com/owner/current.git"}}
				case "include":
					included := filepath.Join(dir, "included.conf")
					if err := os.WriteFile(included, []byte("[remote \"origin\"]\n\turl = https://github.com/owner/current.git\n"), 0600); err != nil {
						t.Fatal(err)
					}
					commands = [][]string{{"config", "--local", "include.path", included}}
				case "worktree":
					commands = [][]string{{"config", "--local", "extensions.worktreeConfig", "true"}, {"config", "--worktree", "remote.origin.url", "https://github.com/owner/current.git"}}
				case "alias":
					commands = [][]string{{"config", "--local", "remote.origin.url", "gh:current"}, {"config", "--local", "url.https://github.com/owner/current.git.insteadOf", "gh:current"}}
				}
				for _, args := range commands {
					if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
						t.Fatalf("git %v: %v: %s", args, err, output)
					}
				}
				repo, err := CurrentRepository()
				if checkout == "missing" {
					if err == nil || !strings.Contains(err.Error(), "checkout-scoped") {
						t.Fatalf("repo=%+v err=%v", repo, err)
					}
				} else if checkout == "alias" {
					if err == nil || repo.Host != "" || repo.Owner != "" || repo.Name != "" {
						t.Fatalf("rewrite-only alias selected %+v, err=%v", repo, err)
					}
				} else if err != nil || repo.Owner != "owner" || repo.Name != "current" {
					t.Fatalf("repo=%+v err=%v", repo, err)
				}
			})
		}
	}
}
