package settings

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

type ClassicForcePush = uimodel.ClassicForcePush
type ClassicForcePushActor = uimodel.ClassicForcePushActor

func ClassicForcePushSelected(g *ClassicForcePush) bool { return g != nil && g.Enabled != nil }

// Acquired policies may contain known Team identities; declarations may not.
func ValidateClassicForcePush(g *ClassicForcePush, acquired bool) error {
	if g == nil {
		return nil
	}
	if g.Enabled == nil {
		if g.WhoCanForcePush != nil || g.Actors != nil {
			return fmt.Errorf("classic force pushes require enabled")
		}
		return nil
	}
	if !*g.Enabled {
		if g.WhoCanForcePush != nil || g.Actors != nil {
			return fmt.Errorf("disabled classic force pushes forbid children")
		}
		return nil
	}
	if g.WhoCanForcePush == nil {
		return fmt.Errorf("enabled classic force pushes require who_can_force_push")
	}
	switch *g.WhoCanForcePush {
	case "everyone":
		if g.Actors != nil {
			return fmt.Errorf("everyone forbids force-push actors")
		}
	case "specified_actors":
		if g.Actors == nil || len(*g.Actors) == 0 {
			return fmt.Errorf("specified force pushes require nonempty actors")
		}
		seen := map[string]bool{}
		for _, a := range *g.Actors {
			if (a.Type != "user" && a.Type != "app" && !(acquired && a.Type == "team")) || strings.TrimSpace(a.ID) == "" || strings.ContainsFunc(a.ID, unicode.IsControl) || seen[a.ID] {
				return fmt.Errorf("invalid or duplicate classic force-push actor")
			}
			seen[a.ID] = true
		}
	default:
		return fmt.Errorf("unknown classic force-push mode")
	}
	return nil
}

func sortedClassicForceActors(values []ClassicForcePushActor) []ClassicForcePushActor {
	values = slices.Clone(values)
	slices.SortFunc(values, func(a, b ClassicForcePushActor) int {
		if c := cmp.Compare(a.Type, b.Type); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return values
}
func CloneClassicForcePush(g *ClassicForcePush) *ClassicForcePush {
	if g == nil {
		return nil
	}
	copy := *g
	copy.Enabled = cloneClassicValue(g.Enabled)
	copy.WhoCanForcePush = cloneClassicValue(g.WhoCanForcePush)
	if g.Actors != nil {
		values := slices.Clone(*g.Actors)
		copy.Actors = &values
	}
	return &copy
}
func ClassicForcePushEqual(a, b *ClassicForcePush) bool {
	if !ClassicForcePushSelected(a) || !ClassicForcePushSelected(b) || *a.Enabled != *b.Enabled {
		return false
	}
	if !*a.Enabled {
		return true
	}
	if a.WhoCanForcePush == nil || b.WhoCanForcePush == nil || *a.WhoCanForcePush != *b.WhoCanForcePush {
		return false
	}
	if *a.WhoCanForcePush == "everyone" {
		return true
	}
	return a.Actors != nil && b.Actors != nil && slices.Equal(sortedClassicForceActors(*a.Actors), sortedClassicForceActors(*b.Actors))
}
func ClassicForcePushProjection(all bool, raw []ClassicActor) (*ClassicForcePush, error) {
	if raw == nil {
		return nil, fmt.Errorf("incomplete classic force-push actors")
	}
	enabled := all || len(raw) > 0
	group := &ClassicForcePush{Enabled: &enabled}
	actors := []ClassicForcePushActor{}
	for _, a := range raw {
		actors = append(actors, ClassicForcePushActor{Type: strings.ToLower(a.Kind), ID: a.ID})
	}
	// Validate even latent evidence before hiding it from the active projection.
	if len(actors) > 0 {
		yes, specified := true, "specified_actors"
		if err := ValidateClassicForcePush(&ClassicForcePush{Enabled: &yes, WhoCanForcePush: &specified, Actors: &actors}, true); err != nil {
			return nil, err
		}
		for _, a := range raw {
			if a.Kind != "User" && a.Kind != "App" && a.Kind != "Team" {
				return nil, fmt.Errorf("unknown classic actor kind")
			}
		}
	}
	if all {
		mode := "everyone"
		group.WhoCanForcePush = &mode
	} else if enabled {
		mode := "specified_actors"
		group.WhoCanForcePush = &mode
		actors = sortedClassicForceActors(actors)
		group.Actors = &actors
	}
	return group, nil
}

func maskSelectedClassicForcePush(before *ClassicCompanions, after ClassicCompanions, desired *ClassicForcePush) {
	if !ClassicForcePushSelected(desired) {
		return
	}
	before.AllowsForcePushes = after.AllowsForcePushes
	// Everyone owns the boolean only; its hidden actor set must still survive.
	if !*desired.Enabled || *desired.WhoCanForcePush == "specified_actors" {
		before.ForcePushActors = after.ForcePushActors
	}
}
