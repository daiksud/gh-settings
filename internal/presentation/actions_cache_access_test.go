package presentation

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestActionsCacheAccessPresentation(t *testing.T) {
	for _, tc := range []struct {
		field         string
		before, after any
		want          string
	}{
		{"actions.cache_retention_days", 7, 14, "repository.settings.actions.general.cache.cache_retention: 7 → 14"},
		{"actions.cache_size_gb", 10, 20, "repository.settings.actions.general.cache.cache_size_eviction_limit: 10 → 20"},
		{"actions.access_level", "none", "user", "repository.settings.actions.general.access: \"not_accessible\" → \"accessible_from_repositories_owned_by_user\""},
	} {
		c, err := settings.NewActionsPolicyChange(tc.field, tc.before, tc.after)
		if err != nil {
			t.Fatal(err)
		}
		if got := ActionsPolicyLine(c); got != tc.want {
			t.Fatalf("got %s want %s", got, tc.want)
		}
	}
}
