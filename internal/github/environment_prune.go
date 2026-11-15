package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ReadEnvironmentPrune acquires only unlisted parents. Retained unsupported
// environments do not become deletion candidates or require child reads.
func (c *EnvironmentsClient) ReadEnvironmentPrune(repo repository.Repository, retained []string) (settings.EnvironmentPruneObservation, error) {
	if _, err := settings.NewEnvironmentDeclarations(environmentNameDeclarations(retained)); err != nil {
		return settings.EnvironmentPruneObservation{}, err
	}
	identity := &Client{rest: c.rest}
	target, err := identity.ReadArchive(repo)
	if err != nil {
		return settings.EnvironmentPruneObservation{}, err
	}
	entries, err := c.inventory(repo)
	if err != nil {
		return settings.EnvironmentPruneObservation{}, err
	}
	keep := map[string]bool{}
	for _, name := range retained {
		keep[settings.EnvironmentIdentity(name)] = true
	}
	candidates := []settings.EnvironmentPruneCandidate{}
	for _, entry := range entries {
		if keep[settings.EnvironmentIdentity(entry.Name)] {
			continue
		}
		candidate, err := c.readEnvironmentPruneCandidate(repo, target, entry)
		if err != nil {
			return settings.EnvironmentPruneObservation{}, fmt.Errorf("acquire Environment deletion candidate %q: %w", entry.Name, err)
		}
		candidates = append(candidates, candidate)
	}
	again, err := identity.ReadArchive(repo)
	if err != nil || again.ID != target.ID || again.Archived != target.Archived {
		return settings.EnvironmentPruneObservation{}, fmt.Errorf("repository identity or archive state changed during Environment prune acquisition")
	}
	return settings.NewEnvironmentPruneObservation(repo, target, candidates)
}

func environmentNameDeclarations(names []string) []settings.EnvironmentDeclaration {
	values := make([]settings.EnvironmentDeclaration, 0, len(names))
	for _, name := range names {
		values = append(values, settings.EnvironmentDeclaration{Name: name})
	}
	return values
}

func (c *EnvironmentsClient) readEnvironmentPruneCandidate(repo repository.Repository, target settings.ArchiveState, entry environmentInventoryEntry) (settings.EnvironmentPruneCandidate, error) {
	state, err := c.readOne(repo, entry.Name) // Includes explicit known-empty custom protections.
	if err != nil {
		return settings.EnvironmentPruneCandidate{}, err
	}
	if state.CustomRules == nil || len(state.CustomRules) != 0 {
		return settings.EnvironmentPruneCandidate{}, fmt.Errorf("Environment deletion requires known-empty custom protection rules")
	}
	if state.ID != entry.ID {
		return settings.EnvironmentPruneCandidate{}, fmt.Errorf("Environment identity changed during prune acquisition")
	}
	variables, err := (&Client{rest: c.rest}).readVariableInventory(environmentPath(repo, entry.Name) + "/variables")
	if err != nil {
		return settings.EnvironmentPruneCandidate{}, err
	}
	secrets, err := c.readEnvironmentSecretMetadata(repo, entry.Name)
	if err != nil {
		return settings.EnvironmentPruneCandidate{}, err
	}
	candidate, err := settings.NewEnvironmentPruneCandidate(repo, target, state, variables, secrets, state.CustomRules != nil && len(state.CustomRules) == 0)
	if err != nil {
		return settings.EnvironmentPruneCandidate{}, err
	}
	// Child reads are name-addressed. Close acquisition with the same parent and
	// explicit custom-protection gate, without inventing an ID lock.
	again, err := c.readOne(repo, entry.Name)
	if err != nil {
		return settings.EnvironmentPruneCandidate{}, err
	}
	confirmed, err := settings.NewEnvironmentPruneCandidate(repo, target, again, variables, secrets, again.CustomRules != nil && len(again.CustomRules) == 0)
	if err != nil || !candidate.Equal(confirmed) {
		return settings.EnvironmentPruneCandidate{}, fmt.Errorf("Environment identity or policy changed during child acquisition")
	}
	return candidate, nil
}

func (c *EnvironmentsClient) readEnvironmentSecretMetadata(repo repository.Repository, name string) ([]settings.EnvironmentSecretMetadata, error) {
	values := []settings.EnvironmentSecretMetadata{}
	expected := -1
	for page := 1; ; page++ {
		response, err := c.rest.Request(http.MethodGet, fmt.Sprintf("%s/secrets?per_page=100&page=%d", environmentPath(repo, name), page), nil)
		if err != nil {
			return nil, fmt.Errorf("read Environment secret metadata page %d failed", page)
		}
		const pageLimit = 1 << 20
		data, readErr := io.ReadAll(io.LimitReader(response.Body, pageLimit+1))
		response.Body.Close()
		links := response.Header.Values("Link")
		for _, link := range links {
			if strings.TrimSpace(link) == "" {
				return nil, fmt.Errorf("invalid Environment secret metadata pagination")
			}
		}
		next, linkErr := webhookHasNext(strings.Join(links, ","))
		var raw map[string]json.RawMessage
		if response.StatusCode != http.StatusOK || readErr != nil || linkErr != nil || len(data) > pageLimit || !validEnvironmentSecretJSONUnicode(data) || webhookJSON(data, &raw) != nil || raw == nil || environmentKnownFields(raw, "total_count", "secrets") != nil {
			return nil, fmt.Errorf("incomplete or invalid Environment secret metadata page %d", page)
		}
		var total int
		var entries []map[string]json.RawMessage
		if requiredEnvironmentJSON(raw, "total_count", &total) != nil || total < 0 || requiredEnvironmentJSON(raw, "secrets", &entries) != nil || entries == nil || len(entries) > 100 {
			return nil, fmt.Errorf("incomplete or invalid Environment secret metadata page %d", page)
		}
		if expected >= 0 && expected != total {
			return nil, fmt.Errorf("Environment secret metadata count changed during pagination")
		}
		expected = total
		for _, entry := range entries {
			var value settings.EnvironmentSecretMetadata
			if entry == nil || environmentKnownFields(entry, "name", "created_at", "updated_at") != nil || requiredEnvironmentJSON(entry, "name", &value.Name) != nil || requiredEnvironmentJSON(entry, "created_at", &value.CreatedAt) != nil || requiredEnvironmentJSON(entry, "updated_at", &value.UpdatedAt) != nil {
				return nil, fmt.Errorf("incomplete Environment secret metadata")
			}
			values = append(values, value)
		}
		if len(values) > expected {
			return nil, fmt.Errorf("Environment secret metadata exceeds total count")
		}
		if len(values) == expected {
			if next {
				return nil, fmt.Errorf("incomplete Environment secret metadata inventory")
			}
			return values, nil
		}
		if len(entries) != 100 || len(links) > 0 && !next {
			return nil, fmt.Errorf("incomplete Environment secret metadata inventory")
		}
	}
}

// encoding/json replaces malformed UTF-8 and unpaired UTF-16 escape values
// with U+FFFD, which cannot establish exact private snapshot evidence.
func validEnvironmentSecretJSONUnicode(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' {
			continue
		}
		if i+1 >= len(data) {
			return false
		}
		i++
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		value, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if value >= 0xdc00 && value <= 0xdfff {
			return false
		}
		if value >= 0xd800 && value <= 0xdbff {
			if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}

func (c *EnvironmentsClient) DeleteEnvironment(repo repository.Repository, candidate settings.EnvironmentPruneCandidate) error {
	pre := func(err error) error { return &environmentPreWriteFailure{err} }
	identity := &Client{rest: c.rest}
	target, err := identity.ReadArchive(repo)
	if err != nil {
		return pre(err)
	}
	if err := candidate.ValidateTarget(repo, target); err != nil {
		return pre(err)
	}
	entries, err := c.inventory(repo)
	if err != nil {
		return pre(err)
	}
	found := false
	for _, entry := range entries {
		if entry.ID != candidate.ID() && settings.EnvironmentIdentity(entry.Name) != settings.EnvironmentIdentity(candidate.Name()) {
			continue
		}
		if entry.ID != candidate.ID() || entry.Name != candidate.Name() {
			return pre(fmt.Errorf("Environment deletion identity changed"))
		}
		fresh, err := c.readEnvironmentPruneCandidate(repo, target, entry)
		if err != nil {
			return pre(err)
		}
		if !candidate.Equal(fresh) {
			return pre(fmt.Errorf("Environment deletion candidate state changed"))
		}
		found = true
	}
	if !found {
		return pre(fmt.Errorf("Environment deletion candidate disappeared"))
	}
	target, err = identity.ReadArchive(repo)
	if err != nil {
		return pre(err)
	}
	if err := candidate.ValidateTarget(repo, target); err != nil {
		return pre(err)
	}
	response, err := c.rest.Request(http.MethodDelete, environmentPath(repo, candidate.Name()), nil)
	if err != nil {
		return fmt.Errorf("delete Environment failed (write may have applied): %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected Environment delete status %d (write may have applied)", response.StatusCode)
	}
	after, err := identity.ReadArchive(repo)
	if err != nil || candidate.ValidateTarget(repo, after) != nil {
		return fmt.Errorf("verify Environment deletion target failed (write may have applied)")
	}
	remaining, err := c.inventory(repo)
	if err != nil {
		return fmt.Errorf("verify Environment deletion inventory (write may have applied): %w", err)
	}
	for _, entry := range remaining {
		if entry.ID == candidate.ID() || settings.EnvironmentIdentity(entry.Name) == settings.EnvironmentIdentity(candidate.Name()) {
			return fmt.Errorf("Environment deletion readback retained or recreated identity (write may have applied)")
		}
	}
	return nil
}
