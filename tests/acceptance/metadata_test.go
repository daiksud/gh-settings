//go:build acceptance

package acceptance

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionMetadata(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture("{}", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(fixtureEnv(fixture.cfg), "GH_REPO=other/repository")
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	if err := runGit(fixture.work, env, "-c", "commit.gpgsign=false", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--quiet", "--allow-empty", "-m", "test: consumer identity"); err != nil {
		t.Fatal(err)
	}
	git := exec.Command("git", "rev-parse", "HEAD")
	git.Dir, git.Env = fixture.work, env
	commitBytes, err := git.Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(commitBytes))
	nested := filepath.Join(fixture.work, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	identity, err := executeCLI(binary, nested, fixture.root, env, []string{"--version"}, false)
	if err != nil || identity.runErr != nil {
		t.Fatalf("version: %v result=%+v", err, identity)
	}
	version := strings.TrimPrefix(strings.TrimSpace(identity.stdout.String()), "gh-settings ")
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	result, err := executeCLI(binary, nested, fixture.root, env, []string{"action", "metadata"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.runErr != nil || result.stderr.Len() != 0 {
		t.Fatalf("metadata: %v stderr=%q", result.runErr, result.stderr.String())
	}
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSuffix(result.stdout.String(), "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if _, duplicate := values[key]; !ok || duplicate {
			t.Fatalf("invalid metadata record %q", line)
		}
		values[key] = value
	}
	if len(values) != 4 || values["config-sha"] != commit || values["target"] != "owner/current" || values["cli-version"] != version || values["cli-sha256"] != digest {
		t.Fatalf("metadata=%v", values)
	}
	if err := fixture.checkRequests(true); err != nil {
		t.Fatal(err)
	}
}

func TestActionMetadataRejectsIndirectOrigin(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name  string
		setup func(string, []string) error
	}{
		{"rewrite-only-alias", setupAliasedOrigin},
		{"username-less-scp", setupUserlessSCPOrigin},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, err := startAPIFixture("{}", 0)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := append(fixtureEnv(fixture.cfg), "GH_REPO=other/repository")
			if err := test.setup(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			if err := runGit(fixture.work, env, "-c", "commit.gpgsign=false", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--quiet", "--allow-empty", "-m", "test: consumer identity"); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "metadata"}, false)
			if err != nil {
				t.Fatal(err)
			}
			exit, ok := result.runErr.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "origin") {
				t.Fatalf("metadata: %v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
			}
			if err := fixture.checkRequests(true); err != nil {
				t.Fatal(err)
			}
		})
	}
}
