package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func pullRequestExemptionsPath(repo repository.Repository) string {
	return fmt.Sprintf("repos/%s/%s/interaction-limits/pulls/bypass-list", repo.Owner, repo.Name)
}

func (c *PullRequestLimitsClient) ReadExemptions(repo repository.Repository) (settings.PullRequestExemptionsObservation, error) {
	response, err := c.rest.Request(http.MethodGet, pullRequestExemptionsPath(repo), nil)
	if err != nil {
		return settings.PullRequestExemptionsObservation{}, fmt.Errorf("read pull request exemptions failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return settings.PullRequestExemptionsObservation{}, fmt.Errorf("unexpected pull request exemptions read status %d", response.StatusCode)
	}
	// The endpoint documents a complete array, not pagination parameters. Never
	// mistake an unexpected paginated response for the authoritative membership.
	if response.Header.Get("Link") != "" {
		return settings.PullRequestExemptionsObservation{}, fmt.Errorf("paginated pull request exemptions response is unsupported")
	}
	var raw []struct {
		ID    *int64  `json:"id"`
		Login *string `json:"login"`
		Type  *string `json:"type"`
	}
	decoder := json.NewDecoder(response.Body)
	if decoder.Decode(&raw) != nil || raw == nil {
		return settings.PullRequestExemptionsObservation{}, fmt.Errorf("incomplete or invalid pull request exemptions")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return settings.PullRequestExemptionsObservation{}, fmt.Errorf("invalid trailing pull request exemptions response")
	}
	users := make([]settings.PullRequestExemptionUser, len(raw))
	for i, user := range raw {
		if user.ID == nil || user.Login == nil || user.Type == nil {
			return settings.PullRequestExemptionsObservation{}, fmt.Errorf("incomplete pull request exemption user")
		}
		users[i] = settings.PullRequestExemptionUser{ID: *user.ID, Login: *user.Login, Type: *user.Type}
	}
	return settings.NewPullRequestExemptionsObservation(users)
}

// resolveExemptionUser establishes the intended account before removing existing
// members. Desired logins have already passed the strict ASCII login validator.
func (c *PullRequestLimitsClient) resolveExemptionUser(login string) (settings.PullRequestExemptionUser, error) {
	response, err := c.rest.Request(http.MethodGet, "users/"+login, nil)
	if err != nil {
		return settings.PullRequestExemptionUser{}, fmt.Errorf("resolve pull request exemption User failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Link") != "" {
		return settings.PullRequestExemptionUser{}, fmt.Errorf("invalid pull request exemption User response status or pagination")
	}
	var raw struct {
		ID    *int64  `json:"id"`
		Login *string `json:"login"`
		Type  *string `json:"type"`
	}
	decoder := json.NewDecoder(response.Body)
	if decoder.Decode(&raw) != nil || raw.ID == nil || raw.Login == nil || raw.Type == nil {
		return settings.PullRequestExemptionUser{}, fmt.Errorf("incomplete pull request exemption User identity")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return settings.PullRequestExemptionUser{}, fmt.Errorf("invalid trailing pull request exemption User response")
	}
	observation, err := settings.NewPullRequestExemptionsObservation([]settings.PullRequestExemptionUser{{ID: *raw.ID, Login: *raw.Login, Type: *raw.Type}})
	if err != nil {
		return settings.PullRequestExemptionUser{}, err
	}
	users, _ := observation.Users()
	if users[0].Login != login {
		return settings.PullRequestExemptionUser{}, fmt.Errorf("resolved pull request exemption User login differs from requested login")
	}
	return users[0], nil
}

func (c *PullRequestLimitsClient) writeExemptions(repo repository.Repository, method string, users []string) error {
	body, _ := json.Marshal(struct {
		Users []string `json:"users"`
	}{users})
	response, err := c.rest.Request(method, pullRequestExemptionsPath(repo), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("write pull request exemptions failed (write effects may already have applied)")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected pull request exemptions write status %d (write effects may already have applied)", response.StatusCode)
	}
	return nil
}

func verifyExemptionMembership(observed settings.PullRequestExemptionsObservation, expected []string, retained map[string]int64) error {
	logins, err := observed.Logins()
	if err != nil {
		return err
	}
	if !slices.Equal(logins, expected) {
		return fmt.Errorf("pull request exemptions membership differs from requested state")
	}
	users, err := observed.Users()
	if err != nil {
		return err
	}
	for _, user := range users {
		if id, exists := retained[user.Login]; exists && id != user.ID {
			return fmt.Errorf("retained pull request exemption identity changed")
		}
	}
	return nil
}

// UpdateExemptions owns the complete selected membership, but never the cap.
// Removals precede additions so replacing a full list does not exceed its cap.
// A true result with an error records verified removals before a later failure;
// the error still describes potentially applied effects from a subsequent write.
func (c *PullRequestLimitsClient) UpdateExemptions(repo repository.Repository, desired settings.PullRequestExemptionsDeclaration) (bool, error) {
	validated, err := settings.NewPullRequestExemptionsDeclaration(desired.Logins())
	if err != nil {
		return false, &pullRequestLimitsPreWriteFailure{err}
	}
	wanted := validated.Logins()
	observed, err := c.ReadExemptions(repo)
	if err != nil {
		return false, &pullRequestLimitsPreWriteFailure{err}
	}
	before, _ := observed.Logins()
	if slices.Equal(before, wanted) {
		return false, nil
	}
	users, _ := observed.Users()
	retained := map[string]int64{}
	var removals, additions []string
	intermediate := make([]string, 0, len(before))
	for _, user := range users {
		if slices.Contains(wanted, user.Login) {
			retained[user.Login] = user.ID
			intermediate = append(intermediate, user.Login)
		} else {
			removals = append(removals, user.Login)
		}
	}
	slices.Sort(intermediate)
	slices.Sort(removals)
	for _, login := range wanted {
		if !slices.Contains(before, login) {
			additions = append(additions, login)
		}
	}
	// Resolve every addition before the first write. Retain these identities for
	// final readback, as a reused login must not silently select another account.
	finalIdentities := make(map[string]int64, len(wanted))
	seenIDs := make(map[int64]bool, len(wanted))
	for login, id := range retained {
		finalIdentities[login] = id
		seenIDs[id] = true
	}
	for _, login := range additions {
		user, err := c.resolveExemptionUser(login)
		if err != nil {
			return false, &pullRequestLimitsPreWriteFailure{err}
		}
		if seenIDs[user.ID] {
			return false, &pullRequestLimitsPreWriteFailure{fmt.Errorf("ambiguous resolved pull request exemption User identity")}
		}
		seenIDs[user.ID] = true
		finalIdentities[login] = user.ID
	}
	confirmed := false
	if len(removals) > 0 {
		if err := c.writeExemptions(repo, http.MethodDelete, removals); err != nil {
			return false, err
		}
		observed, err = c.ReadExemptions(repo)
		if err != nil {
			return false, fmt.Errorf("verify removed pull request exemptions failed (write effects may already have applied): %w", err)
		}
		if err := verifyExemptionMembership(observed, intermediate, retained); err != nil {
			return false, fmt.Errorf("verify removed pull request exemptions failed (write effects may already have applied): %w", err)
		}
		confirmed = true
	}
	if len(additions) > 0 {
		if err := c.writeExemptions(repo, http.MethodPut, additions); err != nil {
			if confirmed {
				return true, fmt.Errorf("earlier exemption removals remain applied: %w", err)
			}
			return false, err
		}
		observed, err = c.ReadExemptions(repo)
		if err == nil {
			err = verifyExemptionMembership(observed, wanted, finalIdentities)
		}
		if err != nil {
			if confirmed {
				return true, fmt.Errorf("earlier exemption removals remain applied; verify added pull request exemptions failed (write effects may already have applied): %w", err)
			}
			return false, fmt.Errorf("verify added pull request exemptions failed (write effects may already have applied): %w", err)
		}
	}
	return true, nil
}
