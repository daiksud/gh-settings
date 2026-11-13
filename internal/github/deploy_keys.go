package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func NewDeployKeysClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, LogIgnoreEnv: true, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure deploy key client")}
	}
	return &Client{rest: rest}, nil
}

const deployKeysResponseLimit = 8 << 20

func deployKeysBody(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, deployKeysResponseLimit+1))
	if err != nil || len(b) > deployKeysResponseLimit {
		return nil, fmt.Errorf("deploy key response unavailable or oversized")
	}
	return b, nil
}
func deployKeyPath(repo repository.Repository) string { return securityRepositoryPath(repo) + "/keys" }
func deployKeyObjectPath(repo repository.Repository, id int64) string {
	return deployKeyPath(repo) + "/" + strconv.FormatInt(id, 10)
}
func deployKeyURL(repo repository.Repository, id int64) string {
	return "https://api.github.com/" + strings.TrimPrefix(deployKeyObjectPath(repo, id), "/")
}
func (c *Client) deployKeysGET(path string, out any) (string, error) {
	response, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("deploy key observation unavailable")
	}
	defer response.Body.Close()
	b, err := deployKeysBody(response.Body)
	if err != nil || response.StatusCode != 200 || json.Unmarshal(b, out) != nil {
		return "", fmt.Errorf("deploy key observation incomplete or unavailable")
	}
	return response.Header.Get("Link"), nil
}
func (c *Client) deployKeysTarget(repo repository.Repository) (settings.DeployKeyTarget, error) {
	var raw struct {
		ID         *int64  `json:"id"`
		FullName   *string `json:"full_name"`
		Archived   *bool   `json:"archived"`
		Visibility *string `json:"visibility"`
		Owner      *struct {
			ID    *int64  `json:"id"`
			Login *string `json:"login"`
			Type  *string `json:"type"`
		} `json:"owner"`
	}
	if repo.Host != "github.com" {
		return settings.DeployKeyTarget{}, fmt.Errorf("deploy keys require a github.com target")
	}
	link, err := c.deployKeysGET(securityRepositoryPath(repo), &raw)
	if err != nil {
		return settings.DeployKeyTarget{}, err
	}
	if link != "" || raw.ID == nil || *raw.ID <= 0 || raw.FullName == nil || !strings.EqualFold(*raw.FullName, repo.Owner+"/"+repo.Name) || raw.Archived == nil || raw.Visibility == nil || *raw.Visibility != "public" || raw.Owner == nil || raw.Owner.ID == nil || *raw.Owner.ID <= 0 || raw.Owner.Login == nil || !strings.EqualFold(*raw.Owner.Login, repo.Owner) || raw.Owner.Type == nil || *raw.Owner.Type != "User" {
		return settings.DeployKeyTarget{}, fmt.Errorf("deploy keys require complete personally owned public repository identity")
	}
	return settings.DeployKeyTarget{ID: *raw.ID, OwnerID: *raw.Owner.ID, FullName: repo.Owner + "/" + repo.Name, Archived: *raw.Archived}, nil
}
func decodeDeployKey(repo repository.Repository, raw map[string]json.RawMessage) (settings.DeployKeyState, error) {
	var result settings.DeployKeyState
	fail := func() (settings.DeployKeyState, error) {
		return settings.DeployKeyState{}, fmt.Errorf("deploy key observation is incomplete or unsupported")
	}
	required := func(key string, out any) bool {
		b, ok := raw[key]
		return ok && !bytes.Equal(bytes.TrimSpace(b), []byte("null")) && json.Unmarshal(b, out) == nil
	}
	var material, rawURL string
	if !required("id", &result.ID) || result.ID <= 0 || !required("title", &result.Title) || !required("key", &material) || !required("read_only", &result.ReadOnly) || !required("verified", &result.Verified) || !required("created_at", &result.CreatedAt) || result.CreatedAt == "" || !required("url", &rawURL) {
		return fail()
	}
	if _, err := time.Parse(time.RFC3339, result.CreatedAt); err != nil {
		return fail()
	}
	key, err := settings.ParseDeployPublicKey(material)
	if err != nil {
		return fail()
	}
	result.Key = key
	expected, err := url.Parse(deployKeyURL(repo, result.ID))
	if err != nil {
		return fail()
	}
	actual, err := url.Parse(rawURL)
	if err != nil || actual.Scheme != "https" || !strings.EqualFold(actual.Host, expected.Host) || actual.User != nil || actual.RawQuery != "" || actual.ForceQuery || actual.Fragment != "" || actual.RawFragment != "" || actual.RawPath != "" || !strings.EqualFold(actual.Path, expected.Path) {
		return fail()
	}
	result.APIURL = expected.String()
	for name, b := range raw {
		switch name {
		case "id", "title", "key", "read_only", "verified", "created_at", "url":
		case "enabled":
			var v *bool
			if json.Unmarshal(b, &v) != nil || v == nil {
				return fail()
			}
			result.Enabled = v
		case "added_by", "last_used":
			value := settings.DeployKeyOptionalString{Present: true, Null: bytes.Equal(bytes.TrimSpace(b), []byte("null"))}
			if !value.Null {
				if json.Unmarshal(b, &value.Value) != nil {
					return fail()
				}
				if name == "last_used" {
					if _, err := time.Parse(time.RFC3339, value.Value); err != nil {
						return fail()
					}
				}
			}
			if name == "added_by" {
				result.AddedBy = value
			} else {
				result.LastUsed = value
			}
		default:
			return fail()
		}
	}
	if result.Declaration().Validate() != nil {
		return fail()
	}
	return result, nil
}
func (c *Client) ReadDeployKeys(repo repository.Repository) (settings.DeployKeyObservation, error) {
	target, err := c.deployKeysTarget(repo)
	if err != nil {
		return settings.DeployKeyObservation{}, err
	}
	values := []settings.DeployKeyState{}
	for page := 1; ; page++ {
		var raw []map[string]json.RawMessage
		link, err := c.deployKeysGET(fmt.Sprintf("%s?per_page=100&page=%d", deployKeyPath(repo), page), &raw)
		if err != nil {
			return settings.DeployKeyObservation{}, err
		}
		if raw == nil || len(raw) > 100 || len(raw) < 100 && strings.Contains(link, `rel="next"`) {
			return settings.DeployKeyObservation{}, fmt.Errorf("deploy key inventory is incomplete")
		}
		for _, item := range raw {
			v, err := decodeDeployKey(repo, item)
			if err != nil {
				return settings.DeployKeyObservation{}, err
			}
			values = append(values, v)
		}
		// Each page must advance unique identities, before requesting another page.
		if _, err := settings.NewDeployKeyObservation(target, values); err != nil {
			return settings.DeployKeyObservation{}, err
		}
		if len(raw) < 100 {
			break
		}
	}
	return settings.NewDeployKeyObservation(target, values)
}
func (c *Client) readDeployKey(repo repository.Repository, id int64) (settings.DeployKeyState, error) {
	var raw map[string]json.RawMessage
	link, err := c.deployKeysGET(deployKeyObjectPath(repo, id), &raw)
	if err != nil {
		return settings.DeployKeyState{}, err
	}
	if link != "" {
		return settings.DeployKeyState{}, fmt.Errorf("deploy key detail has unsupported pagination")
	}
	value, err := decodeDeployKey(repo, raw)
	if err != nil || value.ID != id {
		return settings.DeployKeyState{}, fmt.Errorf("deploy key detail identity is incomplete or mismatched")
	}
	return value, nil
}
func (c *Client) CreateDeployKey(repo repository.Repository, expected settings.DeployKeyObservation, desired settings.DeployKeyDeclaration) (settings.DeployKeyWriteResult, error) {
	var result settings.DeployKeyWriteResult
	pre := func(err error) (settings.DeployKeyWriteResult, error) { return result, &securityPreWriteFailure{err} }
	if desired.ID != nil || desired.Validate() != nil {
		return pre(fmt.Errorf("invalid deploy key creation intent"))
	}
	target, _, err := expected.Values()
	if err != nil || target.Archived {
		return pre(fmt.Errorf("deploy key creation requires a complete active target"))
	}
	fresh, err := c.ReadDeployKeys(repo)
	if err != nil {
		return pre(err)
	}
	if !settings.DeployKeyInventoryEqual(expected, fresh) {
		return pre(fmt.Errorf("deploy key inventory changed before creation"))
	}
	declarations, _ := settings.NewDeployKeyDeclarations([]settings.DeployKeyDeclaration{desired})
	if _, _, err := settings.PrepareDeployKeys(fresh, &declarations, false); err != nil {
		return pre(err)
	}
	input := map[string]any{"title": desired.Title, "key": desired.Key.Canonical(), "read_only": !desired.AllowWriteAccess}
	body, _ := json.Marshal(input)
	response, err := c.rest.Request(http.MethodPost, deployKeyPath(repo), bytes.NewReader(body))
	uncertain := func() (settings.DeployKeyWriteResult, error) {
		return result, fmt.Errorf("deploy key creation is unconfirmed; remote effects may remain")
	}
	if err != nil {
		return uncertain()
	}
	b, readErr := deployKeysBody(response.Body)
	response.Body.Close()
	if readErr != nil || response.StatusCode != 201 {
		return uncertain()
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil || raw == nil {
		return uncertain()
	}
	// Only a successful creation response may supply a separately valid recovery ID.
	if id, ok := raw["id"]; ok {
		var value int64
		if json.Unmarshal(id, &value) == nil && value > 0 && !expected.HasID(value) {
			result.ID = value
		}
	}
	ack, err := decodeDeployKey(repo, raw)
	if err != nil || result.ID == 0 || ack.ID != result.ID || !settings.DeployKeyConfigurationMatches(ack, desired) {
		return uncertain()
	}
	observed, err := c.readDeployKey(repo, result.ID)
	if err != nil {
		return uncertain()
	}
	// Independent GET establishes optional state when ACK did not supply it.
	if ack.Enabled != nil && observed.Enabled != nil && *ack.Enabled != *observed.Enabled {
		return uncertain()
	}
	ack.Enabled = observed.Enabled
	if !settings.DeployKeyStableEqual(ack, observed) || !settings.DeployKeyConfigurationMatches(observed, desired) {
		return uncertain()
	}
	want, err := expected.WithCreated(observed)
	if err != nil {
		return uncertain()
	}
	actual, err := c.ReadDeployKeys(repo)
	if err != nil || !settings.DeployKeyInventoryEqual(want, actual) {
		return uncertain()
	}
	result.Observation = actual
	return result, nil
}
func (c *Client) DeleteDeployKey(repo repository.Repository, expected settings.DeployKeyObservation, id int64) (settings.DeployKeyObservation, error) {
	pre := func(err error) (settings.DeployKeyObservation, error) {
		return settings.DeployKeyObservation{}, &securityPreWriteFailure{err}
	}
	target, _, err := expected.Values()
	if err != nil || target.Archived {
		return pre(fmt.Errorf("deploy key deletion requires a complete active target"))
	}
	candidate, exists := expected.State(id)
	if !exists {
		return pre(fmt.Errorf("deploy key candidate is absent from the fixed inventory"))
	}
	fresh, err := c.ReadDeployKeys(repo)
	if err != nil {
		return pre(err)
	}
	if !settings.DeployKeyInventoryEqual(expected, fresh) {
		return pre(fmt.Errorf("deploy key inventory changed before deletion"))
	}
	detail, err := c.readDeployKey(repo, id)
	if err != nil || !settings.DeployKeyStableEqual(candidate, detail) {
		return pre(fmt.Errorf("deploy key candidate changed before deletion"))
	}
	response, err := c.rest.Request(http.MethodDelete, deployKeyObjectPath(repo, id), nil)
	uncertain := func() (settings.DeployKeyObservation, error) {
		return settings.DeployKeyObservation{}, fmt.Errorf("deploy key deletion is unconfirmed; remote effects may remain")
	}
	if err != nil {
		return uncertain()
	}
	b, readErr := deployKeysBody(response.Body)
	response.Body.Close()
	if readErr != nil || response.StatusCode != 204 || len(b) != 0 {
		return uncertain()
	}
	want, err := expected.Without(id)
	if err != nil {
		return uncertain()
	}
	actual, err := c.ReadDeployKeys(repo)
	if err != nil || !settings.DeployKeyInventoryEqual(want, actual) {
		return uncertain()
	}
	return actual, nil
}
