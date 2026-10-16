// Package actions adapts the verified executable to same-version Actions wiring.
package actions

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/daiksud/gh-settings/internal/workspace"
)

// Run writes nonsecret machine output and maps failures to diagnostics and status.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	if err := execute(args, stdout, version); err != nil {
		fmt.Fprintf(stderr, "gh settings action: %v\n", err)
		return 1
	}
	return 0
}

func execute(args []string, stdout io.Writer, version string) error {
	if len(args) != 1 || args[0] != "metadata" {
		return fmt.Errorf("usage: gh settings action metadata")
	}
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return err
	}
	commit, err := workspace.CurrentCommit()
	if err != nil {
		return err
	}
	for _, value := range []string{commit, repo.Owner, repo.Name, version} {
		if value == "" || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("metadata values must be nonempty single-line values")
		}
	}
	// Keep Linux image identity even when the launch pathname is replaced.
	executable := "/proc/self/exe"
	if runtime.GOOS != "linux" {
		executable, err = os.Executable()
		if err != nil {
			return fmt.Errorf("locate executing binary: %w", err)
		}
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		return fmt.Errorf("hash executing binary: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "config-sha=%s\ntarget=%s/%s\ncli-version=%s\ncli-sha256=%x\n", commit, repo.Owner, repo.Name, version, sha256.Sum256(data)); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}
