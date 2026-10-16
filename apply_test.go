package main

import (
	"bytes"
	"testing"
)

func TestApplyOutputFailure(t *testing.T) {
	for _, message := range []string{"Applied repository settings.\n", "No managed repository settings.\n"} {
		var stderr bytes.Buffer
		code := run([]string{"apply"}, brokenWriter{}, &stderr, commandOperations{apply: func() ([]byte, error) { return []byte(message), nil }})
		if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
			t.Fatalf("status=%d stderr=%q", code, stderr.String())
		}
	}
}
