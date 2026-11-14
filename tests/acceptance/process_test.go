package acceptance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteCLISetupFailureKeepsResult(t *testing.T) {
	root := t.TempDir()
	result, err := executeCLI("unused", root, filepath.Join(root, "missing"), nil, []string{"validate"}, true)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("setup error = %v", err)
	}
	if result == nil {
		t.Fatal("setup failure discarded the diagnostic result")
	}
	if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Fatalf("setup failure ran a process: %+v", result)
	}
}

func TestExecuteCLIExpiredDeadlineKeepsResult(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()
	root := t.TempDir()
	result, err := executeCLIContext(ctx, processHelperBinary(t), root, root, os.Environ(), []string{"-test.run=^TestAcceptanceProcessHelper$"}, false)
	if !errors.Is(err, context.DeadlineExceeded) || result == nil {
		t.Fatalf("deadline result=%v error=%v", result, err)
	}
	if result.runErr == nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Fatalf("expired context started a process: %+v", result)
	}
}

func TestExecuteCLICancellationKeepsOutput(t *testing.T) {
	root := t.TempDir()
	ready := filepath.Join(root, "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type outcome struct {
		result *executionResult
		err    error
	}
	done := make(chan outcome, 1)
	binary := processHelperBinary(t)
	go func() {
		env := append(os.Environ(), "GH_SETTINGS_PROCESS_HELPER=wait", "GH_SETTINGS_PROCESS_READY="+ready)
		result, err := executeCLIContext(ctx, binary, root, root, env, []string{"-test.run=^TestAcceptanceProcessHelper$"}, false)
		done <- outcome{result, err}
	}()
	// Cancellation happens only after the child has written both streams. The
	// deadline bounds a failed handshake; process startup speed is not asserted.
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	readySeen := false
	for !readySeen {
		select {
		case got := <-done:
			t.Fatalf("child ended before handshake: result=%v error=%v", got.result, got.err)
		case <-ctx.Done():
			t.Fatal("child handshake did not complete", ctx.Err())
		case <-poll.C:
			_, err := os.Stat(ready)
			if err == nil {
				readySeen = true
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
		}
	}
	cancel()
	var got outcome
	select {
	case got = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("cancelled child did not stop")
	}
	if !errors.Is(got.err, context.Canceled) || got.result == nil {
		t.Fatalf("cancellation result=%v error=%v", got.result, got.err)
	}
	if got.result.runErr == nil || got.result.stdout.String() != "child stdout\n" || got.result.stderr.String() != "child stderr\n" {
		t.Fatalf("lost process failure/output: exit=%v stdout=%q stderr=%q", got.result.runErr, got.result.stdout.String(), got.result.stderr.String())
	}
}

func TestExecuteCLICommandExitKeepsResult(t *testing.T) {
	for _, mode := range []string{"success", "failure"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			env := append(os.Environ(), "GH_SETTINGS_PROCESS_HELPER="+mode)
			result, err := executeCLI(processHelperBinary(t), root, root, env, []string{"-test.run=^TestAcceptanceProcessHelper$"}, false)
			if err != nil || result == nil {
				t.Fatalf("ordinary command result=%v error=%v", result, err)
			}
			if mode == "success" && result.runErr != nil {
				t.Fatal(result.runErr)
			}
			if mode == "failure" {
				var exit *exec.ExitError
				if !errors.As(result.runErr, &exit) || exit.ExitCode() != 17 {
					t.Fatal("ordinary exit lost", result.runErr)
				}
			}
			if result.stdout.String() != "child stdout\n" || result.stderr.String() != "child stderr\n" {
				t.Fatalf("output lost: stdout=%q stderr=%q", result.stdout.String(), result.stderr.String())
			}
		})
	}
}

func processHelperBinary(t *testing.T) string {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return binary
}

func TestAcceptanceProcessHelper(t *testing.T) {
	mode := os.Getenv("GH_SETTINGS_PROCESS_HELPER")
	if mode == "" {
		return
	}
	fmt.Fprintln(os.Stdout, "child stdout")
	fmt.Fprintln(os.Stderr, "child stderr")
	switch mode {
	case "wait":
		if err := os.WriteFile(os.Getenv("GH_SETTINGS_PROCESS_READY"), nil, 0600); err != nil {
			os.Exit(18)
		}
		for {
			time.Sleep(time.Minute)
		}
	case "failure":
		os.Exit(17)
	case "success":
		os.Exit(0)
	default:
		os.Exit(19)
	}
}
