package actions_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestMetadataUsesRunningImage(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("GHS_METADATA_CHILD") != executable {
		// Replace only a child copy owned by this test, never the test runner.
		child := filepath.Join(t.TempDir(), "metadata-child.test")
		if err := os.WriteFile(child, original, 0700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(child, "-test.run=^TestMetadataUsesRunningImage$", "-test.count=1")
		cmd.Env = append(os.Environ(), "GHS_METADATA_CHILD="+child)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("metadata child: %v: %s", err, output)
		}
		return
	}
	metadataCheckout(t)
	digest := fmt.Sprintf("%x", sha256.Sum256(original))
	replacement := filepath.Join(filepath.Dir(executable), "replacement")
	if err := os.WriteFile(replacement, []byte("replacement image"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, executable); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"metadata"}, &stdout, &stderr, "fixture-version")
	if status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "cli-sha256="+digest+"\n") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
