package workspace

import (
	"strings"
	"testing"
)

func TestUIDeployKeysLiteralRoundtrip(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    deploy_keys:\n      - title: shared-title\n        key: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB comment-preserved\n        allow_write_access: false\n      - title: shared-title\n        key: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC\n        allow_write_access: true\n"
	value, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(value)
	if err != nil || !strings.Contains(string(data), "comment-preserved") || !strings.Contains(string(data), "allow_write_access: false") || !strings.Contains(string(data), "allow_write_access: true") {
		t.Fatal("literal key/configuration roundtrip", err)
	}
}
