package github

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func (c *ClassicProtectionClient) classicForcePushIdentities(group *settings.ClassicForcePush) error {
	if !settings.ClassicForcePushSelected(group) || !*group.Enabled || *group.WhoCanForcePush != "specified_actors" {
		return nil
	}
	values := *group.Actors
	for start := 0; start < len(values); start += 100 {
		end := min(start+100, len(values))
		ids := []string{}
		expected := map[string]string{}
		for _, a := range values[start:end] {
			ids = append(ids, a.ID)
			kind := "User"
			if a.Type == "app" {
				kind = "App"
			}
			expected[a.ID] = kind
		}
		var raw json.RawMessage
		if err := c.graphql.Do(`query ClassicForcePushIdentities($ids:[ID!]!){nodes(ids:$ids){id __typename}}`, map[string]any{"ids": ids}, &raw); err != nil {
			// Upstream messages and malformed member names can contain private actor
			// identities. Only bounded lookup diagnostics may reach public output.
			return fmt.Errorf("classic force-push actor identities unavailable")
		}
		top, err := classicObject(raw, "nodes")
		if err != nil {
			return fmt.Errorf("incomplete classic force-push identity response")
		}
		var nodes *[]json.RawMessage
		if json.Unmarshal(top["nodes"], &nodes) != nil || nodes == nil || len(*nodes) != len(ids) {
			return fmt.Errorf("incomplete classic force-push identity response")
		}
		for _, node := range *nodes {
			m, err := classicObject(node, "id", "__typename")
			if err != nil {
				return fmt.Errorf("incomplete classic force-push identity response")
			}
			id, err := accessString(m, "id")
			if err != nil {
				return fmt.Errorf("incomplete classic force-push identity response")
			}
			kind, err := accessString(m, "__typename")
			if err != nil {
				return fmt.Errorf("incomplete classic force-push identity response")
			}
			if expected[id] == "" || expected[id] != kind {
				return fmt.Errorf("classic force-push actor identity differs")
			}
			delete(expected, id)
		}
	}
	return nil
}

func classicForcePushInputs(input map[string]any, before *settings.ClassicState, group *settings.ClassicForcePush) {
	if !settings.ClassicForcePushSelected(group) {
		return
	}
	if !*group.Enabled {
		if before == nil || before.Companions.AllowsForcePushes || len(before.Companions.ForcePushActors) > 0 {
			input["allowsForcePushes"] = false
			input["bypassForcePushActorIds"] = []string{}
		}
		return
	}
	if *group.WhoCanForcePush == "everyone" {
		if before == nil || !before.Companions.AllowsForcePushes {
			input["allowsForcePushes"] = true
		}
		return
	}
	ids := []string{}
	for _, a := range *group.Actors {
		ids = append(ids, a.ID)
	}
	if before == nil || before.Companions.AllowsForcePushes || len(before.Companions.ForcePushActors) == 0 {
		input["allowsForcePushes"] = false
		input["bypassForcePushActorIds"] = ids
	} else if !settings.ClassicForcePushEqual(before.Rule.AllowForcePushes, group) {
		input["bypassForcePushActorIds"] = ids
	}
}

// Actor connections cannot be reconstructed from the scalar mutation ACK.
// Full membership is confirmed only by the independent paginated rule read.
func classicAcknowledgmentMatches(ack settings.ClassicState, desired settings.ClassicRule) bool {
	if g := desired.AllowForcePushes; settings.ClassicForcePushSelected(g) {
		want := *g.Enabled && *g.WhoCanForcePush == "everyone"
		if ack.Companions.AllowsForcePushes != want {
			return false
		}
	}
	desired.AllowForcePushes = nil
	return settings.ClassicMatches(ack.Rule, desired)
}
