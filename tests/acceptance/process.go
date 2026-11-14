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

// Full export probes every supported family. Keep a finite harness budget with
// headroom for hosted runners; this does not change any product API timeout.
const acceptanceProcessTimeout = 60 * time.Second

func executeCLI(binary, work, root string, env, args []string, outputFailure bool) (*executionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), acceptanceProcessTimeout)
	defer cancel()
	return executeCLIContext(ctx, binary, work, root, env, args, outputFailure)
}

// Always return a result, including setup/cancellation failures. Callers can
// report captured output and runErr without losing the separate harness error.
func executeCLIContext(ctx context.Context, binary, work, root string, env, args []string, outputFailure bool) (*executionResult, error) {
	result := &executionResult{}
	if args == nil {
		args = []string{"export"}
	}
	if v, ok := processFixtures.Load(root); ok {
		f := v.(*apiFixture)
		f.exporting.Store(len(args) > 0 && args[0] == "export")
		defer f.exporting.Store(false)
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = work
	cmd.Env = env
	cmd.Stdout = &result.stdout
	cmd.Stderr = &result.stderr
	if outputFailure {
		file := filepath.Join(root, "read-only-output")
		if err := os.WriteFile(file, nil, 0600); err != nil {
			return result, err
		}
		destination, err := os.Open(file)
		if err != nil {
			return result, err
		}
		defer destination.Close()
		cmd.Stdout = destination
	}
	result.runErr = cmd.Run()
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, nil
}
