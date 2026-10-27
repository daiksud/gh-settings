package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testMacosHash     = "69e398f9f4479aba0fcc353dc9acb6819c0a7d260f6e104ca65aae7ac2c21b85"
	testLinuxHash     = "8e05650e3597d536838387f4fb4f08fbd95624760f1ce44bbff4e35de8a353e8"
	testChecksumsHash = "98f9ece90dea10c02544983c6f8bc3e3ccef162fa20869eecebe1df30b297abe"
	testChecksums     = "69e398f9f4479aba0fcc353dc9acb6819c0a7d260f6e104ca65aae7ac2c21b85  gh-settings-darwin-arm64\n8e05650e3597d536838387f4fb4f08fbd95624760f1ce44bbff4e35de8a353e8  gh-settings-linux-amd64\n"
	testManifest      = `{"version":"v1.2.3","commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","macos_sha256":"69e398f9f4479aba0fcc353dc9acb6819c0a7d260f6e104ca65aae7ac2c21b85","linux_sha256":"8e05650e3597d536838387f4fb4f08fbd95624760f1ce44bbff4e35de8a353e8","checksums_sha256":"98f9ece90dea10c02544983c6f8bc3e3ccef162fa20869eecebe1df30b297abe"}`
)

func preflightFixture(t *testing.T) (string, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	version := filepath.Join(dir, "cli-version")
	for name, data := range map[string]string{"cli-version": "v1.2.3\n", "gh-settings-linux-amd64": "linux-binary", "gh-settings-darwin-arm64": "macos-binary"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return version, map[string]string{
		"VERSION": "v1.2.3", "SOURCE_SHA": strings.Repeat("a", 40),
		"GITHUB_REPOSITORY": "owner/repo", "RELEASE_DIR": dir,
		"LINUX_SHA256": testLinuxHash, "MACOS_SHA256": testMacosHash,
	}
}

func TestPreflightExactIdentityBytes(t *testing.T) {
	_, env := preflightFixture(t)
	identity, err := prepareIdentity(func(key string) string { return env[key] })
	if err != nil {
		t.Fatal(err)
	}
	want := releaseIdentity{
		Version: "v1.2.3", Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Repository: "owner/repo", Directory: env["RELEASE_DIR"],
		MacosSHA256: testMacosHash, LinuxSHA256: testLinuxHash, ChecksumsSHA256: testChecksumsHash,
		Manifest: testManifest, Marker: "<!-- gh-settings-release: " + testManifest + " -->",
	}
	if identity != want {
		t.Fatalf("identity=%+v, want %+v", identity, want)
	}
	data, err := os.ReadFile(filepath.Join(env["RELEASE_DIR"], "SHA256SUMS"))
	if err != nil || string(data) != testChecksums {
		t.Fatalf("SHA256SUMS=%q err=%v", data, err)
	}
}

func TestPreflightPreservesSymlinkDirectoryTraversal(t *testing.T) {
	_, env := preflightFixture(t)
	directory := env["RELEASE_DIR"]
	child := filepath.Join(directory, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(child, link); err != nil {
		t.Fatal(err)
	}
	env["RELEASE_DIR"] = link + "/.."
	identity, err := prepareIdentity(func(key string) string { return env[key] })
	if err != nil || identity.Directory != env["RELEASE_DIR"] {
		t.Fatalf("directory=%q error=%v", identity.Directory, err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "SHA256SUMS"))
	if err != nil || string(data) != testChecksums {
		t.Fatalf("SHA256SUMS=%q err=%v", data, err)
	}
}

func TestPreflightRejectsInvalidInputsBeforeChecksumWrite(t *testing.T) {
	for _, test := range []struct {
		name             string
		env              map[string]string
		missingDirectory bool
		remove, change   []string
		want             string
	}{
		{name: "malformed version", env: map[string]string{"VERSION": "v1.2.3;unsafe"}, want: "version must be vMAJOR.MINOR.PATCH"},
		{name: "leading zero", env: map[string]string{"VERSION": "v01.2.3"}, want: "version must be vMAJOR.MINOR.PATCH"},
		{name: "invalid source", env: map[string]string{"SOURCE_SHA": "main"}, want: "invalid source commit"},
		{name: "invalid Linux hash", env: map[string]string{"LINUX_SHA256": "bad"}, want: "invalid expected binary hash"},
		{name: "invalid macOS hash", env: map[string]string{"MACOS_SHA256": "bad"}, want: "invalid expected binary hash"},
		{name: "missing repository", env: map[string]string{"GITHUB_REPOSITORY": "owner"}, want: "missing repository"},
		{name: "missing directory", missingDirectory: true, want: "missing release directory"},
		{name: "missing Linux binary", remove: []string{"gh-settings-linux-amd64"}, want: "artifact hash mismatch: gh-settings-linux-amd64"},
		{name: "missing macOS binary", remove: []string{"gh-settings-darwin-arm64"}, want: "artifact hash mismatch: gh-settings-darwin-arm64"},
		{name: "changed Linux binary", change: []string{"gh-settings-linux-amd64"}, want: "artifact hash mismatch: gh-settings-linux-amd64"},
		{name: "changed macOS binary", change: []string{"gh-settings-darwin-arm64"}, want: "artifact hash mismatch: gh-settings-darwin-arm64"},
		{name: "version fails before source", env: map[string]string{"VERSION": "bad", "SOURCE_SHA": "bad"}, want: "version must be vMAJOR.MINOR.PATCH"},
		{name: "source fails before hashes", env: map[string]string{"SOURCE_SHA": "bad", "LINUX_SHA256": "bad"}, want: "invalid source commit"},
		{name: "hashes fail before repository", env: map[string]string{"LINUX_SHA256": "bad", "GITHUB_REPOSITORY": "bad"}, want: "invalid expected binary hash"},
		{name: "Linux fails before macOS", remove: []string{"gh-settings-linux-amd64", "gh-settings-darwin-arm64"}, want: "artifact hash mismatch: gh-settings-linux-amd64"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, env := preflightFixture(t)
			dir := env["RELEASE_DIR"]
			for key, value := range test.env {
				env[key] = value
			}
			if test.missingDirectory {
				env["RELEASE_DIR"] = filepath.Join(dir, "absent")
			}
			for _, name := range test.remove {
				if err := os.Remove(filepath.Join(dir, name)); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range test.change {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			identity, err := prepareIdentity(func(key string) string { return env[key] })
			if err == nil || err.Error() != test.want || identity != (releaseIdentity{}) {
				t.Fatalf("identity=%+v error=%v, want no identity and %q", identity, err, test.want)
			}
			if _, err := os.Stat(filepath.Join(dir, "SHA256SUMS")); !os.IsNotExist(err) {
				t.Fatalf("invalid inputs wrote SHA256SUMS: %v", err)
			}
		})
	}
}
