package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type PropertiesClient struct {
	rest    *api.RESTClient
	graphql *api.GraphQLClient
}
type propertiesPreWriteFailure struct{ cause error }

func (e *propertiesPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *propertiesPreWriteFailure) Unwrap() error        { return e.cause }
func (*propertiesPreWriteFailure) ReadFailedBeforeWrite() {}
func NewPropertiesClient(host string) (*PropertiesClient, error) {
	c, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &propertiesPreWriteFailure{err}
	}
	g, err := api.NewGraphQLClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &propertiesPreWriteFailure{err}
	}
	return &PropertiesClient{rest: c, graphql: g}, nil
}

// The REST schema does not document regex. Observe it through the documented
// GraphQL field; an omitted field or partial error never means unconstrained.
const propertyRegexQuery = `query($owner: String!, $property: String!) {
 organization(login: $owner) { databaseId login repositoryCustomProperty(propertyName: $property) { propertyName valueType regex } }
}`

func (c *PropertiesClient) readRegex(id propertyIdentity, name string) (string, error) {
	var response map[string]json.RawMessage
	if err := c.graphql.Do(propertyRegexQuery, map[string]any{"owner": id.Owner, "property": name}, &response); err != nil {
		return "", fmt.Errorf("read custom property string constraint: %w", err)
	}
	org, err := accessObject(response["organization"], "custom property organization")
	if err != nil {
		return "", err
	}
	var databaseID *int64
	login, err := accessString(org, "login")
	if err != nil || json.Unmarshal(org["databaseId"], &databaseID) != nil || databaseID == nil || *databaseID != id.OwnerID || !strings.EqualFold(login, id.Owner) {
		return "", fmt.Errorf("custom property organization identity mismatch")
	}
	property, err := accessObject(org["repositoryCustomProperty"], "custom property string schema")
	if err != nil {
		return "", err
	}
	actualName, err := accessString(property, "propertyName")
	if err != nil || actualName != name {
		return "", fmt.Errorf("custom property schema name mismatch")
	}
	kind, err := accessString(property, "valueType")
	if err != nil || kind != "STRING" {
		return "", fmt.Errorf("custom property schema type mismatch")
	}
	var regex *string
	if json.Unmarshal(property["regex"], &regex) != nil {
		return "", fmt.Errorf("missing or invalid custom property regex constraint")
	}
	if regex == nil {
		return "", nil
	}
	return *regex, nil
}

type propertyIdentity struct {
	ID, OwnerID     int64
	FullName, Owner string
	Archived        bool
}

func (c *PropertiesClient) identity(repo repository.Repository) (propertyIdentity, error) {
	var v struct {
		ID       *int64  `json:"id"`
		FullName *string `json:"full_name"`
		Archived *bool   `json:"archived"`
		Owner    *struct {
			ID          *int64 `json:"id"`
			Login, Type *string
		} `json:"owner"`
	}
	if err := c.rest.Get("repos/"+repo.Owner+"/"+repo.Name, &v); err != nil {
		return propertyIdentity{}, err
	}
	if v.ID == nil || *v.ID <= 0 || v.FullName == nil || !strings.EqualFold(*v.FullName, repo.Owner+"/"+repo.Name) || v.Archived == nil || v.Owner == nil || v.Owner.ID == nil || *v.Owner.ID <= 0 || v.Owner.Login == nil || !strings.EqualFold(*v.Owner.Login, repo.Owner) || v.Owner.Type == nil || *v.Owner.Type != "Organization" {
		return propertyIdentity{}, fmt.Errorf("custom properties require a completely observed organization-owned repository")
	}
	return propertyIdentity{*v.ID, *v.Owner.ID, *v.FullName, *v.Owner.Login, *v.Archived}, nil
}
func (c *PropertiesClient) completeList(path string) ([]map[string]json.RawMessage, error) {
	resp, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// These two documented endpoints return a complete array, not a paged envelope.
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Link") != "" {
		return nil, fmt.Errorf("incomplete or unexpected custom property collection response")
	}
	var rows []map[string]json.RawMessage
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&rows); err != nil || rows == nil {
		return nil, fmt.Errorf("invalid custom property collection")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing custom property collection data")
	}
	return rows, nil
}
func propertyRawValue(raw json.RawMessage) (any, error) {
	if raw == nil {
		return nil, fmt.Errorf("missing custom property value")
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return settings.PropertyValue(v)
}
func (c *PropertiesClient) read(repo repository.Repository, selected map[string]bool) (settings.PropertyObservation, propertyIdentity, error) {
	id, err := c.identity(repo)
	if err != nil {
		return settings.PropertyObservation{}, id, err
	}
	rows, err := c.completeList("orgs/" + id.Owner + "/properties/schema")
	if err != nil {
		return settings.PropertyObservation{}, id, fmt.Errorf("read custom property schema: %w", err)
	}
	schemas := make([]settings.PropertySchema, 0, len(rows))
	for _, row := range rows {
		var s settings.PropertySchema
		for key, target := range map[string]*string{"property_name": &s.Name, "value_type": &s.Type, "source_type": &s.Source, "values_editable_by": &s.Editable} {
			if raw, ok := row[key]; ok {
				if err := json.Unmarshal(raw, target); err != nil {
					return settings.PropertyObservation{}, id, fmt.Errorf("invalid schema %s", key)
				}
			}
		}
		for key, target := range map[string]**bool{"required": &s.Required, "require_explicit_values": &s.Explicit} {
			if raw, ok := row[key]; ok {
				if err := json.Unmarshal(raw, target); err != nil {
					return settings.PropertyObservation{}, id, fmt.Errorf("invalid schema %s", key)
				}
			}
		}
		if raw, ok := row["default_value"]; ok {
			s.DefaultPresent = true
			s.Default, err = propertyRawValue(raw)
			if err != nil {
				return settings.PropertyObservation{}, id, err
			}
		}
		if raw, ok := row["allowed_values"]; ok {
			if err := json.Unmarshal(raw, &s.Allowed); err != nil {
				return settings.PropertyObservation{}, id, fmt.Errorf("invalid allowed values")
			}
		}
		if s.Type == "string" && (selected == nil || selected[s.Name]) {
			s.Regex, err = c.readRegex(id, s.Name)
			if err != nil {
				return settings.PropertyObservation{}, id, err
			}
			s.RegexObserved = true
		}
		schemas = append(schemas, s)
	}
	rows, err = c.completeList("repos/" + repo.Owner + "/" + repo.Name + "/properties/values")
	if err != nil {
		return settings.PropertyObservation{}, id, fmt.Errorf("read custom property values: %w", err)
	}
	values := settings.PropertyDeclarations{}
	seen := map[string]bool{}
	for _, row := range rows {
		var name string
		if json.Unmarshal(row["property_name"], &name) != nil || !settings.ValidPropertyName(name) || seen[name] {
			return settings.PropertyObservation{}, id, fmt.Errorf("invalid or duplicate property name")
		}
		seen[name] = true
		v, err := propertyRawValue(row["value"])
		if err != nil {
			return settings.PropertyObservation{}, id, err
		}
		values[name] = v
	}
	o, err := settings.NewPropertyObservation(schemas, values)
	if err != nil {
		return o, id, err
	}
	end, err := c.identity(repo)
	if err != nil || end != id {
		return settings.PropertyObservation{}, id, fmt.Errorf("repository identity changed during custom property observation")
	}
	return o, id, nil
}
func (c *PropertiesClient) Read(repo repository.Repository) (settings.PropertyObservation, error) {
	o, _, err := c.read(repo, nil)
	return o, err
}

func selectedProperties(names []string) (map[string]bool, error) {
	selected := make(map[string]bool, len(names))
	for _, name := range names {
		if !settings.ValidPropertyName(name) {
			return nil, fmt.Errorf("invalid custom property name %q", name)
		}
		if selected[name] {
			return nil, fmt.Errorf("duplicate custom property selection %q", name)
		}
		selected[name] = true
	}
	return selected, nil
}

func (c *PropertiesClient) ReadSelected(repo repository.Repository, names []string) (settings.PropertyObservation, error) {
	selected, err := selectedProperties(names)
	if err != nil {
		return settings.PropertyObservation{}, err
	}
	o, _, err := c.read(repo, selected)
	return o, err
}
func (c *PropertiesClient) Update(repo repository.Repository, d settings.PropertyDeclarations) (bool, error) {
	pre := func(err error) (bool, error) { return false, &propertiesPreWriteFailure{err} }
	if err := d.Validate(); err != nil {
		return pre(err)
	}
	if len(d) == 0 {
		return false, nil
	}
	names := make([]string, 0, len(d))
	for name := range d {
		names = append(names, name)
	}
	sort.Strings(names)
	selected, err := selectedProperties(names)
	if err != nil {
		return pre(err)
	}
	o, id, err := c.read(repo, selected)
	if err != nil {
		return pre(err)
	}
	if id.Archived {
		return pre(fmt.Errorf("archived repository cannot update custom properties without explicit unarchive"))
	}
	changes, err := settings.CompareProperties(o, d)
	if err != nil {
		return pre(err)
	}
	if len(changes) == 0 {
		return false, nil
	}
	expected, _ := o.Values()
	properties := []map[string]any{}
	for _, name := range names {
		v, _ := settings.PropertyValue(d[name])
		properties = append(properties, map[string]any{"property_name": name, "value": v})
		if v == nil {
			delete(expected, name)
		} else {
			expected[name] = v
		}
	}
	body, err := json.Marshal(map[string]any{"properties": properties})
	if err != nil {
		return pre(err)
	}
	fresh, freshID, err := c.read(repo, selected)
	if err != nil {
		return pre(err)
	}
	if freshID != id || !reflect.DeepEqual(o, fresh) {
		return pre(fmt.Errorf("custom property schema, values or repository changed before update"))
	}
	resp, err := c.rest.Request(http.MethodPatch, "repos/"+repo.Owner+"/"+repo.Name+"/properties/values", bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("update custom properties (write may have applied): %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return false, fmt.Errorf("unexpected custom property update status %d (write may have applied)", resp.StatusCode)
	}
	actual, actualID, err := c.read(repo, selected)
	if err != nil {
		return false, fmt.Errorf("verify custom properties (write may have applied): %w", err)
	}
	values, _ := actual.Values()
	if actualID != id || !reflect.DeepEqual(expected, values) {
		return false, fmt.Errorf("custom property identity/value readback mismatch (write may have applied)")
	}
	for name := range d {
		s1, _ := o.Schema(name)
		s2, err := actual.Schema(name)
		if err != nil || !reflect.DeepEqual(s1, s2) {
			return false, fmt.Errorf("custom property schema changed during update (write may have applied)")
		}
	}
	return true, nil
}
