//go:build !darwin && !linux

package main

import "fmt"

func writeExportFile(root string, data []byte) error {
	return fmt.Errorf("safe export file output requires macOS or Linux")
}
