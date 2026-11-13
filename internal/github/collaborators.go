package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// All collaborator requests reject redirects, including export and identity reads.
func NewCollaboratorsClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, LogIgnoreEnv: true, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &collaboratorPreWriteFailure{fmt.Errorf("configure collaborator client: %w", err)}
	}
	return &Client{rest: rest}, nil
}

// Errors are bounded: invitation payloads can contain private email addresses.
func (c *Client) collaboratorGET(path string, out any) (string, error) {
	response, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("read collaborator state failed")
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	var extra any
	if response.StatusCode != http.StatusOK || decoder.Decode(out) != nil || decoder.Decode(&extra) != io.EOF {
		return "", fmt.Errorf("incomplete or malformed collaborator response")
	}
	return response.Header.Get("Link"), nil
}

type collaboratorRawUser struct {
	ID    *int64  `json:"id"`
	Login *string `json:"login"`
	Type  *string `json:"type"`
}

func (raw collaboratorRawUser) person() (settings.CollaboratorIdentity, error) {
	if raw.ID == nil || raw.Login == nil || raw.Type == nil || *raw.Type != "User" {
		return settings.CollaboratorIdentity{}, fmt.Errorf("unsupported or incomplete collaborator User identity")
	}
	person := settings.CollaboratorIdentity{ID: *raw.ID, Username: *raw.Login}
	return person, person.Validate()
}

type collaboratorRawRepository struct {
	ID       *int64               `json:"id"`
	FullName *string              `json:"full_name"`
	Private  *bool                `json:"private"`
	Archived *bool                `json:"archived"`
	Owner    *collaboratorRawUser `json:"owner"`
}

func (c *Client) collaboratorTarget(repo repository.Repository) (settings.CollaboratorTarget, error) {
	var raw collaboratorRawRepository
	link, err := c.collaboratorGET("repos/"+repo.Owner+"/"+repo.Name, &raw)
	if err != nil {
		return settings.CollaboratorTarget{}, err
	}
	if link != "" || raw.ID == nil || *raw.ID <= 0 || raw.FullName == nil || !strings.EqualFold(*raw.FullName, repo.Owner+"/"+repo.Name) || raw.Private == nil || *raw.Private || raw.Archived == nil || raw.Owner == nil {
		return settings.CollaboratorTarget{}, fmt.Errorf("collaborators require a complete personally owned public repository identity")
	}
	owner, err := raw.Owner.person()
	if err != nil || !strings.EqualFold(owner.Username, repo.Owner) {
		return settings.CollaboratorTarget{}, fmt.Errorf("unsupported or mismatching personal repository owner")
	}
	return settings.CollaboratorTarget{ID: *raw.ID, OwnerID: owner.ID, FullName: *raw.FullName, Archived: *raw.Archived}, nil
}

type collaboratorRawActive struct {
	collaboratorRawUser
	Role        *string         `json:"role_name"`
	Permissions json.RawMessage `json:"permissions"`
}

func (raw collaboratorRawActive) validateRole() error {
	if raw.Role == nil || *raw.Role != "write" {
		return fmt.Errorf("unsupported personal collaborator role")
	}
	if raw.Permissions != nil {
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw.Permissions, &fields) != nil || fields == nil {
			return fmt.Errorf("malformed collaborator permissions")
		}
		for _, key := range []string{"pull", "push", "admin"} {
			if _, ok := fields[key]; !ok {
				return fmt.Errorf("incomplete collaborator permissions")
			}
		}
		for key, value := range fields {
			var actual *bool
			if json.Unmarshal(value, &actual) != nil || actual == nil {
				return fmt.Errorf("malformed collaborator permission")
			}
			want := key == "pull" || key == "push" || key == "triage"
			if (key != "pull" && key != "push" && key != "triage" && key != "maintain" && key != "admin") || *actual != want {
				return fmt.Errorf("unsupported collaborator permissions")
			}
		}
	}
	return nil
}

type collaboratorRawInvitation struct {
	ID         *int64 `json:"id"`
	Repository *struct {
		ID       *int64  `json:"id"`
		FullName *string `json:"full_name"`
	} `json:"repository"`
	Invitee     *collaboratorRawUser `json:"invitee"`
	Permissions *string              `json:"permissions"`
	CreatedAt   *string              `json:"created_at"`
	Expired     json.RawMessage      `json:"expired"`
}

func (raw collaboratorRawInvitation) state(target settings.CollaboratorTarget) (settings.CollaboratorState, error) {
	if raw.ID == nil || *raw.ID <= 0 || raw.Repository == nil || raw.Repository.ID == nil || *raw.Repository.ID != target.ID || raw.Repository.FullName == nil || !strings.EqualFold(*raw.Repository.FullName, target.FullName) || raw.Invitee == nil || raw.Permissions == nil || *raw.Permissions != "write" || raw.CreatedAt == nil {
		return settings.CollaboratorState{}, fmt.Errorf("unsupported or incomplete personal repository invitation")
	}
	if _, err := time.Parse(time.RFC3339, *raw.CreatedAt); err != nil {
		return settings.CollaboratorState{}, fmt.Errorf("malformed invitation creation timestamp")
	}
	if raw.Expired != nil {
		var expired *bool
		if json.Unmarshal(raw.Expired, &expired) != nil || expired == nil || *expired {
			return settings.CollaboratorState{}, fmt.Errorf("expired or malformed invitation requires explicit resolution")
		}
	}
	person, err := raw.Invitee.person()
	if err != nil {
		return settings.CollaboratorState{}, err
	}
	return settings.CollaboratorState{Person: person, Status: "pending", InvitationID: *raw.ID}, nil
}

func collaboratorPageComplete(length int, link string) (bool, error) {
	if length > 100 || length < 100 && strings.Contains(link, `rel="next"`) {
		return false, fmt.Errorf("incomplete collaborator pagination")
	}
	return length < 100, nil
}
func (c *Client) ReadCollaborators(repo repository.Repository) (settings.CollaboratorObservation, error) {
	target, err := c.collaboratorTarget(repo)
	if err != nil {
		return settings.CollaboratorObservation{}, err
	}
	rows := []settings.CollaboratorState{}
	seenIDs, seenNames, seenInvitations := map[int64]bool{}, map[string]bool{}, map[int64]bool{}
	appendRow := func(row settings.CollaboratorState) error {
		key := strings.ToLower(row.Person.Username)
		if seenIDs[row.Person.ID] || seenNames[key] || row.InvitationID > 0 && seenInvitations[row.InvitationID] {
			return fmt.Errorf("duplicate or overlapping collaborator identity during pagination")
		}
		seenIDs[row.Person.ID], seenNames[key] = true, true
		if row.InvitationID > 0 {
			seenInvitations[row.InvitationID] = true
		}
		rows = append(rows, row)
		return nil
	}
	path := "repos/" + repo.Owner + "/" + repo.Name
	ownerSeen := false
	for page := 1; ; page++ {
		var raw []collaboratorRawActive
		link, err := c.collaboratorGET(fmt.Sprintf("%s/collaborators?affiliation=all&per_page=100&page=%d", path, page), &raw)
		if err != nil || raw == nil {
			return settings.CollaboratorObservation{}, fmt.Errorf("active collaborator page %d unavailable or incomplete", page)
		}
		for _, row := range raw {
			person, err := row.person()
			if err != nil {
				return settings.CollaboratorObservation{}, err
			}
			if person.ID == target.OwnerID {
				if ownerSeen || !strings.EqualFold(person.Username, repo.Owner) {
					return settings.CollaboratorObservation{}, fmt.Errorf("ambiguous repository owner in collaborators")
				}
				ownerSeen = true
				continue
			}
			if err := row.validateRole(); err != nil {
				return settings.CollaboratorObservation{}, err
			}
			if err := appendRow(settings.CollaboratorState{Person: person, Status: "active"}); err != nil {
				return settings.CollaboratorObservation{}, err
			}
		}
		done, err := collaboratorPageComplete(len(raw), link)
		if err != nil {
			return settings.CollaboratorObservation{}, err
		}
		if done {
			break
		}
	}
	for page := 1; ; page++ {
		var raw []collaboratorRawInvitation
		link, err := c.collaboratorGET(fmt.Sprintf("%s/invitations?per_page=100&page=%d", path, page), &raw)
		if err != nil || raw == nil {
			return settings.CollaboratorObservation{}, fmt.Errorf("pending invitation page %d unavailable or incomplete", page)
		}
		for _, invitation := range raw {
			row, err := invitation.state(target)
			if err != nil {
				return settings.CollaboratorObservation{}, err
			}
			if err := appendRow(row); err != nil {
				return settings.CollaboratorObservation{}, err
			}
		}
		done, err := collaboratorPageComplete(len(raw), link)
		if err != nil {
			return settings.CollaboratorObservation{}, err
		}
		if done {
			break
		}
	}
	// Collection reads are not atomic; reject a target changed during acquisition.
	after, err := c.collaboratorTarget(repo)
	if err != nil || after != target {
		return settings.CollaboratorObservation{}, fmt.Errorf("collaborator repository identity changed during acquisition")
	}
	return settings.NewCollaboratorObservation(target, rows)
}

func (c *Client) ResolveCollaborator(desired settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error) {
	if err := settings.ValidateCollaboratorUsername(desired.Username); err != nil {
		return settings.CollaboratorIdentity{}, err
	}
	path := "users/" + desired.Username
	if desired.ID != nil {
		if *desired.ID <= 0 {
			return settings.CollaboratorIdentity{}, fmt.Errorf("invalid collaborator user id")
		}
		path = fmt.Sprintf("user/%d", *desired.ID)
	}
	var raw collaboratorRawUser
	link, err := c.collaboratorGET(path, &raw)
	if err != nil {
		return settings.CollaboratorIdentity{}, err
	}
	if link != "" {
		return settings.CollaboratorIdentity{}, fmt.Errorf("unexpected collaborator User pagination")
	}
	person, err := raw.person()
	if err != nil {
		return settings.CollaboratorIdentity{}, err
	}
	if desired.ID != nil && person.ID != *desired.ID {
		return settings.CollaboratorIdentity{}, fmt.Errorf("collaborator User ID changed")
	}
	if !strings.EqualFold(person.Username, desired.Username) {
		return settings.CollaboratorIdentity{}, fmt.Errorf("user id %d now uses username %q; update the declaration deliberately", person.ID, person.Username)
	}
	return person, nil
}

type collaboratorPreWriteFailure struct{ error }

func (*collaboratorPreWriteFailure) ReadFailedBeforeWrite() {}
func (e *collaboratorPreWriteFailure) Unwrap() error        { return e.error }

func (c *Client) ReconcileCollaborator(repo repository.Repository, selected settings.CollaboratorSelection) (settings.CollaboratorWriteResult, error) {
	result := settings.CollaboratorWriteResult{Person: selected.Person}
	pre := func(err error) (settings.CollaboratorWriteResult, error) {
		return result, &collaboratorPreWriteFailure{err}
	}
	if err := selected.Validate(); err != nil {
		return pre(err)
	}
	fresh, err := c.ReadCollaborators(repo)
	if err != nil {
		return pre(err)
	}
	target, _ := fresh.Target()
	if target != selected.Target {
		return pre(fmt.Errorf("collaborator target changed before reconciliation"))
	}
	// Resolve the username again even for ID-bound declarations: PUT has no
	// atomic expected-user-ID condition, so its residual rename race is explicit.
	person, err := c.ResolveCollaborator(settings.CollaboratorDeclaration{Username: selected.Person.Username})
	if err != nil {
		return pre(err)
	}
	if person != selected.Person && !(person.ID == selected.Person.ID && strings.EqualFold(person.Username, selected.Person.Username)) {
		return pre(fmt.Errorf("collaborator username changed identity before reconciliation"))
	}
	rows, _ := fresh.Values()
	for _, row := range rows {
		if strings.EqualFold(row.Person.Username, person.Username) && row.Person.ID != person.ID {
			return pre(fmt.Errorf("collaborator identity collision"))
		}
		if row.Person.ID == person.ID {
			if !strings.EqualFold(row.Person.Username, person.Username) {
				return pre(fmt.Errorf("collaborator username changed during acquisition"))
			}
			if selected.Status == "active" && row.Status != "active" || selected.Status == "pending" && row.Status == "pending" && row.InvitationID != selected.InvitationID {
				return pre(fmt.Errorf("selected collaborator access changed; refusing invitation fallback"))
			}
			result.Person, result.Status, result.InvitationID = person, row.Status, row.InvitationID
			return result, nil
		}
	}
	if selected.Status != "absent" {
		return pre(fmt.Errorf("selected collaborator access disappeared; refusing to reinvite"))
	}
	if target.Archived {
		return pre(fmt.Errorf("unarchive the repository explicitly before inviting collaborators"))
	}
	result.Attempted = true
	response, err := c.rest.Request(http.MethodPut, "repos/"+repo.Owner+"/"+repo.Name+"/collaborators/"+person.Username, nil)
	if err != nil {
		return result, fmt.Errorf("collaborator invitation request failed; it may have notified the user; resolve access before retrying")
	}
	status := response.StatusCode
	decoder := json.NewDecoder(response.Body)
	var extra any
	if status == http.StatusCreated {
		var raw collaboratorRawInvitation
		decodeErr := decoder.Decode(&raw)
		if raw.ID != nil && *raw.ID > 0 {
			result.InvitationID = *raw.ID
		}
		row, verifyErr := raw.state(target)
		if decodeErr != nil || verifyErr != nil || row.Person.ID != person.ID || !strings.EqualFold(row.Person.Username, person.Username) || decoder.Decode(&extra) != io.EOF {
			response.Body.Close()
			return result, fmt.Errorf("invitation acknowledgment is incomplete or mismatched; resolve access before retrying")
		}
		result.Invited = true
	} else if status != http.StatusNoContent || decoder.Decode(&extra) != io.EOF {
		response.Body.Close()
		return result, fmt.Errorf("unexpected collaborator invitation response; resolve access before retrying")
	}
	response.Body.Close()
	after, err := c.ReadCollaborators(repo)
	if err != nil {
		return result, fmt.Errorf("read back collaborator access failed (invitation may have been sent); resolve access before retrying")
	}
	afterTarget, _ := after.Target()
	if afterTarget != target {
		return result, fmt.Errorf("collaborator target changed after invitation request")
	}
	rows, _ = after.Values()
	for _, row := range rows {
		if row.Person.ID == person.ID && strings.EqualFold(row.Person.Username, person.Username) {
			if row.Status == "active" || status == http.StatusCreated && row.Status == "pending" && row.InvitationID == result.InvitationID {
				result.Status = row.Status
				return result, nil
			}
		}
	}
	return result, fmt.Errorf("read back did not confirm expected collaborator or invitation (request may have applied); resolve access before retrying")
}
