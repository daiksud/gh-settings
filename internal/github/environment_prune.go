package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
		var raw struct {
			Total   *int `json:"total_count"`
			Secrets *[]struct {
				Name      *string `json:"name"`
				CreatedAt *string `json:"created_at"`
				UpdatedAt *string `json:"updated_at"`
			} `json:"secrets"`
		}
		decoder := json.NewDecoder(response.Body)
		decoder.DisallowUnknownFields()
		decodeErr := decoder.Decode(&raw)
		var extra any
		trailing := decoder.Decode(&extra)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || decodeErr != nil || trailing != io.EOF || raw.Total == nil || *raw.Total < 0 || raw.Secrets == nil || *raw.Secrets == nil {
			return nil, fmt.Errorf("incomplete or invalid Environment secret metadata page %d", page)
		}
		if expected >= 0 && expected != *raw.Total {
			return nil, fmt.Errorf("Environment secret metadata count changed during pagination")
		}
		expected = *raw.Total
		for _, entry := range *raw.Secrets {
			if entry.Name == nil || entry.CreatedAt == nil || entry.UpdatedAt == nil {
				return nil, fmt.Errorf("incomplete Environment secret metadata")
			}
			values = append(values, settings.EnvironmentSecretMetadata{Name: *entry.Name, CreatedAt: *entry.CreatedAt, UpdatedAt: *entry.UpdatedAt})
		}
		if len(values) > expected {
			return nil, fmt.Errorf("Environment secret metadata exceeds total count")
		}
		if len(values) == expected {
			return values, nil
		}
		if len(*raw.Secrets) != 100 {
			return nil, fmt.Errorf("incomplete Environment secret metadata inventory")
		}
	}
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
