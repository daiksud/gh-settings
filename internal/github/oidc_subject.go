package github

import (
	"bytes"
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

type OIDCSubjectClient struct{ rest *api.RESTClient }

func NewOIDCSubjectClient(host string) (*OIDCSubjectClient, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, LogIgnoreEnv: true, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure OIDC client: %w", err)}
	}
	return &OIDCSubjectClient{rest}, nil
}
func oidcSubjectPath(repo repository.Repository) string {
	return securityRepositoryPath(repo) + "/actions/oidc/customization/sub"
}

type oidcSubjectState struct {
	id, ownerID int64
	archived    bool
	value       settings.OIDCSubjectTemplate
	immutable   *bool
}

const oidcResponseLimit = 1 << 20

func readOIDCBody(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, oidcResponseLimit+1))
	if err != nil || len(b) > oidcResponseLimit {
		return nil, fmt.Errorf("OIDC response unavailable or oversized")
	}
	return b, nil
}
func (c *OIDCSubjectClient) get(path string, out any) error {
	r, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("OIDC observation unavailable")
	}
	defer r.Body.Close()
	b, err := readOIDCBody(r.Body)
	if err != nil || r.StatusCode != http.StatusOK || json.Unmarshal(b, out) != nil {
		return fmt.Errorf("OIDC observation incomplete or unavailable")
	}
	return nil
}
func decodeOIDCSubject(raw map[string]json.RawMessage) (settings.OIDCSubjectTemplate, *bool, error) {
	var v settings.OIDCSubjectTemplate
	var immutable *bool
	for k, b := range raw {
		if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
			return v, nil, fmt.Errorf("OIDC observation contains null")
		}
		switch k {
		case "use_default":
			if json.Unmarshal(b, &v.UseDefaultTemplate) != nil {
				return v, nil, fmt.Errorf("invalid OIDC mode")
			}
		case "include_claim_keys":
			var keys []string
			// A null element must not collapse into an empty string, which is invalid too.
			if json.Unmarshal(b, &keys) != nil {
				return v, nil, fmt.Errorf("invalid OIDC claim keys")
			}
			v.SubjectClaimTemplate = &keys
		case "use_immutable_subject":
			if json.Unmarshal(b, &immutable) != nil {
				return v, nil, fmt.Errorf("invalid OIDC immutable companion")
			}
		case "sub_claim_prefix":
			var prefix string
			if json.Unmarshal(b, &prefix) != nil {
				return v, nil, fmt.Errorf("invalid OIDC prefix")
			}
		default:
			return v, nil, fmt.Errorf("unsupported OIDC response member")
		}
	}
	if v.UseDefaultTemplate == nil {
		return v, nil, fmt.Errorf("missing OIDC mode")
	}
	// Validate any returned keys before discarding inactive evidence. Default does
	// not promise hidden-template storage, but malformed present data is unavailable.
	if v.SubjectClaimTemplate != nil {
		custom := false
		probe := settings.OIDCSubjectTemplate{UseDefaultTemplate: &custom, SubjectClaimTemplate: v.SubjectClaimTemplate}
		if err := settings.ValidateOIDCSubject(probe); err != nil {
			return v, nil, err
		}
	}
	if *v.UseDefaultTemplate {
		v.SubjectClaimTemplate = nil
	}
	if err := settings.ValidateOIDCSubject(v); err != nil {
		return v, nil, err
	}
	return v, immutable, nil
}
func (c *OIDCSubjectClient) read(repo repository.Repository) (oidcSubjectState, error) {
	var t struct {
		ID         *int64  `json:"id"`
		FullName   *string `json:"full_name"`
		Visibility *string `json:"visibility"`
		Archived   *bool   `json:"archived"`
		Owner      *struct {
			ID   *int64  `json:"id"`
			Type *string `json:"type"`
		} `json:"owner"`
	}
	if err := c.get(securityRepositoryPath(repo), &t); err != nil {
		return oidcSubjectState{}, err
	}
	if t.ID == nil || *t.ID <= 0 || t.FullName == nil || !strings.EqualFold(*t.FullName, repo.Owner+"/"+repo.Name) || t.Visibility == nil || *t.Visibility != "public" || t.Archived == nil || t.Owner == nil || t.Owner.ID == nil || *t.Owner.ID <= 0 || t.Owner.Type == nil || *t.Owner.Type != "User" {
		return oidcSubjectState{}, fmt.Errorf("OIDC requires complete personal-public repository identity")
	}
	var raw map[string]json.RawMessage
	if err := c.get(oidcSubjectPath(repo), &raw); err != nil {
		return oidcSubjectState{}, err
	}
	value, immutable, err := decodeOIDCSubject(raw)
	if err != nil {
		return oidcSubjectState{}, err
	}
	return oidcSubjectState{*t.ID, *t.Owner.ID, *t.Archived, value, immutable}, nil
}
func (c *OIDCSubjectClient) Read(repo repository.Repository) (settings.OIDCSubjectObservation, error) {
	state, err := c.read(repo)
	if err != nil {
		return settings.OIDCSubjectObservation{}, err
	}
	state.value.UseImmutableSubjectClaim = state.immutable
	return settings.NewOIDCSubjectObservation(state.value)
}
func validOIDCAcknowledgment(b []byte) bool {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return true
	}
	var obj map[string]json.RawMessage
	return json.Unmarshal(b, &obj) == nil && obj != nil && len(obj) == 0
}
func (c *OIDCSubjectClient) Update(repo repository.Repository, desired settings.OIDCSubjectTemplate) (bool, error) {
	pre := func(err error) (bool, error) { return false, &securityPreWriteFailure{err} }
	if err := settings.ValidateOIDCSubject(desired); err != nil {
		return pre(err)
	}
	desired = settings.CloneOIDCSubject(desired)
	before, err := c.read(repo)
	if err != nil {
		return pre(err)
	}
	before.value.UseImmutableSubjectClaim = before.immutable
	if _, err := settings.OIDCSubjectBefore(before.value, desired); err != nil {
		return pre(err)
	}
	if settings.OIDCSubjectMatches(before.value, desired) {
		return false, nil
	}
	if before.archived {
		return pre(fmt.Errorf("archived repository cannot update OIDC template"))
	}
	if before.immutable == nil {
		return pre(fmt.Errorf("OIDC immutable companion is unavailable; cannot preserve it for update"))
	}
	complete := settings.CloneOIDCSubject(before.value)
	if desired.UseDefaultTemplate != nil {
		complete.UseDefaultTemplate, complete.SubjectClaimTemplate = desired.UseDefaultTemplate, desired.SubjectClaimTemplate
	}
	if desired.UseImmutableSubjectClaim != nil {
		complete.UseImmutableSubjectClaim = desired.UseImmutableSubjectClaim
	}
	input := map[string]any{"use_default": *complete.UseDefaultTemplate, "use_immutable_subject": *complete.UseImmutableSubjectClaim}
	if !*complete.UseDefaultTemplate {
		input["include_claim_keys"] = *complete.SubjectClaimTemplate
	}
	body, _ := json.Marshal(input)
	r, err := c.rest.Request(http.MethodPut, oidcSubjectPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("OIDC update failed; configuration may have changed")
	}
	b, readErr := readOIDCBody(r.Body)
	r.Body.Close()
	if readErr != nil || r.StatusCode != http.StatusCreated || !validOIDCAcknowledgment(b) {
		return false, fmt.Errorf("OIDC acknowledgment invalid; configuration may have changed")
	}
	after, err := c.read(repo)
	if err != nil {
		return false, fmt.Errorf("OIDC readback unavailable; configuration may have changed")
	}
	after.value.UseImmutableSubjectClaim = after.immutable
	if after.id != before.id || after.ownerID != before.ownerID || after.archived != before.archived || !settings.OIDCSubjectMatches(after.value, complete) {
		return false, fmt.Errorf("OIDC readback did not confirm original identity, template and immutable companion; configuration may have changed")
	}
	return true, nil
}
