package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		t.Fatal("version command must not call GitHub")
		return nil, nil
	}})
	if code != 0 || stdout.String() != "gh-settings "+version+"\n" || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestHelpAndVersionOutputFailure(t *testing.T) {
	for _, test := range []struct {
		arg        string
		wantStderr string
	}{
		{"--help", "gh settings: write help: output unavailable\n"},
		{"-h", "gh settings: write help: output unavailable\n"},
		{"--version", "gh settings: write version: output unavailable\n"},
		{"-v", "gh settings: write version: output unavailable\n"},
	} {
		t.Run(test.arg, func(t *testing.T) {
			var stderr bytes.Buffer
			status := run([]string{test.arg}, brokenWriter{}, &stderr, commandOperations{
				export: func() ([]byte, error) {
					t.Fatal("help/version output failure invoked export")
					return nil, nil
				},
				validate: func() error {
					t.Fatal("help/version output failure invoked validate")
					return nil
				},
			})
			if status != 1 || stderr.String() != test.wantStderr {
				t.Fatalf("status=%d stderr=%q, want %q", status, stderr.String(), test.wantStderr)
			}
		})
	}
}

func TestRunSelectsOperation(t *testing.T) {
	operationError := errors.New("operation failed")
	for _, test := range []struct {
		name              string
		args              []string
		operationError    error
		wantExportCalls   int
		wantValidateCalls int
		wantStatus        int
	}{
		{"export", []string{"export"}, nil, 1, 0, 0},
		{"validate", []string{"validate"}, nil, 0, 1, 0},
		{"export failure", []string{"export"}, operationError, 1, 0, 1},
		{"validate failure", []string{"validate"}, operationError, 0, 1, 1},
		{"help", []string{"--help"}, nil, 0, 0, 0},
		{"short help", []string{"-h"}, nil, 0, 0, 0},
		{"version", []string{"--version"}, nil, 0, 0, 0},
		{"short version", []string{"-v"}, nil, 0, 0, 0},
		{"no arguments", nil, nil, 0, 0, 1},
		{"apply", []string{"apply"}, nil, 0, 0, 1},
		{"plan", []string{"plan"}, nil, 0, 0, 1},
		{"export arguments", []string{"export", "--repo", "owner/other"}, nil, 0, 0, 1},
		{"validate arguments", []string{"validate", "--repo", "owner/other"}, nil, 0, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exportCalls, validateCalls := 0, 0
			status := run(test.args, &stdout, &stderr, commandOperations{
				export: func() ([]byte, error) {
					exportCalls++
					return []byte("settings\n"), test.operationError
				},
				validate: func() error {
					validateCalls++
					return test.operationError
				},
			})
			if exportCalls != test.wantExportCalls || validateCalls != test.wantValidateCalls {
				t.Fatalf("export calls=%d, validate calls=%d", exportCalls, validateCalls)
			}
			if status != test.wantStatus {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
			if test.operationError != nil && (stdout.Len() != 0 || stderr.String() != "gh settings: operation failed\n") {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "validate" && (stdout.Len() != 0 || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "export" && (stdout.String() != "settings\n" || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestUnsupportedCommand(t *testing.T) {
	for _, args := range [][]string{nil, {"apply"}, {"plan"}, {"export", "--repo", "owner/other"}, {"validate", "--repo", "owner/other"}} {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
			t.Fatal("unsupported command invoked API operation")
			return nil, errors.New("unexpected API call")
		}})
		if code != 1 || stdout.Len() != 0 || stderr.String() != "gh settings: unsupported command or arguments; usage: gh settings <export|validate>\n" {
			t.Fatalf("args=%v status=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestExportOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"export"}, brokenWriter{}, &stderr, commandOperations{export: func() ([]byte, error) { return []byte("description: test\n"), nil }})
	if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"--help"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) { t.Fatal("help invoked API operation"); return nil, nil }})
	if status != 0 || !strings.Contains(stdout.String(), "gh settings export") || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
