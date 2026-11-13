package workspace

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"os"
	"strings"
	"testing"
)

func TestCreatedDeployKeyIDCanonicalSelection(t *testing.T) {
	first := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
	second := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC"
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    deploy_keys:\n      - title: same\n        key: " + first + " first-comment # YAML comment\n        allow_write_access: false\n      - title: same\n        key: " + second + " second-comment\n        allow_write_access: true\n"
	root, path := writebackFixture(t, input)
	record := createdIDRecorder(root, []byte(input))
	a, _ := settings.ParseDeployPublicKey(first)
	b, _ := settings.ParseDeployPublicKey(second)
	if err := record("deploy_keys", b.Fingerprint(), 22); err != nil {
		t.Fatal(err)
	}
	if err := record("deploy_keys", a.Fingerprint(), 11); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	decoded, err := DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	rows := *decoded.Settings.DeployKeys
	if *rows[0].ID != 11 || *rows[1].ID != 22 || !strings.Contains(string(data), "first-comment # YAML comment") || !strings.Contains(string(data), "second-comment") {
		t.Fatal("wrong binding or changed source", string(data))
	}
	if err := record("deploy_keys", a.Fingerprint(), 33); err == nil {
		t.Fatal("replaced bound id")
	}
	for _, mode := range []string{"ambiguous", "concurrent", "alias", "merged"} {
		t.Run(mode, func(t *testing.T) {
			text := input
			if mode == "ambiguous" {
				text = strings.ReplaceAll(text, second, first)
			}
			if mode == "alias" {
				text = "value: &row {title: same, key: " + first + ", allow_write_access: false}\nrepository: {settings: {deploy_keys: [*row]}}\n"
			}
			if mode == "merged" {
				text = "repository: {settings: {deploy_keys: [{key: " + first + ", <<: {title: same, allow_write_access: false}}]}}\n"
			}
			root, path := writebackFixture(t, text)
			r := createdIDRecorder(root, []byte(text))
			want := text
			if mode == "concurrent" {
				want += "# changed\n"
				os.WriteFile(path, []byte(want), 0600)
			}
			if err := r("deploy_keys", a.Fingerprint(), 11); err == nil {
				t.Fatal("unsafe binding")
			}
			data, _ := os.ReadFile(path)
			if string(data) != want {
				t.Fatal("file changed on failure")
			}
		})
	}
}
