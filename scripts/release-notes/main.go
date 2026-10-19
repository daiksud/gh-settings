// Command release-notes renders the frozen release's Git history as Markdown.
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
	repoPattern    = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	headerPattern  = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*)(\(([^()]+)\))?(!)?: (.+)$`)
	footerPattern  = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*|BREAKING CHANGE)(: | #)(.*)$`)
	fencePattern   = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
)

type release struct {
	ID          int64      `json:"id"`
	Tag         string     `json:"tag_name"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	PublishedAt *time.Time `json:"published_at"`
}

type change struct {
	sha, subject, scope string
	group               int
	breaking            []string
}

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: release-notes OWNER/REPO VERSION SOURCE_SHA RELEASES_JSON")
		os.Exit(1)
	}
	catalog, err := os.Open(os.Args[4])
	if err != nil {
		fmt.Fprintln(os.Stderr, "release-notes:", err)
		os.Exit(1)
	}
	defer catalog.Close()
	notes, err := generate(".", os.Args[1], os.Args[2], os.Args[3], catalog)
	if err == nil {
		_, err = io.WriteString(os.Stdout, notes)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "release-notes:", err)
		os.Exit(1)
	}
}

func generate(directory, repository, version, source string, catalog io.Reader) (string, error) {
	if !repoPattern.MatchString(repository) || !versionPattern.MatchString(version) || !sourcePattern.MatchString(source) {
		return "", fmt.Errorf("invalid repository, version, or source commit")
	}
	var pages [][]release
	if err := json.NewDecoder(catalog).Decode(&pages); err != nil {
		return "", fmt.Errorf("read published releases: %w", err)
	}
	if len(pages) == 0 {
		return "", fmt.Errorf("missing paginated release list")
	}
	var previous *release
	for _, page := range pages {
		if page == nil {
			return "", fmt.Errorf("missing release list page")
		}
		for _, candidate := range page {
			if candidate.Tag == version || candidate.Draft || candidate.Prerelease || candidate.PublishedAt == nil || !versionPattern.MatchString(candidate.Tag) {
				continue
			}
			if previous == nil || candidate.PublishedAt.After(*previous.PublishedAt) ||
				(candidate.PublishedAt.Equal(*previous.PublishedAt) && candidate.ID > previous.ID) {
				copy := candidate
				previous = &copy
			}
		}
	}
	shallow, err := git(directory, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(shallow) != "false" {
		return "", fmt.Errorf("release notes require complete Git history, not a shallow checkout")
	}
	resolved, err := git(directory, "rev-parse", "--verify", source+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve source commit: %w", err)
	}
	if strings.TrimSpace(resolved) != source {
		return "", fmt.Errorf("source must identify a commit directly")
	}
	revisionRange, previousSHA := source, ""
	if previous != nil {
		resolved, err := git(directory, "rev-parse", "--verify", "refs/tags/"+previous.Tag+"^{commit}")
		if err != nil {
			return "", fmt.Errorf("resolve previous release %s: %w", previous.Tag, err)
		}
		previousSHA = strings.TrimSpace(resolved)
		if _, err := git(directory, "merge-base", "--is-ancestor", previousSHA, source); err != nil {
			return "", fmt.Errorf("previous release %s is not an ancestor of source %s: %w", previous.Tag, source, err)
		}
		revisionRange = previousSHA + ".." + source
	}
	log, err := git(directory, "log", "--reverse", "--topo-order", "--no-decorate", "--no-notes", "--no-color", "--encoding=UTF-8", "-z", "--format=%H%x00%B", revisionRange, "--")
	if err != nil {
		return "", fmt.Errorf("read release history: %w", err)
	}
	var groups [4][]change
	if log != "" {
		fields := strings.Split(strings.TrimSuffix(log, "\x00"), "\x00")
		if len(fields)%2 != 0 {
			return "", fmt.Errorf("invalid Git log record")
		}
		for i := 0; i < len(fields); i += 2 {
			if !sourcePattern.MatchString(fields[i]) {
				return "", fmt.Errorf("invalid Git log commit")
			}
			entry := parseChange(fields[i], fields[i+1])
			groups[entry.group] = append(groups[entry.group], entry)
		}
	}
	var notes strings.Builder
	url := "https://github.com/" + repository
	for index, heading := range []string{"🚨 BREAKING CHANGE", "✨ New Feature", "🐛 Bug Fix", "🧹 Chore"} {
		if len(groups[index]) == 0 {
			continue
		}
		fmt.Fprintf(&notes, "## %s\n\n", heading)
		for _, entry := range groups[index] {
			notes.WriteString("- ")
			if entry.scope != "" {
				fmt.Fprintf(&notes, "**%s**: ", markdownText(entry.scope))
			}
			fmt.Fprintf(&notes, "%s ([`%s`](%s/commit/%s))\n", markdownText(entry.subject), entry.sha[:7], url, entry.sha)
			for _, detail := range entry.breaking {
				notes.WriteByte('\n')
				for _, line := range strings.Split(detail, "\n") {
					fmt.Fprintf(&notes, "  > %s\n", markdownText(line))
				}
			}
		}
		notes.WriteByte('\n')
	}
	if log == "" {
		notes.WriteString("_No changes since the previous release._\n\n")
	}
	if previous != nil {
		fmt.Fprintf(&notes, "**Full Changelog**: [%s → %s](%s/compare/%s...%s)\n", previous.Tag, version, url, previousSHA, source)
	} else {
		fmt.Fprintf(&notes, "**Full Changelog**: [History up to %s](%s/commits/%s)\n", version, url, source)
	}
	return notes.String(), nil
}

func git(directory string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"--no-pager", "-c", "log.showSignature=false"}, args...)...)
	cmd.Dir = directory
	data, err := cmd.Output()
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s (%w)", args[0], strings.TrimSpace(string(failure.Stderr)), err)
		}
		return "", err
	}
	return string(data), nil
}

func parseChange(sha, message string) change {
	lines := strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n")
	entry := change{sha: sha, subject: lines[0], group: 3}
	if match := headerPattern.FindStringSubmatch(lines[0]); match != nil {
		entry.scope, entry.subject = match[3], match[5]
		switch strings.ToLower(match[1]) {
		case "feat", "perf":
			entry.group = 1
		case "fix":
			entry.group = 2
		}
		if match[4] != "" {
			entry.group = 0
		}
	}
	entry.breaking = breakingFooters(lines[1:])
	if len(entry.breaking) != 0 {
		entry.group = 0
	}
	return entry
}

// Footer values continue through paragraphs until the next footer token. Fenced
// examples in PR bodies are not footers, even when they show a breaking marker.
func breakingFooters(lines []string) []string {
	var result []string
	var value []string
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
			blank = false
			continue
		}
		if collecting {
			value = append(value, line)
		}
		blank = strings.TrimSpace(line) == ""
	}
	flush()
	return result
}

func markdownText(text string) string {
	return strings.NewReplacer(
		"\\", "\\\\", "&", "&amp;", "<", "&lt;", ">", "&gt;",
		"`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "#", "\\#", "~", "\\~", "|", "\\|",
	).Replace(text)
}
