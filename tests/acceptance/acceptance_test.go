//go:build acceptance

package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var binary = flag.String("binary", "", "path to the already-built gh-settings executable")

func TestAcceptance(t *testing.T) {
	path := acceptanceBinary(t)
	for _, test := range exportScenarios() {
		t.Run(test.name, func(t *testing.T) {
			if err := check(path, test); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, test := range validateScenarios() {
		t.Run(test.name, func(t *testing.T) {
			if err := checkValidate(path, test); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func acceptanceBinary(t *testing.T) string {
	t.Helper()
	if *binary == "" {
		t.Fatal("acceptance requires -binary PATH")
	}
	path, err := filepath.Abs(*binary)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("acceptance binary %q: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("acceptance binary %q is not a regular file", path)
	}
	return path
}
