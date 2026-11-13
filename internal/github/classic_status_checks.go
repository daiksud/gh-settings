package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func decodeClassicStatusChecks(m map[string]json.RawMessage, s *settings.ClassicState) error {
	c := &s.Companions
	var err error
	c.RequiresStatusChecks, err = classicBool(m, "requiresStatusChecks")
	if err != nil {
		return err
	}
	c.RequiresStrictStatusChecks, err = classicBool(m, "requiresStrictStatusChecks")
	if err != nil {
		return err
	}
	c.Contexts, err = classicStringList(m["requiredStatusCheckContexts"])
	if err != nil {
		return err
	}
	checks, err := classicNullable[[]json.RawMessage](m["requiredStatusChecks"])
	if err != nil {
		return err
	}
	c.Checks.Present = true
	if checks.Value != nil {
		values := make([]settings.ClassicCheck, 0, len(*checks.Value))
		for _, raw := range *checks.Value {
			m, e := classicObject(raw, "context", "app")
			if e != nil {
				return e
			}
			v := settings.ClassicCheck{}
			var context *string
			if json.Unmarshal(m["context"], &context) != nil || context == nil {
				return fmt.Errorf("missing or invalid classic check context")
			}
			v.Context = *context
			if !bytes.Equal(bytes.TrimSpace(m["app"]), []byte("null")) {
				a, e := classicObject(m["app"], "id")
				if e != nil {
					return e
				}
				id, e := accessString(a, "id")
				if e != nil {
					return e
				}
				v.AppID = &id
			}
			values = append(values, v)
		}
		c.Checks.Value = &values
	}
	s.Rule.RequireStatusChecksToPassBeforeMerging, err = settings.ProjectClassicStatusChecks(*c)
	return err
}

const classicStatusAppQuery = `query ClassicStatusApp($id:ID!){node(id:$id){__typename ... on App{id}}}`

func (c *ClassicProtectionClient) verifyStatusApps(group *settings.ClassicStatusChecks) error {
	for _, id := range settings.ClassicStatusCheckAppIDs(group) {
		var response json.RawMessage
		if err := c.graphql.Do(classicStatusAppQuery, map[string]any{"id": id}, &response); err != nil {
			return fmt.Errorf("classic status-check App identity unavailable")
		}
		top, err := classicObject(response, "node")
		if err != nil {
			return fmt.Errorf("classic status-check App identity incomplete")
		}
		node, err := classicObject(top["node"], "__typename", "id")
		if err != nil {
			return fmt.Errorf("classic status-check App identity incomplete")
		}
		kind, kindErr := accessString(node, "__typename")
		actual, idErr := accessString(node, "id")
		if kindErr != nil || idErr != nil || kind != "App" || actual != id {
			return fmt.Errorf("classic status-check App identity differs")
		}
	}
	return nil
}
