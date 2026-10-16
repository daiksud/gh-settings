package main

import (
	"bytes"
	"testing"
)

func TestPlanOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"plan"}, brokenWriter{}, &stderr, commandOperations{plan: func() ([]byte, error) { return []byte("No managed repository settings.\n"), nil }})
	if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}
