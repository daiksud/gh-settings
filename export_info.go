package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/daiksud/gh-settings/internal/app"
)

func exportInfo(path string, cause error) error { return writeExportInfo(os.Stderr, path, cause) }

// Never render upstream messages, response bodies, URLs or supplied values.
func writeExportInfo(output io.Writer, path string, cause error) error {
	reason := "observation unavailable, incomplete or unsupported"
	var httpError *api.HTTPError
	var networkError net.Error
	if errors.Is(cause, app.ErrExportCodeQLDisabled) {
		_, err := fmt.Fprintf(output, "info: export %s: CodeQL default setup is confirmed disabled; existing configuration removed from export\n", path)
		if err != nil {
			return fmt.Errorf("write export info: %w", err)
		}
		return nil
	} else if errors.As(cause, &httpError) {
		reason = fmt.Sprintf("HTTP %d", httpError.StatusCode)
	} else if errors.As(cause, &networkError) {
		if networkError.Timeout() {
			reason = "request timed out"
		} else {
			reason = "network request failed"
		}
	}
	_, err := fmt.Fprintf(output, "info: export %s: %s; omitted on first export, existing declaration retained if present\n", path, reason)
	if err != nil {
		return fmt.Errorf("write export info: %w", err)
	}
	return nil
}
