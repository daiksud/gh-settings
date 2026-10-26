// Command release-publisher publishes the immutable, tested release artifacts.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
	if err := runCommand(os.Args[1:], os.Stdout, ghCommand); err != nil {
		var failure *commandError
		if errors.As(err, &failure) {
			_, _ = os.Stderr.Write(failure.stderr)
			if failure.context != "" || len(failure.stderr) == 0 {
				fmt.Fprintln(os.Stderr, "release:", failure)
			}
			os.Exit(failure.status)
		}
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func runCommand(args []string, output io.Writer, gh ghAPI) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: release-publisher SOURCE_ROOT WORK_DIR")
	}
	identity, err := prepareIdentity(os.Getenv)
	if err != nil {
		return err
	}
	return publishRelease(identity, args[0], args[1], output, gh)
}

func publishRelease(identity releaseIdentity, root, work string, output io.Writer, gh ghAPI) error {
	observed, err := lookupRelease(identity, work, gh)
	if err != nil {
		return err
	}
	var notes string
	if !observed.Exists {
		// Freeze the complete body before creating any remote state. Existing
		// drafts and published releases retain their reviewed notes on rerun.
		notes, err = prepareNotes(identity, root, work)
		if err != nil {
			return err
		}
	}
	tag, err := inspectTag(identity, true, "existing tag targets another commit", gh)
	if err != nil {
		return err
	}
	if !tag.Exists {
		if observed.Exists {
			return fmt.Errorf("existing release has no tag; refusing to recreate it")
		}
		if _, err := gh("api", "-X", "POST", "repos/"+identity.Repository+"/git/refs", "-f", "ref=refs/tags/"+identity.Version, "-f", "sha="+identity.Commit); err != nil {
			return err
		}
	}
	id, view := observed.ID, observed.view
	if !observed.Exists {
		request := struct {
			Tag        string `json:"tag_name"`
			Commit     string `json:"target_commitish"`
			Name       string `json:"name"`
			Body       string `json:"body"`
			Draft      bool   `json:"draft"`
			Prerelease bool   `json:"prerelease"`
		}{identity.Version, identity.Commit, identity.Version, notes, true, false}
		data, err := mutateRelease(work+"/create.json", "POST", "repos/"+identity.Repository+"/releases", request, gh)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &view); err != nil {
			return fmt.Errorf("decode release: %w", err)
		}
		id, err = checkedReleaseID(view.ID, "created draft has no valid ID")
		if err != nil {
			return err
		}
	}
	if _, err := checkedReleaseState(identity, view, "initial"); err != nil {
		return err
	}
	assets, err := verifyAssets(identity, id, work, gh)
	if err != nil {
		return err
	}
	state, err := checkedReleaseState(identity, assets.view, "observed")
	if err != nil {
		return err
	}
	if state == publishedRelease {
		if len(assets.Missing) != 0 {
			return fmt.Errorf("published release is missing assets")
		}
		_, err := fmt.Fprintf(output, "release %s already published with matching identity\n", identity.Version)
		return err
	}
	for _, name := range assets.Missing {
		if _, err := gh("release", "upload", identity.Version, identity.Directory+"/"+name, "--repo", identity.Repository); err != nil {
			return err
		}
	}
	assets, err = verifyAssets(identity, id, work, gh)
	if err != nil {
		return err
	}
	if len(assets.Missing) != 0 {
		return fmt.Errorf("draft is missing assets")
	}
	if _, err := checkedReleaseState(identity, assets.view, "upload"); err != nil {
		return err
	}
	if _, err := inspectTag(identity, false, "tag changed before publication", gh); err != nil {
		return err
	}
	endpoint := "repos/" + identity.Repository + "/releases/" + id
	if _, err := mutateRelease(work+"/publish.json", "PATCH", endpoint, struct {
		Draft bool `json:"draft"`
	}{false}, gh); err != nil {
		return err
	}
	data, err := gh("api", endpoint)
	if err != nil {
		return err
	}
	var confirmation releaseView
	if err := json.Unmarshal(data, &confirmation); err != nil {
		return fmt.Errorf("decode release: %w", err)
	}
	if _, err := checkedReleaseState(identity, confirmation, "published"); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "published %s at %s with verified assets\n", identity.Version, identity.Commit)
	return err
}

// Request files are input to gh, not intermediate publisher state.
func mutateRelease(path, method, endpoint string, request any, gh ghAPI) ([]byte, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return nil, err
	}
	return gh("api", "-X", method, endpoint, "--input", path)
}

func prepareNotes(identity releaseIdentity, root, work string) (string, error) {
	command := exec.Command("go", "run", "./scripts/release-metadata/main.go", identity.Repository, identity.Version, identity.Commit, work+"/releases.json")
	command.Dir = root
	var changes, stderr bytes.Buffer
	command.Stdout, command.Stderr = &changes, &stderr
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return "", err
		}
		return "", &commandError{stderr: stderr.Bytes(), status: max(1, exit.ExitCode())}
	}
	if _, err := os.Stderr.Write(stderr.Bytes()); err != nil {
		return "", err
	}
	checksums, err := os.ReadFile(identity.Directory + "/" + checksumsAsset)
	if err != nil {
		return "", err
	}
	extension := identity.Repository[strings.LastIndex(identity.Repository, "/")+1:]
	return strings.Join([]string{
		identity.Marker, "", strings.TrimRight(changes.String(), "\n"), "", "---", "",
		"## 📦 Installation", "", "```sh", "gh extension install " + identity.Repository, "```", "",
		"Already installed? Run `gh extension upgrade " + extension + "`.", "",
		"Or download `" + macosAsset + "` (macOS Apple Silicon) or `" + linuxAsset + "` (Linux x86_64) below and verify it with `SHA256SUMS`.", "",
		"<details>", "<summary>🔐 SHA-256 checksums</summary>", "", "```text", strings.TrimRight(string(checksums), "\n"), "```", "", "</details>", "",
		"Built from main commit [`" + identity.Commit[:7] + "`](https://github.com/" + identity.Repository + "/commit/" + identity.Commit + ").", "",
	}, "\n"), nil
}

func prepareIdentity(getenv func(string) string) (releaseIdentity, error) {
	var identity releaseIdentity
	identity.Version = getenv("VERSION")
	if !versionPattern.MatchString(identity.Version) {
		return releaseIdentity{}, fmt.Errorf("version must be vMAJOR.MINOR.PATCH")
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

// ghAPI preserves the existing gh authentication and argument boundary.
type ghAPI func(...string) ([]byte, error)

type commandError struct {
	stderr  []byte
	status  int
	context string
}

func (failure *commandError) Error() string {
	if failure.context != "" {
		return failure.context
	}
	if len(failure.stderr) != 0 {
		return strings.TrimSpace(string(failure.stderr))
	}
	return fmt.Sprintf("gh exited with status %d", failure.status)
}
func ghCommand(args ...string) ([]byte, error) {
	command := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return nil, err
		}
		status := exit.ExitCode()
		if status < 1 {
			status = 1
		}
		return stdout.Bytes(), &commandError{stderr: stderr.Bytes(), status: status}
	}
	return stdout.Bytes(), nil
}

type releaseView struct {
	ID         json.RawMessage `json:"id"`
	Tag        json.RawMessage `json:"tag_name"`
	Name       json.RawMessage `json:"name"`
	Prerelease json.RawMessage `json:"prerelease"`
	Draft      json.RawMessage `json:"draft"`
	Body       json.RawMessage `json:"body"`
}

// Preserve the scalar forms previously accepted by jq -r at each checkpoint.
func scalarText(raw json.RawMessage) string {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return "null"
	}
	switch value := value.(type) {
	case string:
		return strings.TrimRight(value, "\n")
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(value)
	case json.Number:
		text := value.String()
		mantissa, exponent, scientific := strings.Cut(strings.ToLower(text), "e")
		if !scientific || strings.HasPrefix(text, "-") {
			return text
		}
		_, fraction, _ := strings.Cut(mantissa, ".")
		power, err := strconv.Atoi(exponent)
		// jq 1.8 retains decimal scale: 123.0 and 1E+6 do not pass the
		// integer-ID check, while 1.23e2 is rendered as 123.
		if err == nil && power == len(fraction) {
			digits := strings.TrimLeft(strings.ReplaceAll(mantissa, ".", ""), "0")
			if digits == "" {
				return "0"
			}
			return digits
		}
		return text
	}
	return strings.TrimSpace(string(raw))
}

var releaseIDPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

func checkedReleaseID(raw json.RawMessage, message string) (string, error) {
	id := scalarText(raw)
	if !releaseIDPattern.MatchString(id) {
		return "", fmt.Errorf("%s", message)
	}
	return id, nil
}
func markerMatches(identity releaseIdentity, view releaseView) bool {
	var body string
	return json.Unmarshal(view.Body, &body) == nil && strings.HasPrefix(body, identity.Marker+"\n")
}

type releaseLookup struct {
	Exists bool
	ID     string
	view   releaseView
}

func lookupRelease(identity releaseIdentity, work string, gh ghAPI) (releaseLookup, error) {
	catalog, err := gh("api", "repos/"+identity.Repository+"/releases?per_page=100", "--paginate", "--slurp")
	if err != nil {
		return releaseLookup{}, err
	}
	if err := os.WriteFile(filepath.Join(work, "releases.json"), catalog, 0600); err != nil {
		return releaseLookup{}, err
	}
	var pages [][]releaseView
	if err := json.Unmarshal(catalog, &pages); err != nil {
		return releaseLookup{}, fmt.Errorf("decode release list: %w", err)
	}
	if pages == nil {
		return releaseLookup{}, fmt.Errorf("missing release list")
	}
	var matches []releaseView
	for _, page := range pages {
		if page == nil {
			return releaseLookup{}, fmt.Errorf("missing release list page")
		}
		for _, view := range page {
			var tag string
			if json.Unmarshal(view.Tag, &tag) == nil && tag == identity.Version {
				matches = append(matches, view)
			}
		}
	}
	if len(matches) > 1 {
		return releaseLookup{}, fmt.Errorf("multiple releases use the requested tag")
	}
	if len(matches) == 0 {
		return releaseLookup{}, nil
	}
	id, err := checkedReleaseID(matches[0].ID, "release has no valid ID")
	if err != nil {
		return releaseLookup{}, err
	}
	data, err := gh("api", "repos/"+identity.Repository+"/releases/"+id)
	if err != nil {
		return releaseLookup{}, err
	}
	var view releaseView
	if err := json.Unmarshal(data, &view); err != nil {
		return releaseLookup{}, fmt.Errorf("decode release: %w", err)
	}
	if scalarText(view.ID) != id {
		return releaseLookup{}, fmt.Errorf("release ID changed")
	}
	if scalarText(view.Tag) != identity.Version {
		return releaseLookup{}, fmt.Errorf("existing release has another tag")
	}
	if scalarText(view.Name) != identity.Version {
		return releaseLookup{}, fmt.Errorf("existing release has another title")
	}
	if scalarText(view.Prerelease) != "false" {
		return releaseLookup{}, fmt.Errorf("existing release is a prerelease")
	}
	if !markerMatches(identity, view) {
		return releaseLookup{}, fmt.Errorf("existing release lacks matching version, commit, and binary identity evidence")
	}
	// Draft state is deliberately unchecked until after tag processing/creation.
	return releaseLookup{Exists: true, ID: id, view: view}, nil
}

type tagObservation struct {
	Exists bool   `json:"exists"`
	Type   string `json:"type"`
	SHA    string `json:"sha"`
}

func inspectTag(identity releaseIdentity, optional bool, context string, gh ghAPI) (tagObservation, error) {
	endpoint := "repos/" + identity.Repository + "/git/ref/tags/" + identity.Version
	data, err := gh("api", endpoint)
	if err != nil {
		var failure *commandError
		if optional && errors.As(err, &failure) {
			if bytes.Contains(failure.stderr, []byte("HTTP 404")) {
				return tagObservation{}, nil
			}
			copy := *failure
			copy.context, copy.status = "GitHub API lookup failed: "+endpoint, 1
			return tagObservation{}, &copy
		}
		return tagObservation{}, err
	}
	decode := func(data []byte) (string, string, error) {
		var view struct {
			Object struct {
				Type json.RawMessage `json:"type"`
				SHA  json.RawMessage `json:"sha"`
			} `json:"object"`
		}
		if err := json.Unmarshal(data, &view); err != nil {
			return "", "", err
		}
		return scalarText(view.Object.Type), scalarText(view.Object.SHA), nil
	}
	kind, sha, err := decode(data)
	if err != nil {
		return tagObservation{}, fmt.Errorf("decode tag: %w", err)
	}
	if kind == "tag" {
		data, err = gh("api", "repos/"+identity.Repository+"/git/tags/"+sha)
		if err != nil {
			return tagObservation{}, err
		}
		kind, sha, err = decode(data)
		if err != nil {
			return tagObservation{}, fmt.Errorf("decode annotated tag: %w", err)
		}
	}
	if kind != "commit" || sha != identity.Commit {
		return tagObservation{}, fmt.Errorf("%s", context)
	}
	return tagObservation{Exists: true, Type: kind, SHA: sha}, nil
}

type releaseState string

const (
	draftRelease     releaseState = "draft"
	publishedRelease releaseState = "published"
)

func checkedReleaseState(identity releaseIdentity, view releaseView, checkpoint string) (releaseState, error) {
	draft := scalarText(view.Draft)
	switch checkpoint {
	case "initial", "observed":
		if draft != "true" && draft != "false" {
			return "", fmt.Errorf("unknown release state")
		}
		if checkpoint == "initial" && !markerMatches(identity, view) {
			return "", fmt.Errorf("release notes do not match tested identity")
		}
	case "upload":
		if draft != "true" {
			return "", fmt.Errorf("release changed state during upload")
		}
		if !markerMatches(identity, view) {
			return "", fmt.Errorf("release notes changed during upload")
		}
	case "published":
		if draft != "false" {
			return "", fmt.Errorf("release publication not confirmed")
		}
	default:
		return "", fmt.Errorf("unknown release checkpoint %q", checkpoint)
	}
	if draft == "true" {
		return draftRelease, nil
	}
	return publishedRelease, nil
}

type assetVerification struct {
	Verified []string
	Missing  []string
	view     releaseView
}

func verifyAssets(identity releaseIdentity, id, work string, gh ghAPI) (assetVerification, error) {
	data, err := gh("api", "repos/"+identity.Repository+"/releases/"+id)
	if err != nil {
		return assetVerification{}, err
	}
	var response struct {
		Assets *[]struct {
			Name json.RawMessage `json:"name"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return assetVerification{}, fmt.Errorf("decode release assets: %w", err)
	}
	if response.Assets == nil {
		return assetVerification{}, fmt.Errorf("missing release assets")
	}
	seen := make(map[string]bool)
	names := make([]string, 0, len(*response.Assets))
	// Check the whole name list for duplicates before rejecting unexpected names,
	// preserving the original validation order even for duplicate unknown names.
	for _, asset := range *response.Assets {
		var value any
		if len(asset.Name) != 0 {
			if err := json.Unmarshal(asset.Name, &value); err != nil {
				return assetVerification{}, fmt.Errorf("decode asset name: %w", err)
			}
		}
		key, err := json.Marshal(value)
		if err != nil {
			return assetVerification{}, fmt.Errorf("encode asset name: %w", err)
		}
		if seen[string(key)] {
			return assetVerification{}, fmt.Errorf("duplicate asset names")
		}
		seen[string(key)] = true
		name, _ := value.(string)
		names = append(names, name)
	}
	present := make(map[string]bool)
	for _, name := range names {
		if name != macosAsset && name != linuxAsset && name != checksumsAsset {
			return assetVerification{}, fmt.Errorf("unexpected release asset")
		}
		present[name] = true
	}
	downloads, err := os.MkdirTemp(work, "downloads.")
	if err != nil {
		return assetVerification{}, err
	}
	result := assetVerification{Verified: []string{}, Missing: []string{}}
	for _, asset := range []struct{ name, digest string }{
		{macosAsset, identity.MacosSHA256}, {linuxAsset, identity.LinuxSHA256}, {checksumsAsset, identity.ChecksumsSHA256},
	} {
		if !present[asset.name] {
			result.Missing = append(result.Missing, asset.name)
			continue
		}
		if _, err := gh("release", "download", identity.Version, "--repo", identity.Repository, "--pattern", asset.name, "--dir", downloads); err != nil {
			return assetVerification{}, err
		}
		if err := checkBinary(downloads+"/"+asset.name, asset.digest); err != nil {
			return assetVerification{}, err
		}
		result.Verified = append(result.Verified, asset.name)
	}
	if err := json.Unmarshal(data, &result.view); err != nil {
		return assetVerification{}, fmt.Errorf("decode release: %w", err)
	}
	return result, nil
}
