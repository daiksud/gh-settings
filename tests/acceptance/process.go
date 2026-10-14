package acceptance

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type executionResult struct {
	stdout, stderr bytes.Buffer
	runErr         error
}

func executeCLI(binary, work, root string, env, args []string, outputFailure bool) (*executionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if args == nil {
		args = []string{"export"}
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = work
	cmd.Env = env
	var result executionResult
	cmd.Stdout = &result.stdout
	cmd.Stderr = &result.stderr
	if outputFailure {
		file := filepath.Join(root, "read-only-output")
		if err := os.WriteFile(file, nil, 0600); err != nil {
			return nil, err
		}
		destination, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer destination.Close()
		cmd.Stdout = destination
	}
	result.runErr = cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &result, nil
}
