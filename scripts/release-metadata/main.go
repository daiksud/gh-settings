// Command release-metadata computes versions and requests GitHub release notes.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	versionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	sourcePattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	hashPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	repoPattern    = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	headerPattern  = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*)(\(([^()]+)\))?(!)?: (.+)$`)
	footerPattern  = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*|BREAKING CHANGE)(: | #)(.*)$`)
	fencePattern   = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
)

type release struct {
	ID          int64      `json:"id"`
	Tag         string     `json:"tag_name"`
	Draft       *bool      `json:"draft"`
	Prerelease  *bool      `json:"prerelease"`
	PublishedAt *time.Time `json:"published_at"`
	Body        *string    `json:"body"`
}

type change struct {
	sha, subject, kind, scope string
	major, minor              bool
	breaking                  []string
}

type interval struct {
	previous    *release
	previousSHA string
	changes     []change
}

type proposal struct {
	version, previousTag, previousSHA string
	reuse                             bool
}

type ghAPI func(...string) ([]byte, error)

func main() {
	if err := run(".", os.Args[1:], os.Stdout, ghCommand); err != nil {
		fmt.Fprintln(os.Stderr, "release-metadata:", err)
		os.Exit(1)
	}
}

func run(directory string, args []string, output io.Writer, gh ghAPI) error {
	if len(args) == 2 && args[0] == "prepare" {
		return prepare(directory, args[1], output, gh)
	}
	if len(args) == 3 && args[0] == "resolve" {
		repository, source := args[1], args[2]
		if !repoPattern.MatchString(repository) || !sourcePattern.MatchString(source) {
			return fmt.Errorf("invalid repository or source commit")
		}
		catalog, err := gh("api", "repos/"+repository+"/releases?per_page=100", "--paginate", "--slurp")
		if err != nil {
			return err
		}
		p, err := propose(directory, source, bytes.NewReader(catalog), true)
		if err != nil {
			return err
		}
		committed, err := git(directory, "show", source+":action/cli-version")
		if err != nil || strings.TrimSpace(committed) != p.version {
			return fmt.Errorf("prepare and merge action/cli-version=%s before dispatching", p.version)
		}
		_, err = fmt.Fprintf(output, "version=%s\nprevious-tag=%s\nprevious-sha=%s\nsource-sha=%s\n", p.version, p.previousTag, p.previousSHA, source)
		return err
	}
	if len(args) == 4 {
		catalog, err := os.Open(args[3])
		if err != nil {
			return err
		}
		defer catalog.Close()
		notes, err := generate(directory, args[0], args[1], args[2], catalog, os.Getenv("PREVIOUS_TAG"), os.Getenv("PREVIOUS_SHA"), gh)
		if err != nil {
			return err
		}
		_, err = io.WriteString(output, notes)
		return err
	}
	return fmt.Errorf("usage: release-metadata prepare OWNER/REPO | resolve OWNER/REPO SOURCE_SHA | OWNER/REPO VERSION SOURCE_SHA RELEASES_JSON")
}

func prepare(directory, repository string, output io.Writer, gh ghAPI) error {
	if !repoPattern.MatchString(repository) {
		return fmt.Errorf("invalid repository")
	}
	status, err := git(directory, "status", "--porcelain", "--untracked-files=no")
	if err != nil || status != "" {
		return fmt.Errorf("commit or preserve tracked changes before version preparation")
	}
	source, err := git(directory, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	catalog, err := gh("api", "repos/"+repository+"/releases?per_page=100", "--paginate", "--slurp")
	if err != nil {
		return err
	}
	p, err := propose(directory, source, bytes.NewReader(catalog), false)
	if err != nil {
		return err
	}
	path := filepath.Join(directory, "action", "cli-version")
	if _, err := git(directory, "show", source+":action/cli-version"); err != nil {
		return fmt.Errorf("committed action/cli-version required")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("regular committed action/cli-version required")
	}
	if err := os.WriteFile(path, []byte(p.version+"\n"), info.Mode().Perm()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Prepared %s from %s. Review and commit action/cli-version; no tag or release was created.\n", p.version, p.previousTag)
	return err
}

func readCatalog(reader io.Reader) ([]release, error) {
	var pages [][]release
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&pages); err != nil {
		return nil, fmt.Errorf("read published releases: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF || len(pages) == 0 {
		return nil, fmt.Errorf("missing or invalid paginated release list")
	}
	var result []release
	seen := map[int64]bool{}
	for _, page := range pages {
		if page == nil {
			return nil, fmt.Errorf("missing release list page")
		}
		for _, r := range page {
			if r.ID <= 0 || r.Tag == "" || r.Draft == nil || r.Prerelease == nil || (!*r.Draft && r.PublishedAt == nil) || seen[r.ID] {
				return nil, fmt.Errorf("incomplete or duplicated release metadata")
			}
			seen[r.ID] = true
			result = append(result, r)
		}
	}
	return result, nil
}

func history(directory, source, excludeTag string, catalog []release) (interval, error) {
	var result interval
	if !sourcePattern.MatchString(source) {
		return result, fmt.Errorf("source must be a full commit SHA")
	}
	shallow, err := git(directory, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(shallow) != "false" {
		return result, fmt.Errorf("complete Git history is required")
	}
	resolved, err := git(directory, "rev-parse", "--verify", source+"^{commit}")
	if err != nil || strings.TrimSpace(resolved) != source {
		return result, fmt.Errorf("source must identify a commit directly")
	}
	for _, r := range catalog {
		if r.Tag == excludeTag || *r.Draft || *r.Prerelease || !versionPattern.MatchString(r.Tag) {
			continue
		}
		if result.previous == nil || r.PublishedAt.After(*result.previous.PublishedAt) || (r.PublishedAt.Equal(*result.previous.PublishedAt) && r.ID > result.previous.ID) {
			copy := r
			result.previous = &copy
		}
	}
	rangeSpec := source
	if result.previous != nil {
		resolved, err := git(directory, "rev-parse", "--verify", "refs/tags/"+result.previous.Tag+"^{commit}")
		if err != nil {
			return result, fmt.Errorf("resolve previous release %s: %w", result.previous.Tag, err)
		}
		result.previousSHA = strings.TrimSpace(resolved)
		if _, err := git(directory, "merge-base", "--is-ancestor", result.previousSHA, source); err != nil {
			return result, fmt.Errorf("previous published release is not an ancestor of source")
		}
		rangeSpec = result.previousSHA + ".." + source
	}
	log, err := git(directory, "log", "--reverse", "--topo-order", "--no-decorate", "--no-notes", "--no-color", "--encoding=UTF-8", "-z", "--format=%H%x00%B", rangeSpec, "--")
	if err != nil {
		return result, err
	}
	if log == "" {
		return result, nil
	}
	fields := strings.Split(strings.TrimSuffix(log, "\x00"), "\x00")
	if len(fields)%2 != 0 {
		return result, fmt.Errorf("invalid Git log record")
	}
	for i := 0; i < len(fields); i += 2 {
		if !sourcePattern.MatchString(fields[i]) {
			return result, fmt.Errorf("invalid Git log commit")
		}
		result.changes = append(result.changes, parseChange(fields[i], fields[i+1]))
	}
	return result, nil
}

func propose(directory, source string, reader io.Reader, allowReuse bool) (proposal, error) {
	catalog, err := readCatalog(reader)
	if err != nil {
		return proposal{}, err
	}
	if allowReuse {
		committed, err := git(directory, "show", source+":action/cli-version")
		if err != nil {
			return proposal{}, err
		}
		version := strings.TrimSpace(committed)
		matches := 0
		for _, r := range catalog {
			if r.Tag == version {
				matches++
			}
		}
		if matches > 1 {
			return proposal{}, fmt.Errorf("multiple releases use the committed version")
		}
		for _, r := range catalog {
			if r.Tag != version || *r.Prerelease || r.Body == nil {
				continue
			}
			line, _, _ := strings.Cut(*r.Body, "\n")
			if !strings.HasPrefix(line, "<!-- gh-settings-release: ") || !strings.HasSuffix(line, " -->") {
				continue
			}
			var identity struct {
				Version, Commit string
				Macos           string `json:"macos_sha256"`
				Linux           string `json:"linux_sha256"`
				Checksums       string `json:"checksums_sha256"`
			}
			data := strings.TrimSuffix(strings.TrimPrefix(line, "<!-- gh-settings-release: "), " -->")
			if json.Unmarshal([]byte(data), &identity) != nil || identity.Version != version || identity.Commit != source || !versionPattern.MatchString(version) || !hashPattern.MatchString(identity.Macos) || !hashPattern.MatchString(identity.Linux) || !hashPattern.MatchString(identity.Checksums) {
				continue
			}
			tag, err := git(directory, "rev-parse", "--verify", "refs/tags/"+version+"^{commit}")
			if err != nil || strings.TrimSpace(tag) != source {
				return proposal{}, fmt.Errorf("existing release tag does not match its source")
			}
			// The publisher still verifies every artifact and retains stored notes.
			return proposal{version: version, reuse: true}, nil
		}
	}
	interval, err := history(directory, source, "", catalog)
	if err != nil {
		return proposal{}, err
	}
	if interval.previous == nil {
		return proposal{}, fmt.Errorf("initial release requires a separately chosen baseline")
	}
	tier, count := 0, 0
	for _, c := range interval.changes {
		if (c.kind == "build" || c.kind == "chore") && c.scope == "release" && !c.major && !c.minor {
			paths, err := git(directory, "diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "-z", c.sha, "--")
			if err != nil {
				return proposal{}, err
			}
			if paths == "action/cli-version\x00" {
				continue
			}
		}
		count++
		if c.major {
			tier = 2
		} else if c.minor && tier < 1 {
			tier = 1
		}
	}
	if count == 0 {
		return proposal{}, fmt.Errorf("no changes other than version preparation since the previous release")
	}
	version, err := bump(interval.previous.Tag, tier)
	if err != nil {
		return proposal{}, err
	}
	return proposal{version: version, previousTag: interval.previous.Tag, previousSHA: interval.previousSHA}, nil
}

func bump(version string, tier int) (string, error) {
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid stable version")
	}
	var parts [3]uint64
	for i, part := range strings.Split(version[1:], ".") {
		v, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return "", fmt.Errorf("version component exceeds supported range")
		}
		parts[i] = v
	}
	// SemVer permits breaking changes before 1.0 without graduating to 1.0.
	// Keep that explicit milestone manual by treating the strongest signal as
	// a minor increment while the published baseline is still 0.x.
	if tier == 2 && parts[0] == 0 {
		tier = 1
	}
	index := 2 - tier
	if index < 0 || index > 2 || parts[index] == ^uint64(0) {
		return "", fmt.Errorf("invalid or overflowing version increment")
	}
	parts[index]++
	for i := index + 1; i < 3; i++ {
		parts[i] = 0
	}
	return fmt.Sprintf("v%d.%d.%d", parts[0], parts[1], parts[2]), nil
}

func generate(directory, repository, version, source string, reader io.Reader, expectedPrevious, expectedSHA string, gh ghAPI) (string, error) {
	if !repoPattern.MatchString(repository) || !versionPattern.MatchString(version) || !sourcePattern.MatchString(source) {
		return "", fmt.Errorf("invalid repository, version, or source commit")
	}
	catalog, err := readCatalog(reader)
	if err != nil {
		return "", err
	}
	interval, err := history(directory, source, version, catalog)
	if err != nil {
		return "", err
	}
	previous := ""
	if interval.previous != nil {
		previous = interval.previous.Tag
	}
	if (expectedPrevious != "" && previous != expectedPrevious) || (expectedSHA != "" && interval.previousSHA != expectedSHA) {
		return "", fmt.Errorf("published release boundary changed since version resolution")
	}
	if _, err := git(directory, "cat-file", "-e", source+":.github/release.yml"); err != nil {
		return "", fmt.Errorf("release source lacks .github/release.yml")
	}
	tag, err := git(directory, "tag", "--list", version)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(tag) != "" {
		resolved, err := git(directory, "rev-parse", "--verify", "refs/tags/"+version+"^{commit}")
		if err != nil || strings.TrimSpace(resolved) != source {
			return "", fmt.Errorf("existing tag targets another commit")
		}
	}
	args := []string{"api", "-X", "POST", "repos/" + repository + "/releases/generate-notes", "-f", "tag_name=" + version, "-f", "target_commitish=" + source, "-f", "configuration_file_path=.github/release.yml"}
	if previous != "" {
		args = append(args, "-f", "previous_tag_name="+previous)
	}
	data, err := gh(args...)
	if err != nil {
		return "", fmt.Errorf("GitHub release-note generation failed: %w", err)
	}
	var generated struct {
		Body *string `json:"body"`
	}
	if json.Unmarshal(data, &generated) != nil || generated.Body == nil || strings.TrimSpace(*generated.Body) == "" {
		return "", fmt.Errorf("GitHub returned no release-note body")
	}
	var notes strings.Builder
	notes.WriteString(strings.TrimRight(*generated.Body, "\n"))
	notes.WriteString("\n")
	migration := false
	for _, c := range interval.changes {
		if !c.major {
			continue
		}
		if !migration {
			notes.WriteString("\n## ⚠️ Migration guidance\n\n")
			migration = true
		}
		fmt.Fprintf(&notes, "- [%s](https://github.com/%s/commit/%s)\n", markdownText(c.subject), repository, c.sha)
		if len(c.breaking) == 0 {
			notes.WriteString("\n  > No migration footer was recorded; review the linked change before upgrading.\n")
		}
		for _, detail := range c.breaking {
			notes.WriteByte('\n')
			for _, line := range strings.Split(detail, "\n") {
				fmt.Fprintf(&notes, "  > %s\n", markdownText(line))
			}
		}
	}
	if previous != "" {
		fmt.Fprintf(&notes, "\n**Frozen comparison**: [%s → %s](https://github.com/%s/compare/%s...%s)\n", previous, version, repository, interval.previousSHA, source)
	}
	return notes.String(), nil
}

func parseChange(sha, message string) change {
	lines := strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n")
	c := change{sha: sha, subject: lines[0]}
	if m := headerPattern.FindStringSubmatch(lines[0]); m != nil {
		c.kind, c.scope, c.subject = strings.ToLower(m[1]), m[3], m[5]
		c.major = m[4] != ""
		c.minor = c.kind == "feat" || c.kind == "perf"
	}
	c.breaking, c.minor = footers(lines[1:], c.minor)
	c.major = c.major || len(c.breaking) > 0
	return c
}

// Extract compatibility footers, not example markers inside fenced code.
func footers(lines []string, minor bool) ([]string, bool) {
	var result, value []string
	inFooters, collecting, blank := false, false, false
	fence := ""
	flush := func() {
		if collecting {
			result = append(result, strings.TrimSpace(strings.Join(value, "\n")))
		}
		value = nil
	}
	for _, line := range lines {
		match := fencePattern.FindStringSubmatch(line)
		if fence != "" {
			if match != nil && match[1][0] == fence[0] && len(match[1]) >= len(fence) && strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(line, " "), match[1])) == "" {
				fence = ""
			}
		} else if match != nil {
			fence = match[1]
		} else if footer := footerPattern.FindStringSubmatch(line); footer != nil && (blank || inFooters) {
			flush()
			inFooters = true
			collecting = footer[2] == ": " && (footer[1] == "BREAKING CHANGE" || footer[1] == "BREAKING-CHANGE")
			if collecting {
				value = append(value, footer[3])
			}
			if footer[1] == "DEPRECATED" && footer[2] == ": " {
				minor = true
			}
			blank = false
			continue
		}
		if collecting {
			value = append(value, line)
		}
		blank = strings.TrimSpace(line) == ""
	}
	flush()
	return result, minor
}

func markdownText(text string) string {
	return strings.NewReplacer("\\", "\\\\", "&", "&amp;", "<", "&lt;", ">", "&gt;", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "#", "\\#", "~", "\\~", "|", "\\|").Replace(text)
}

func git(directory string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"--no-pager", "-c", "log.showSignature=false"}, args...)...)
	command.Dir = directory
	data, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w", args[0], err)
	}
	return string(data), nil
}

func ghCommand(args ...string) ([]byte, error) {
	data, err := exec.Command("gh", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("GitHub metadata request failed: %w", err)
	}
	return data, nil
}
