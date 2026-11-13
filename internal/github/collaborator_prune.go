package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func (raw collaboratorRawInvitation) pruneEntry(row settings.CollaboratorState) (settings.CollaboratorPruneEntry, error) {
	result := settings.CollaboratorPruneEntry{State: row, CreatedAt: *raw.CreatedAt}
	if raw.Expired != nil {
		if json.Unmarshal(raw.Expired, &result.Expired) != nil || result.Expired == nil {
			return result, fmt.Errorf("invalid invitation expiry evidence")
		}
	}
	if raw.Inviter != nil && string(raw.Inviter) != "null" {
		var inviter collaboratorRawUser
		if json.Unmarshal(raw.Inviter, &inviter) != nil || inviter.ID == nil || inviter.Login == nil || inviter.Type == nil {
			return result, fmt.Errorf("incomplete invitation inviter evidence")
		}
		result.Inviter = &settings.CollaboratorInviter{ID: *inviter.ID, Login: *inviter.Login, Type: *inviter.Type}
	}
	return result, nil
}

// A negative endpoint capability is distinct from malformed positive evidence.
// Both probes use the same authenticated, redirect-rejecting client; no token
// prefix, environment actor or second credential determines the principal.
func (c *Client) collaboratorOperatorProbe(path string, out any) (bool, string, error) {
	response, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == 401 || httpErr.StatusCode == 403 || httpErr.StatusCode == 404) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("collaborator management path could not be observed")
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	var extra any
	if response.StatusCode != 200 || decoder.Decode(out) != nil || decoder.Decode(&extra) != io.EOF {
		return false, "", fmt.Errorf("malformed collaborator management path evidence")
	}
	return true, response.Header.Get("Link"), nil
}
func (c *Client) collaboratorOperator(target settings.CollaboratorTarget) (settings.CollaboratorOperator, error) {
	var raw collaboratorRawUser
	userOK, link, err := c.collaboratorOperatorProbe("user", &raw)
	if err != nil {
		return settings.CollaboratorOperator{}, err
	}
	var user settings.CollaboratorIdentity
	if userOK {
		user, err = raw.person()
		if err != nil || link != "" {
			return settings.CollaboratorOperator{}, fmt.Errorf("malformed authenticated user identity")
		}
	}
	installationOK, err := c.collaboratorInstallationTarget(target)
	if err != nil {
		return settings.CollaboratorOperator{}, err
	}
	if userOK == installationOK {
		return settings.CollaboratorOperator{}, fmt.Errorf("ambiguous or unavailable collaborator management path")
	}
	if userOK {
		if user.ID != target.OwnerID || !strings.EqualFold(user.Username, strings.Split(target.FullName, "/")[0]) {
			return settings.CollaboratorOperator{}, fmt.Errorf("collaborator prune requires the authenticated personal repository owner")
		}
		return settings.CollaboratorOperator{Kind: "owner", User: user}, nil
	}
	return settings.CollaboratorOperator{Kind: "installation"}, nil
}
func (c *Client) collaboratorInstallationTarget(target settings.CollaboratorTarget) (bool, error) {
	seen := map[int64]bool{}
	names := map[string]bool{}
	total := -1
	found := false
	for page := 1; ; page++ {
		var raw struct {
			Total        *int                        `json:"total_count"`
			Repositories []collaboratorRawRepository `json:"repositories"`
		}
		ok, link, err := c.collaboratorOperatorProbe(fmt.Sprintf("installation/repositories?per_page=100&page=%d", page), &raw)
		if err != nil {
			return false, err
		}
		if !ok {
			if page == 1 {
				return false, nil
			}
			return false, fmt.Errorf("incomplete installation repository inventory")
		}
		if raw.Total == nil || *raw.Total < 0 || raw.Repositories == nil || total >= 0 && total != *raw.Total {
			return false, fmt.Errorf("incomplete installation repository inventory")
		}
		total = *raw.Total
		for _, row := range raw.Repositories {
			if row.ID == nil || *row.ID <= 0 || row.FullName == nil || len(strings.Split(*row.FullName, "/")) != 2 || strings.HasSuffix(*row.FullName, "/") || row.Owner == nil || row.Owner.ID == nil || *row.Owner.ID <= 0 || row.Owner.Login == nil || row.Owner.Type == nil || !strings.EqualFold(*row.Owner.Login, strings.Split(*row.FullName, "/")[0]) || settings.ValidateCollaboratorUsername(*row.Owner.Login) != nil || (*row.Owner.Type != "User" && *row.Owner.Type != "Organization") || seen[*row.ID] || names[strings.ToLower(*row.FullName)] {
				return false, fmt.Errorf("invalid or duplicate installation repository identity")
			}
			seen[*row.ID] = true
			names[strings.ToLower(*row.FullName)] = true
			if *row.ID == target.ID || strings.EqualFold(*row.FullName, target.FullName) {
				if *row.ID != target.ID || *row.FullName != target.FullName || *row.Owner.ID != target.OwnerID || *row.Owner.Type != "User" || !strings.EqualFold(*row.Owner.Login, strings.Split(target.FullName, "/")[0]) || row.Private == nil || *row.Private || row.Archived == nil || *row.Archived != target.Archived {
					return false, fmt.Errorf("installation target identity changed")
				}
				found = true
			}
		}
		done, err := collaboratorPageComplete(len(raw.Repositories), link)
		if err != nil || len(seen) > total {
			return false, fmt.Errorf("incomplete installation repository pagination")
		}
		if done {
			if len(seen) != total || !found {
				return false, fmt.Errorf("installation target access was not completely established")
			}
			return true, nil
		}
	}
}
func (c *Client) readCollaboratorPruneState(repo repository.Repository, operator bool) (settings.CollaboratorPruneObservation, error) {
	var entries []settings.CollaboratorPruneEntry
	observed, err := c.readCollaborators(repo, &entries)
	if err != nil {
		return settings.CollaboratorPruneObservation{}, err
	}
	var principal settings.CollaboratorOperator
	if operator {
		target, _ := observed.Target()
		principal, err = c.collaboratorOperator(target)
		if err != nil {
			return settings.CollaboratorPruneObservation{}, err
		}
		after, err := c.collaboratorTarget(repo)
		if err != nil || after != target {
			return settings.CollaboratorPruneObservation{}, fmt.Errorf("collaborator target changed during operator acquisition")
		}
	}
	return settings.NewCollaboratorPruneObservation(observed, entries, principal)
}
func (c *Client) ReadCollaboratorPrune(repo repository.Repository, desired settings.CollaboratorDeclarations) (settings.CollaboratorPruneObservation, error) {
	// Acquire operator proof only if the complete retained union leaves candidates.
	var entries []settings.CollaboratorPruneEntry
	observed, err := c.readCollaborators(repo, &entries)
	if err != nil {
		return settings.CollaboratorPruneObservation{}, err
	}
	resolved := make([]settings.CollaboratorIdentity, 0, desired.Len())
	keep := map[int64]bool{}
	for _, d := range desired.Values() {
		person, err := c.ResolveCollaborator(d)
		if err != nil {
			return settings.CollaboratorPruneObservation{}, err
		}
		resolved = append(resolved, person)
		keep[person.ID] = true
	}
	rows, _ := observed.Values()
	needsOperator := false
	for _, row := range rows {
		needsOperator = needsOperator || !keep[row.Person.ID]
	}
	var operator settings.CollaboratorOperator
	if needsOperator {
		target, _ := observed.Target()
		operator, err = c.collaboratorOperator(target)
		if err != nil {
			return settings.CollaboratorPruneObservation{}, err
		}
	}
	initial, err := settings.NewCollaboratorPruneObservation(observed, entries, operator)
	if err != nil {
		return settings.CollaboratorPruneObservation{}, err
	}
	initial, err = initial.Prepare(desired, resolved)
	if err != nil {
		return settings.CollaboratorPruneObservation{}, err
	}
	// Resolve/read acquisition is non-atomic; reject drift before global preflight
	// authorizes unrelated writes as well as the later deletion stage.
	return c.RefreshCollaboratorPrune(repo, initial, nil)
}
func (c *Client) RefreshCollaboratorPrune(repo repository.Repository, initial settings.CollaboratorPruneObservation, reconciled *settings.CollaboratorWriteResult) (settings.CollaboratorPruneObservation, error) {
	fresh, err := c.readCollaboratorPruneState(repo, initial.NeedsOperator())
	if err != nil {
		return settings.CollaboratorPruneObservation{}, err
	}
	return initial.Refresh(fresh, reconciled)
}
func (c *Client) DeleteCollaborator(repo repository.Repository, initial settings.CollaboratorPruneObservation, deletion settings.CollaboratorDeletion) (settings.CollaboratorPruneObservation, settings.CollaboratorDeleteResult, error) {
	result := settings.CollaboratorDeleteResult{Deletion: deletion}
	pre := func(err error) (settings.CollaboratorPruneObservation, settings.CollaboratorDeleteResult, error) {
		return initial, result, &collaboratorPreWriteFailure{err}
	}
	fresh, err := c.RefreshCollaboratorPrune(repo, initial, nil)
	if err != nil {
		return pre(err)
	}
	if err := fresh.BeforeDelete(deletion); err != nil {
		return pre(err)
	}
	path := "repos/" + repo.Owner + "/" + repo.Name
	if deletion.InvitationID > 0 {
		path += fmt.Sprintf("/invitations/%d", deletion.InvitationID)
	} else {
		// Both immutable-ID and last username reads precede the name-addressed DELETE.
		id := deletion.Person.ID
		person, err := c.ResolveCollaborator(settings.CollaboratorDeclaration{Username: deletion.Person.Username, ID: &id})
		if err != nil || person != deletion.Person {
			return pre(fmt.Errorf("collaborator deletion user identity changed"))
		}
		person, err = c.ResolveCollaborator(settings.CollaboratorDeclaration{Username: deletion.Person.Username})
		if err != nil || person != deletion.Person {
			return pre(fmt.Errorf("collaborator deletion username changed identity"))
		}
		path += "/collaborators/" + deletion.Person.Username
	}
	result.Attempted = true
	response, err := c.rest.Request(http.MethodDelete, path, nil)
	if err != nil {
		return initial, result, fmt.Errorf("collaborator deletion request failed; effects are unknown")
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 1))
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent || readErr != nil || len(body) != 0 {
		return initial, result, fmt.Errorf("unexpected collaborator deletion acknowledgment; effects are unknown")
	}
	result.Acknowledged = true
	after, err := c.readCollaboratorPruneState(repo, true)
	if err != nil {
		return initial, result, fmt.Errorf("read back collaborator deletion failed; deletion was acknowledged but access is unverified")
	}
	verified, err := fresh.AfterDelete(after, deletion)
	if err != nil {
		return initial, result, fmt.Errorf("read back collaborator deletion did not confirm fixed remaining access")
	}
	result.Verified = true
	return verified, result, nil
}
