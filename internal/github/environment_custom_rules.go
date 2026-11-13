package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type environmentApp struct {
	ID                int64
	NodeID, Slug, URL string
}

func (c *EnvironmentsClient) customRequest(method, path string, body []byte, status int) ([]byte, error) {
	response, err := c.rest.Request(method, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("custom environment protection %s request failed", method)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil || len(data) > 1<<20 {
		return nil, fmt.Errorf("custom environment protection response unavailable or too large")
	}
	if response.StatusCode != status {
		return data, fmt.Errorf("custom environment protection %s returned unexpected status %d", method, response.StatusCode)
	}
	if status == http.StatusNoContent && len(data) != 0 {
		return nil, fmt.Errorf("custom environment protection DELETE returned a nonempty response")
	}
	return data, nil
}

func customFields(data []byte, allowed ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return nil, fmt.Errorf("custom environment protection requires an object response")
	}
	if err := environmentKnownFields(fields, allowed...); err != nil {
		return nil, fmt.Errorf("unsupported custom environment protection response fields")
	}
	return fields, nil
}

func customString(fields map[string]json.RawMessage, key string) (string, error) {
	var value string
	if err := requiredEnvironmentJSON(fields, key, &value); err != nil {
		return "", err
	}
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsFunc(value, unicode.IsControl) {
		return "", fmt.Errorf("invalid custom environment protection identity %s", key)
	}
	return value, nil
}

func decodeEnvironmentApp(data []byte) (environmentApp, error) {
	var app environmentApp
	fields, err := customFields(data, "id", "node_id", "slug", "integration_url")
	if err != nil {
		return app, err
	}
	if err := requiredEnvironmentJSON(fields, "id", &app.ID); err != nil || app.ID <= 0 {
		return app, fmt.Errorf("custom environment protection App requires positive id")
	}
	if app.NodeID, err = customString(fields, "node_id"); err != nil {
		return app, err
	}
	if app.Slug, err = customString(fields, "slug"); err != nil {
		return app, err
	}
	if app.URL, err = customString(fields, "integration_url"); err != nil {
		return app, err
	}
	return app, nil
}

// Positive acknowledgment IDs survive invalid companion data for recovery only.
func decodeEnvironmentCustomRule(data []byte) (settings.EnvironmentCustomRule, error) {
	var result settings.EnvironmentCustomRule
	var recoverID struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(data, &recoverID)
	if recoverID.ID > 0 {
		result.ID = recoverID.ID
	}
	fields, err := customFields(data, "id", "node_id", "enabled", "app")
	if err != nil {
		return result, err
	}
	if err := requiredEnvironmentJSON(fields, "id", &result.ID); err != nil || result.ID <= 0 {
		return result, fmt.Errorf("custom environment protection rule requires positive id")
	}
	if result.NodeID, err = customString(fields, "node_id"); err != nil {
		return result, err
	}
	var enabled bool
	if err := requiredEnvironmentJSON(fields, "enabled", &enabled); err != nil || !enabled {
		return result, fmt.Errorf("custom environment protection rule must be explicitly enabled")
	}
	app, err := decodeEnvironmentApp(fields["app"])
	if err != nil {
		return result, err
	}
	result.AppID, result.AppNodeID, result.Slug, result.IntegrationURL = app.ID, app.NodeID, app.Slug, app.URL
	return result, nil
}

func (c *EnvironmentsClient) readCustomRules(repo repository.Repository, name string) ([]settings.EnvironmentCustomRule, error) {
	data, err := c.customRequest(http.MethodGet, environmentPath(repo, name)+"/deployment_protection_rules", nil, 200)
	if err != nil {
		return nil, err
	}
	fields, err := customFields(data, "total_count", "custom_deployment_protection_rules")
	if err != nil {
		return nil, err
	}
	var count int
	var raw []json.RawMessage
	if err := requiredEnvironmentJSON(fields, "total_count", &count); err != nil {
		return nil, err
	}
	if err := requiredEnvironmentJSON(fields, "custom_deployment_protection_rules", &raw); err != nil {
		return nil, err
	}
	if count < 0 || count != len(raw) || count > 6 {
		return nil, fmt.Errorf("custom environment protection inventory is incomplete")
	}
	rules := make([]settings.EnvironmentCustomRule, 0, len(raw))
	for _, entry := range raw {
		rule, err := decodeEnvironmentCustomRule(entry)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if _, err := settings.EnvironmentCustomRuleIDs(rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *EnvironmentsClient) availableCustomApps(repo repository.Repository, name string) (map[int64]environmentApp, error) {
	apps := map[int64]environmentApp{}
	nodes, slugs := map[string]bool{}, map[string]bool{}
	total := -1
	for page := 1; ; page++ {
		data, err := c.customRequest(http.MethodGet, fmt.Sprintf("%s/deployment_protection_rules/apps?per_page=100&page=%d", environmentPath(repo, name), page), nil, 200)
		if err != nil {
			return nil, err
		}
		fields, err := customFields(data, "total_count", "available_custom_deployment_protection_rule_integrations")
		if err != nil {
			return nil, err
		}
		var count int
		var raw []json.RawMessage
		if err := requiredEnvironmentJSON(fields, "total_count", &count); err != nil {
			return nil, err
		}
		if err := requiredEnvironmentJSON(fields, "available_custom_deployment_protection_rule_integrations", &raw); err != nil {
			return nil, err
		}
		if count < 0 || len(raw) > 100 || (total >= 0 && count != total) {
			return nil, fmt.Errorf("custom environment protection availability count changed or is invalid")
		}
		total = count
		for _, entry := range raw {
			app, err := decodeEnvironmentApp(entry)
			if err != nil {
				return nil, err
			}
			if _, exists := apps[app.ID]; exists || nodes[app.NodeID] || slugs[app.Slug] {
				return nil, fmt.Errorf("duplicate available custom environment protection App")
			}
			apps[app.ID] = app
			nodes[app.NodeID], slugs[app.Slug] = true, true
		}
		if len(apps) > total {
			return nil, fmt.Errorf("custom environment protection availability exceeds total")
		}
		if len(apps) == total {
			return apps, nil
		}
		if len(raw) < 100 {
			return nil, fmt.Errorf("incomplete custom environment protection availability")
		}
	}
}

func (c *EnvironmentsClient) prepareCustomApps(repo repository.Repository, name string, current []settings.EnvironmentCustomRule, desired *[]int64) (map[int64]environmentApp, error) {
	add, _, err := settings.EnvironmentCustomRuleChanges(current, desired)
	if err != nil {
		return nil, err
	}
	if len(add) == 0 {
		return nil, nil
	}
	available, err := c.availableCustomApps(repo, name)
	if err != nil {
		return nil, err
	}
	selected := map[int64]environmentApp{}
	for _, id := range add {
		app, ok := available[id]
		if !ok {
			return nil, fmt.Errorf("custom environment protection App %d is not available for this Environment", id)
		}
		selected[id] = app
	}
	return selected, nil
}

type environmentCustomRuleFailure struct {
	appID, ruleID int64
	operation     string
	cause         error
}

func (e *environmentCustomRuleFailure) Error() string {
	return fmt.Sprintf("%s custom environment protection App %d (binding rule ID %d; write may have applied, inspect before retrying): %v", e.operation, e.appID, e.ruleID, e.cause)
}
func (e *environmentCustomRuleFailure) Unwrap() error { return e.cause }

// App writes preserve the exact complete parent/pattern snapshot and every
// expected binding. New identities never become additional deletion candidates.
func sameEnvironmentAppCompanions(a, b settings.EnvironmentState) bool {
	if a.ID != b.ID || a.Policy.Name != b.Policy.Name || a.DeploymentModeNormalization != b.DeploymentModeNormalization {
		return false
	}
	ap, bp := a.Policy, b.Policy
	ap.CustomRules, bp.CustomRules = nil, nil
	if !reflect.DeepEqual(ap, bp) {
		return false
	}
	aPatterns, bPatterns := slices.Clone(a.Patterns), slices.Clone(b.Patterns)
	slices.SortFunc(aPatterns, func(a, b settings.EnvironmentPatternState) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	slices.SortFunc(bPatterns, func(a, b settings.EnvironmentPatternState) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return reflect.DeepEqual(aPatterns, bPatterns)
}

func (c *EnvironmentsClient) applyCustomApps(repo repository.Repository, target settings.ArchiveState, start settings.EnvironmentState, desired *[]int64, available map[int64]environmentApp) (bool, error) {
	add, remove, err := settings.EnvironmentCustomRuleChanges(start.CustomRules, desired)
	if err != nil {
		return false, &environmentPreWriteFailure{err}
	}
	if len(add)+len(remove) == 0 {
		return false, nil
	}
	expected := slices.Clone(start.CustomRules)
	confirmed := false
	attempted := false
	fail := func(err error) (bool, error) {
		if confirmed {
			return false, &environmentWriteEffectsFailure{err}
		}
		if !attempted {
			return false, &environmentPreWriteFailure{err}
		}
		return false, err
	}
	check := func() error {
		fresh, err := (&Client{rest: c.rest}).ReadArchive(repo)
		if err != nil || fresh != target {
			return fmt.Errorf("repository identity/archive changed before custom environment protection write")
		}
		actual, err := c.readOne(repo, start.Policy.Name)
		if err != nil {
			return err
		}
		if !sameEnvironmentAppCompanions(start, actual) || !settings.SameEnvironmentCustomRules(expected, actual.CustomRules) {
			return fmt.Errorf("Environment identity, companions or custom bindings changed")
		}
		return nil
	}
	verify := func() error {
		fresh, err := (&Client{rest: c.rest}).ReadArchive(repo)
		if err != nil || fresh != target {
			return fmt.Errorf("repository identity/archive changed during custom environment protection readback")
		}
		actual, err := c.readOne(repo, start.Policy.Name)
		if err != nil {
			return err
		}
		if !sameEnvironmentAppCompanions(start, actual) || !settings.SameEnvironmentCustomRules(expected, actual.CustomRules) {
			return fmt.Errorf("custom environment protection readback differs or changed retained companions")
		}
		return nil
	}
	for _, appID := range add {
		if err := check(); err != nil {
			return fail(err)
		}
		// Recheck availability without changing the frozen addition identities.
		fresh, err := c.availableCustomApps(repo, start.Policy.Name)
		if err != nil {
			return fail(err)
		}
		if app, ok := fresh[appID]; !ok || app != available[appID] {
			return fail(fmt.Errorf("selected custom environment protection App identity changed"))
		}
		body, _ := json.Marshal(map[string]int64{"integration_id": appID})
		attempted = true
		data, err := c.customRequest(http.MethodPost, environmentPath(repo, start.Policy.Name)+"/deployment_protection_rules", body, 201)
		if err != nil {
			recovery, _ := decodeEnvironmentCustomRule(data)
			return fail(&environmentCustomRuleFailure{appID: appID, ruleID: recovery.ID, operation: "enable", cause: err})
		}
		rule, err := decodeEnvironmentCustomRule(data)
		wrap := func(err error) error {
			return &environmentCustomRuleFailure{appID: appID, ruleID: rule.ID, operation: "enable", cause: err}
		}
		app := available[appID]
		if err != nil {
			return fail(wrap(err))
		}
		if rule.AppID != appID || rule.AppNodeID != app.NodeID || rule.Slug != app.Slug || rule.IntegrationURL != app.URL {
			return fail(wrap(fmt.Errorf("acknowledgment App identity differs")))
		}
		expected = append(expected, rule)
		if _, err := settings.EnvironmentCustomRuleIDs(expected); err != nil {
			return fail(wrap(err))
		}
		data, err = c.customRequest(http.MethodGet, fmt.Sprintf("%s/deployment_protection_rules/%d", environmentPath(repo, start.Policy.Name), rule.ID), nil, 200)
		if err != nil {
			return fail(wrap(err))
		}
		readback, err := decodeEnvironmentCustomRule(data)
		if err != nil || readback != rule {
			return fail(wrap(fmt.Errorf("independent binding readback differs or is unavailable")))
		}
		if err := verify(); err != nil {
			return fail(wrap(err))
		}
		confirmed = true
	}
	for _, rule := range remove {
		if err := check(); err != nil {
			return fail(err)
		}
		attempted = true
		_, err := c.customRequest(http.MethodDelete, fmt.Sprintf("%s/deployment_protection_rules/%d", environmentPath(repo, start.Policy.Name), rule.ID), nil, 204)
		wrap := func(err error) error {
			return &environmentCustomRuleFailure{appID: rule.AppID, ruleID: rule.ID, operation: "disable", cause: err}
		}
		if err != nil {
			return fail(wrap(err))
		}
		expected = slices.DeleteFunc(expected, func(v settings.EnvironmentCustomRule) bool { return v.ID == rule.ID })
		if err := verify(); err != nil {
			return fail(wrap(err))
		}
		confirmed = true
	}
	return confirmed, nil
}
