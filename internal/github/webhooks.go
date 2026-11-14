package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func NewWebhooksClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, LogIgnoreEnv: true, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure webhook client failed")}
	}
	return &Client{rest: rest}, nil
}
func webhookPath(repo repository.Repository, id int64) string {
	p := securityRepositoryPath(repo) + "/hooks"
	if id > 0 {
		p += "/" + strconv.FormatInt(id, 10)
	}
	return p
}

// Inspect every JSON object before decoding so duplicate keys cannot supply conflicting evidence.
func webhookJSON(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var value func() error
	value = func() error {
		t, e := d.Token()
		if e != nil {
			return e
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				key, ok := k.(string)
				if !ok || seen[key] {
					return fmt.Errorf("duplicate member")
				}
				seen[key] = true
				if e = value(); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = value(); e != nil {
					return e
				}
			}
		default:
			return fmt.Errorf("invalid delimiter")
		}
		_, e = d.Token()
		return e
	}
	if value() != nil {
		return fmt.Errorf("incomplete webhook response")
	}
	if _, e := d.Token(); e != io.EOF {
		return fmt.Errorf("incomplete webhook response")
	}
	if json.Unmarshal(data, out) != nil {
		return fmt.Errorf("incomplete webhook response")
	}
	return nil
}
func (c *Client) webhookRequest(method, path string, payload any, out any) (string, error) {
	var body io.Reader
	if payload != nil {
		b, e := json.Marshal(payload)
		if e != nil {
			return "", fmt.Errorf("invalid webhook request")
		}
		body = bytes.NewReader(b)
	}
	response, err := c.rest.Request(method, path, body)
	if err != nil {
		return "", fmt.Errorf("webhook request unavailable")
	}
	defer response.Body.Close()
	b, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil || len(b) > 8<<20 || response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("webhook response unavailable or incomplete")
	}
	if err = webhookJSON(b, out); err != nil {
		return "", err
	}
	return strings.Join(response.Header.Values("Link"), ","), nil
}
func (c *Client) webhookTarget(repo repository.Repository) (settings.WebhookTarget, error) {
	var raw struct {
		ID         *int64  `json:"id"`
		FullName   *string `json:"full_name"`
		Visibility *string `json:"visibility"`
		Private    *bool   `json:"private"`
		Archived   *bool   `json:"archived"`
		Owner      *struct {
			ID    *int64  `json:"id"`
			Login *string `json:"login"`
			Type  *string `json:"type"`
		} `json:"owner"`
	}
	if repo.Host != "github.com" {
		return settings.WebhookTarget{}, fmt.Errorf("webhooks require a github.com target")
	}
	link, err := c.webhookRequest(http.MethodGet, securityRepositoryPath(repo), nil, &raw)
	if err != nil {
		return settings.WebhookTarget{}, err
	}
	if link != "" || raw.ID == nil || *raw.ID <= 0 || raw.FullName == nil || *raw.FullName != repo.Owner+"/"+repo.Name || raw.Visibility == nil || *raw.Visibility != "public" || raw.Private == nil || *raw.Private || raw.Archived == nil || raw.Owner == nil || raw.Owner.ID == nil || *raw.Owner.ID <= 0 || raw.Owner.Login == nil || *raw.Owner.Login != repo.Owner || raw.Owner.Type == nil || *raw.Owner.Type != "User" {
		return settings.WebhookTarget{}, fmt.Errorf("webhooks require complete personally owned public repository identity")
	}
	return settings.WebhookTarget{ID: *raw.ID, OwnerID: *raw.Owner.ID, FullName: *raw.FullName, Archived: *raw.Archived}, nil
}
func webhookRequired(raw map[string]json.RawMessage, name string, out any) bool {
	b, ok := raw[name]
	return ok && !bytes.Equal(bytes.TrimSpace(b), []byte("null")) && json.Unmarshal(b, out) == nil
}

// Compare only exact numeric zero/one, without floating-point rounding or
// allocating powers for a hostile exponent in an otherwise valid JSON number.
func webhookNumericBit(value string) (bool, bool) {
	mantissa, exponent := value, int64(0)
	if i := strings.IndexAny(value, "eE"); i >= 0 {
		mantissa = value[:i]
	}
	negative := strings.HasPrefix(mantissa, "-")
	mantissa = strings.TrimPrefix(mantissa, "-")
	fraction := 0
	if i := strings.IndexByte(mantissa, '.'); i >= 0 {
		fraction = len(mantissa) - i - 1
		mantissa = mantissa[:i] + mantissa[i+1:]
	}
	digits := strings.TrimLeft(mantissa, "0")
	if digits == "" {
		return false, true
	}
	if negative {
		return false, false
	}
	significant := strings.TrimRight(digits, "0")
	if significant != "1" {
		return false, false
	}
	if i := strings.IndexAny(value, "eE"); i >= 0 {
		var err error
		exponent, err = strconv.ParseInt(value[i+1:], 10, 64)
		if err != nil {
			return false, false
		}
	}
	return true, exponent == int64(fraction-(len(digits)-len(significant)))
}

func decodeWebhookConfig(raw map[string]json.RawMessage) (settings.WebhookConfig, error) {
	var result settings.WebhookConfig
	fail := func() (settings.WebhookConfig, error) {
		return settings.WebhookConfig{}, fmt.Errorf("webhook configuration is incomplete or unsupported")
	}
	var native string
	if !webhookRequired(raw, "url", &result.PayloadURL) || !webhookRequired(raw, "content_type", &native) {
		return fail()
	}
	switch native {
	case "json":
		result.ContentType = "application/json"
	case "form":
		result.ContentType = "application/x-www-form-urlencoded"
	default:
		return fail()
	}
	ssl := bytes.TrimSpace(raw["insecure_ssl"])
	if len(ssl) > 0 && ssl[0] == '"' {
		var value string
		if json.Unmarshal(ssl, &value) != nil || (value != "0" && value != "1") {
			return fail()
		}
		result.InsecureSSL = value == "1"
	} else {
		var number json.Number
		if len(ssl) == 0 || ssl[0] == '"' || bytes.Equal(ssl, []byte("null")) || json.Unmarshal(ssl, &number) != nil {
			return fail()
		}
		bit, ok := webhookNumericBit(number.String())
		if !ok {
			return fail()
		}
		result.InsecureSSL = bit
	}

	for k, b := range raw {
		switch k {
		case "url", "content_type", "insecure_ssl":
		case "secret":
			var ignored *string
			if json.Unmarshal(b, &ignored) != nil || ignored == nil {
				return fail()
			}
		default:
			return fail()
		}
	}
	if result.Validate() != nil {
		return fail()
	}
	return result, nil
}
func decodeWebhook(repo repository.Repository, raw map[string]json.RawMessage) (settings.WebhookState, error) {
	var s settings.WebhookState
	var name, kind, apiURL string
	var config map[string]json.RawMessage
	fail := func() (settings.WebhookState, error) {
		return settings.WebhookState{}, fmt.Errorf("webhook identity or parent configuration is incomplete")
	}
	if !webhookRequired(raw, "id", &s.ID) || s.ID <= 0 || !webhookRequired(raw, "name", &name) || name != "web" || !webhookRequired(raw, "type", &kind) || kind != "Repository" || !webhookRequired(raw, "url", &apiURL) || apiURL != "https://api.github.com/"+strings.TrimPrefix(webhookPath(repo, s.ID), "/") || !webhookRequired(raw, "created_at", &s.CreatedAt) || !webhookRequired(raw, "active", &s.Active) || !webhookRequired(raw, "events", &s.Events) || !webhookRequired(raw, "config", &config) {
		return fail()
	}
	if _, err := time.Parse(time.RFC3339, s.CreatedAt); err != nil {
		return fail()
	}
	var err error
	s.Config, err = decodeWebhookConfig(config)
	if err != nil {
		return settings.WebhookState{}, err
	}
	if s.Validate() != nil {
		return fail()
	}
	return s, nil
}
func (c *Client) readWebhook(repo repository.Repository, id int64) (settings.WebhookState, error) {
	var raw map[string]json.RawMessage
	link, err := c.webhookRequest(http.MethodGet, webhookPath(repo, id), nil, &raw)
	if err != nil {
		return settings.WebhookState{}, err
	}
	if link != "" {
		return settings.WebhookState{}, fmt.Errorf("unexpected webhook detail pagination")
	}
	s, err := decodeWebhook(repo, raw)
	if err != nil {
		return settings.WebhookState{}, err
	}
	if s.ID != id {
		return settings.WebhookState{}, fmt.Errorf("webhook identity differs")
	}
	var cfg map[string]json.RawMessage
	link, err = c.webhookRequest(http.MethodGet, webhookPath(repo, id)+"/config", nil, &cfg)
	if err != nil {
		return settings.WebhookState{}, err
	}
	config, err := decodeWebhookConfig(cfg)
	if err != nil {
		return settings.WebhookState{}, err
	}
	if link != "" || config != s.Config {
		return settings.WebhookState{}, fmt.Errorf("webhook parent and configuration disagree")
	}
	return s, nil
}

// Parse relation parameters, never pagination destinations. Even a syntactically
// different next relation makes a short page incomplete. Malformed evidence is
// unavailable rather than an implicit terminal page.
func webhookHasNext(link string) (bool, error) {
	next := false
	for remaining := strings.TrimSpace(link); remaining != ""; {
		fail := func() (bool, error) { return false, fmt.Errorf("incomplete webhook pagination metadata") }
		end := strings.IndexByte(remaining, '>')
		if !strings.HasPrefix(remaining, "<") || end <= 1 {
			return fail()
		}
		suffix := remaining[end+1:]
		cut := len(suffix)
		quoted, escaped := false, false
		for i, b := range []byte(suffix) {
			if escaped {
				escaped = false
				continue
			}
			if quoted && b == '\\' {
				escaped = true
				continue
			}
			if b == '"' {
				quoted = !quoted
				continue
			}
			if !quoted && b == ';' {
				// Link relations use plain rel. MIME extended-parameter decoding
				// could overwrite it, so reject rel* and its continuation forms.
				parameter := suffix[i+1:]
				if end := strings.IndexAny(parameter, "=;,"); end >= 0 {
					parameter = parameter[:end]
				}
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(parameter)), "rel*") {
					return fail()
				}
			}
			if !quoted && b == ',' {
				cut = i
				break
			}
		}
		if quoted || escaped {
			return fail()
		}
		_, params, err := mime.ParseMediaType("application/link" + suffix[:cut])
		if err != nil || len(strings.Fields(params["rel"])) == 0 {
			return fail()
		}
		for _, relation := range strings.Fields(params["rel"]) {
			next = next || strings.EqualFold(relation, "next")
		}
		if cut == len(suffix) {
			remaining = ""
		} else {
			remaining = strings.TrimSpace(suffix[cut+1:])
			if remaining == "" {
				return fail()
			}
		}
	}
	return next, nil
}

// nil IDs requests a complete export inventory; an explicit empty selection reads nothing.
func (c *Client) ReadWebhooks(repo repository.Repository, ids []int64) (settings.WebhookObservation, error) {
	var o settings.WebhookObservation
	inventory := map[int64]settings.WebhookState{}
	var err error
	if ids != nil && len(ids) == 0 {
		return o, fmt.Errorf("empty webhook selection has no observation")
	}
	o.Target, err = c.webhookTarget(repo)
	if err != nil {
		return settings.WebhookObservation{}, err
	}
	if ids == nil {
		ids = []int64{}
		seen := map[int64]bool{}
		for page := 1; ; page++ {
			var rows []map[string]json.RawMessage
			link, e := c.webhookRequest(http.MethodGet, fmt.Sprintf("%s?per_page=100&page=%d", webhookPath(repo, 0), page), nil, &rows)
			if e != nil {
				return o, e
			}
			hasNext, linkErr := webhookHasNext(link)
			if linkErr != nil {
				return settings.WebhookObservation{}, linkErr
			}
			if rows == nil || len(rows) > 100 || len(rows) < 100 && hasNext {
				return o, fmt.Errorf("incomplete webhook inventory")
			}
			for _, row := range rows {
				s, e := decodeWebhook(repo, row)
				if e != nil {
					return o, e
				}
				if seen[s.ID] {
					return o, fmt.Errorf("repeated webhook inventory identity")
				}
				seen[s.ID] = true
				inventory[s.ID] = s
				ids = append(ids, s.ID)
			}
			if len(rows) < 100 {
				break
			}
		}
	}
	o.Values = make([]settings.WebhookState, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return settings.WebhookObservation{}, fmt.Errorf("invalid selected webhook id")
		}
		s, e := c.readWebhook(repo, id)
		if e != nil {
			return settings.WebhookObservation{}, e
		}
		if initial, ok := inventory[id]; ok && !settings.WebhookIdentityMatches(initial, s) {
			return settings.WebhookObservation{}, fmt.Errorf("webhook inventory identity changed")
		}
		o.Values = append(o.Values, s)
	}
	// Bind the complete inventory to the same repository after all reads.
	actual, err := c.webhookTarget(repo)
	if err != nil {
		return settings.WebhookObservation{}, err
	}
	if actual != o.Target {
		return settings.WebhookObservation{}, fmt.Errorf("webhook repository identity changed")
	}
	return o, o.Validate()
}
func (c *Client) ApplyWebhook(repo repository.Repository, selected settings.WebhookSelection) (bool, error) {
	pre := func(err error) (bool, error) { return false, &securityPreWriteFailure{err} }
	if selected.Desired.Validate() != nil || selected.Before.Validate() != nil || selected.Before.ID != selected.Desired.ID || selected.Target.Validate() != nil {
		return pre(fmt.Errorf("invalid webhook selection"))
	}
	target, err := c.webhookTarget(repo)
	if err != nil {
		return pre(err)
	}
	if target != selected.Target {
		return pre(fmt.Errorf("webhook repository identity changed"))
	}
	before, err := c.readWebhook(repo, selected.Desired.ID)
	if err != nil {
		return pre(err)
	}
	if !settings.WebhookIdentityMatches(selected.Before, before) {
		return pre(fmt.Errorf("webhook identity changed"))
	}
	if before.Config.Matches(selected.Desired) {
		return false, nil
	}
	// A second identity probe immediately precedes every needed write.
	target, err = c.webhookTarget(repo)
	if err != nil {
		return pre(err)
	}
	if target != selected.Target || target.Archived {
		return pre(fmt.Errorf("webhook target changed or is archived"))
	}
	payload := map[string]string{}
	if selected.Desired.PayloadURL != nil && *selected.Desired.PayloadURL != before.Config.PayloadURL {
		payload["url"] = *selected.Desired.PayloadURL
	}
	if selected.Desired.ContentType != nil && *selected.Desired.ContentType != before.Config.ContentType {
		native := "json"
		if *selected.Desired.ContentType == "application/x-www-form-urlencoded" {
			native = "form"
		}
		payload["content_type"] = native
	}
	var raw map[string]json.RawMessage
	link, err := c.webhookRequest(http.MethodPatch, webhookPath(repo, before.ID)+"/config", payload, &raw)
	if err != nil {
		return true, err
	}
	ack, err := decodeWebhookConfig(raw)
	if err != nil {
		return true, err
	}
	if link != "" || ack != before.Config.Overlay(selected.Desired) {
		return true, fmt.Errorf("webhook acknowledgment differs from requested configuration")
	}
	after, err := c.readWebhook(repo, before.ID)
	if err != nil {
		return true, err
	}
	target, err = c.webhookTarget(repo)
	if err != nil {
		return true, err
	}
	if target != selected.Target || !settings.WebhookPreserved(before, after, selected.Desired) {
		return true, fmt.Errorf("webhook verification differs from requested or preserved state")
	}
	return true, nil
}
