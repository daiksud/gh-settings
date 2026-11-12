package main

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestExportInfoNeverLogsUpstreamSecrets(t *testing.T) {
	secret := "secret-token-and-value"
	for _, cause := range []error{errors.New(secret), fmtWrapped(&api.HTTPError{StatusCode: 403, Message: secret, RequestURL: &url.URL{Scheme: "https", Host: "example.com", RawQuery: "token=" + secret}})} {
		var b strings.Builder
		if err := writeExportInfo(&b, "repository.settings.advanced_security", cause); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(b.String(), secret) || !strings.HasPrefix(b.String(), "info: export ") {
			t.Fatal(b.String())
		}
		var he *api.HTTPError
		if errors.As(cause, &he) && !strings.Contains(b.String(), "HTTP 403") {
			t.Fatal(b.String())
		}
	}
	if err := writeExportInfo(exportBrokenWriter{}, "resource", errors.New(secret)); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}
func fmtWrapped(e error) error { return &wrappedExportTestError{e} }

type wrappedExportTestError struct{ error }

func (e *wrappedExportTestError) Unwrap() error { return e.error }

type exportBrokenWriter struct{}

func (exportBrokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
