// Command release-metadata generates release notes from a frozen Git interval.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
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
	major, deprecated         bool
	breaking                  []string
}

const (
	breakingCategory = iota
	deprecatedCategory
	featureCategory
	fixCategory
	performanceCategory
	maintenanceCategory
)

var categoryHeadings = [...]string{
	"💥 BREAKING CHANGE",
	"⚠️ DEPRECATED",
	"✨ New Feature",
	"🐛 Bug Fix",
	"⚡ Performance",
	"🛠️ Maintenance",
}

type interval struct {
	previous    *release
	previousSHA string
	changes     []change
}

func main() {
	if err := run(".", os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "release-metadata:", err)
		os.Exit(1)
	}
}

func run(directory string, args []string, output io.Writer) error {
	if len(args) != 4 {
		return fmt.Errorf("usage: release-metadata OWNER/REPO VERSION SOURCE_SHA RELEASES_JSON")
	}
	catalog, err := os.Open(args[3])
	if err != nil {
		return err
	}
	defer catalog.Close()
	notes, err := generate(directory, args[0], args[1], args[2], catalog, os.Getenv("PREVIOUS_TAG"), os.Getenv("PREVIOUS_SHA"))
	if err != nil {
		return err
	}
	_, err = io.WriteString(output, notes)
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
	log, err := git(directory, "log", "--first-parent", "--reverse", "--topo-order", "--no-decorate", "--no-notes", "--no-color", "--encoding=UTF-8", "-z", "--format=%H%x00%P%x00%B", rangeSpec, "--")
	if err != nil {
		return result, err
	}
	if log == "" {
		return result, nil
	}
	fields := strings.Split(strings.TrimSuffix(log, "\x00"), "\x00")
	if len(fields)%3 != 0 {
		return result, fmt.Errorf("invalid Git log record")
	}
	for i := 0; i < len(fields); i += 3 {
		if !sourcePattern.MatchString(fields[i]) {
			return result, fmt.Errorf("invalid Git log commit")
		}
		// A main-line squash commit represents one change. Empty commits do not
		// change the product, even if their messages contain release markers.
		if parents := strings.Fields(fields[i+1]); len(parents) > 0 {
			diff, err := git(directory, "diff-tree", "--no-commit-id", "--name-only", "-r", parents[0], fields[i], "--")
			if err != nil {
				return result, err
			}
			if diff == "" {
				continue
			}
		}
		result.changes = append(result.changes, parseChange(fields[i], fields[i+2]))
	}
	return result, nil
}

func generate(directory, repository, version, source string, reader io.Reader, expectedPrevious, expectedSHA string) (string, error) {
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
	var notes strings.Builder
	notes.WriteString("## What's Changed\n")
	groups := make([][]change, len(categoryHeadings))
	for _, c := range interval.changes {
		category := c.category()
		groups[category] = append(groups[category], c)
	}
	for category, changes := range groups {
		if len(changes) == 0 {
			continue
		}
		fmt.Fprintf(&notes, "\n### %s\n\n", categoryHeadings[category])
		for _, c := range changes {
			fmt.Fprintf(&notes, "- [%s](https://github.com/%s/commit/%s)\n", markdownText(c.subject), repository, c.sha)
		}
	}
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

func (c change) category() int {
	switch {
	case c.major:
		return breakingCategory
	case c.deprecated:
		return deprecatedCategory
	case c.kind == "feat":
		return featureCategory
	case c.kind == "fix":
		return fixCategory
	case c.kind == "perf":
		return performanceCategory
	default:
		return maintenanceCategory
	}
}

func parseChange(sha, message string) change {
	lines := strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n")
	c := change{sha: sha, subject: lines[0]}
	if m := headerPattern.FindStringSubmatch(lines[0]); m != nil {
		c.kind, c.scope, c.subject = strings.ToLower(m[1]), m[3], m[5]
		c.major = m[4] != ""
	}
	c.breaking, c.deprecated = footers(lines[1:])
	c.major = c.major || len(c.breaking) > 0
	return c
}

// Extract compatibility footers, not example markers inside fenced code.
func footers(lines []string) ([]string, bool) {
	var result, value []string
	inFooters, collecting, blank := false, false, false
	deprecated := false
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
				deprecated = true
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
	return result, deprecated
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
