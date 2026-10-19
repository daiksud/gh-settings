// Command release-publisher validates the immutable inputs to release publication.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	linuxAsset     = "gh-settings-linux-amd64"
	macosAsset     = "gh-settings-darwin-arm64"
	checksumsAsset = "SHA256SUMS"
)

var (
	versionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	sourcePattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	hashPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type releaseIdentity struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	Repository      string `json:"repository"`
	Directory       string `json:"directory"`
	MacosSHA256     string `json:"macos_sha256"`
	LinuxSHA256     string `json:"linux_sha256"`
	ChecksumsSHA256 string `json:"checksums_sha256"`
	Manifest        string `json:"manifest"`
	Marker          string `json:"marker"`
}

func main() {
	if len(os.Args) != 3 || os.Args[1] != "preflight" {
		fmt.Fprintln(os.Stderr, "usage: release-publisher preflight COMMITTED_VERSION_FILE")
		os.Exit(1)
	}
	identity, err := prepareIdentity(os.Args[2], os.Getenv)
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(identity)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func prepareIdentity(committedVersionFile string, getenv func(string) string) (releaseIdentity, error) {
	var identity releaseIdentity
	identity.Version = getenv("VERSION")
	if !versionPattern.MatchString(identity.Version) {
		return releaseIdentity{}, fmt.Errorf("version must be vMAJOR.MINOR.PATCH")
	}
	committed, err := os.ReadFile(committedVersionFile)
	if err != nil || strings.TrimRight(string(committed), "\n") != identity.Version {
		return releaseIdentity{}, fmt.Errorf("release version differs from committed CLI version")
	}
	identity.Commit = getenv("SOURCE_SHA")
	if !sourcePattern.MatchString(identity.Commit) {
		return releaseIdentity{}, fmt.Errorf("invalid source commit")
	}
	identity.LinuxSHA256, identity.MacosSHA256 = getenv("LINUX_SHA256"), getenv("MACOS_SHA256")
	if !hashPattern.MatchString(identity.LinuxSHA256) || !hashPattern.MatchString(identity.MacosSHA256) {
		return releaseIdentity{}, fmt.Errorf("invalid expected binary hash")
	}
	identity.Repository = getenv("GITHUB_REPOSITORY")
	if !strings.Contains(identity.Repository, "/") {
		return releaseIdentity{}, fmt.Errorf("missing repository")
	}
	identity.Directory = getenv("RELEASE_DIR")
	info, err := os.Stat(identity.Directory)
	if err != nil || !info.IsDir() {
		return releaseIdentity{}, fmt.Errorf("missing release directory")
	}
	// Preserve filesystem traversal through symlinks and "..", as the shell's
	// quoted directory/asset paths did; filepath.Join would clean them lexically.
	if err := checkBinary(identity.Directory+"/"+linuxAsset, identity.LinuxSHA256); err != nil {
		return releaseIdentity{}, err
	}
	if err := checkBinary(identity.Directory+"/"+macosAsset, identity.MacosSHA256); err != nil {
		return releaseIdentity{}, err
	}
	checksums := []byte(fmt.Sprintf("%s  %s\n%s  %s\n", identity.MacosSHA256, macosAsset, identity.LinuxSHA256, linuxAsset))
	if err := os.WriteFile(identity.Directory+"/"+checksumsAsset, checksums, 0666); err != nil {
		return releaseIdentity{}, fmt.Errorf("write SHA256SUMS: %w", err)
	}
	sum := sha256.Sum256(checksums)
	identity.ChecksumsSHA256 = hex.EncodeToString(sum[:])
	manifest, err := json.Marshal(struct {
		Version         string `json:"version"`
		Commit          string `json:"commit"`
		MacosSHA256     string `json:"macos_sha256"`
		LinuxSHA256     string `json:"linux_sha256"`
		ChecksumsSHA256 string `json:"checksums_sha256"`
	}{identity.Version, identity.Commit, identity.MacosSHA256, identity.LinuxSHA256, identity.ChecksumsSHA256})
	if err != nil {
		return releaseIdentity{}, fmt.Errorf("encode release manifest: %w", err)
	}
	identity.Manifest = string(manifest)
	identity.Marker = "<!-- gh-settings-release: " + identity.Manifest + " -->"
	return identity, nil
}

func checkBinary(path, expected string) error {
	info, err := os.Stat(path)
	if err == nil && info.Mode().IsRegular() {
		var digest string
		digest, err = digestFile(path)
		if err == nil && digest == expected {
			return nil
		}
	}
	return fmt.Errorf("artifact hash mismatch: %s", filepath.Base(path))
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
