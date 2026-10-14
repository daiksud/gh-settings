//go:build !darwin && !linux

package main

import (
	"fmt"
	"os"
)

func openExportRoot(root string) (*os.File, error) {
	return nil, fmt.Errorf("safe export file output requires macOS or Linux")
}

func writeExportFile(root *os.File, data []byte) error {
	return fmt.Errorf("safe export file output requires macOS or Linux")
}
