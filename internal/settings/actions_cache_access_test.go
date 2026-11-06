package settings

import (
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"math"
	"testing"
)

func TestCacheRetentionObservationAndComparison(t *testing.T) {
	if _, _, err := (CacheRetentionObservation{}).Values(); err == nil {
		t.Fatal("unacquired accepted")
	}
	for _, days := range []int{0, -1, 366} {
		if _, err := NewCacheRetentionObservation(days, "private"); err == nil {
			t.Fatalf("accepted %d", days)
		}
	}
	if _, err := NewCacheRetentionObservation(7, "unknown"); err == nil {
		t.Fatal("unknown visibility")
	}
	for _, visibility := range []string{"public", "private", "internal"} {
		o, err := NewCacheRetentionObservation(7, visibility)
		if err != nil {
			t.Fatal(err)
		}
		for _, days := range []int{1, 90, 91, 365} {
			n, changes, err := CompareCacheRetention(o, ActionsPolicyDeclaration{CacheRetentionDays: &days})
			if visibility == "public" && days > 90 {
				if err == nil {
					t.Fatal("public cap ignored")
				}
				continue
			}
			if err != nil || n != 1 || len(changes) != 1 {
				t.Fatalf("compare %s %d: %d %v %v", visibility, days, n, changes, err)
			}
		}
		n, c, err := CompareCacheRetention(o, ActionsPolicyDeclaration{})
		if err != nil || n != 0 || len(c) != 0 {
			t.Fatal("omission managed")
		}
	}
	o, _ := NewCacheRetentionObservation(100, "public")
	if _, err := o.Declaration(); err == nil {
		t.Fatal("invalid export accepted")
	}
}
func TestCacheStorageObservationAndComparison(t *testing.T) {
	if _, err := (CacheStorageObservation{}).Value(); err == nil {
		t.Fatal("unacquired accepted")
	}
	for _, gb := range []int{0, -1, 10001} {
		if _, err := NewCacheStorageObservation(gb); err == nil {
			t.Fatalf("accepted %d", gb)
		}
	}
	for _, gb := range []int{1, 10, 10000} {
		o, err := NewCacheStorageObservation(gb)
		if err != nil {
			t.Fatal(err)
		}
		d, err := o.Declaration()
		if err != nil {
			t.Fatal(err)
		}
		n, c, err := CompareCacheStorage(o, d)
		if err != nil || n != 1 || len(c) != 0 {
			t.Fatal("equal comparison failed")
		}
		if d.Empty() || !d.ExecutionEmpty() || !d.RetentionEmpty() || !d.AccessEmpty() {
			t.Fatal("independent ownership failed")
		}
	}
}
func TestActionsAccessStrictMappingAndOwnership(t *testing.T) {
	if _, _, _, err := (ActionsAccessObservation{}).Values(); err == nil {
		t.Fatal("unacquired accepted")
	}
	for _, pair := range []struct{ level, owner string }{{"none", "User"}, {"user", "User"}, {"none", "Organization"}, {"organization", "Organization"}} {
		o, err := NewActionsAccessObservation(pair.level, "private", pair.owner)
		if err != nil {
			t.Fatal(err)
		}
		d, err := o.Declaration()
		if err != nil {
			t.Fatal(err)
		}
		n, c, err := CompareActionsAccess(o, d)
		if err != nil || n != 1 || len(c) != 0 {
			t.Fatal("equal comparison failed")
		}
		choice, ok := ActionsAccessChoiceForValue(pair.level)
		if !ok {
			t.Fatal("missing mapping")
		}
		value, ok := ActionsAccessValueForChoice(choice)
		if !ok || value != pair.level {
			t.Fatal("mapping changed")
		}
		wrong := "user"
		if pair.owner == "User" {
			wrong = "organization"
		}
		if _, _, err := CompareActionsAccess(o, ActionsPolicyDeclaration{AccessLevel: &wrong}); err == nil {
			t.Fatal("owner mismatch accepted")
		}
	}
	for _, visibility := range []string{"public", "internal", "unknown"} {
		if _, err := NewActionsAccessObservation("none", visibility, "User"); err == nil {
			t.Fatal("nonprivate accepted")
		}
	}
	for _, level := range []string{"enterprise", "", "not_accessible"} {
		if _, err := NewActionsAccessObservation(level, "private", "User"); err == nil {
			t.Fatal("unknown native accepted")
		}
	}
	for _, choice := range []uimodel.ActionsAccessChoice{"none", "user", "organization", "unknown"} {
		if _, ok := ActionsAccessValueForChoice(choice); ok {
			t.Fatal("native alias accepted")
		}
	}
}
func TestCacheAccessChangesClosedValues(t *testing.T) {
	for _, field := range []string{"actions.cache_retention_days", "actions.cache_size_gb"} {
		for _, value := range []any{nil, "7", true, 0, -1, 10001, 1.5, math.NaN(), math.Inf(1)} {
			if _, err := NewActionsPolicyChange(field, 7, value); err == nil {
				t.Fatalf("accepted %s %v", field, value)
			}
		}
		if _, err := NewActionsPolicyChange(field, nil, 10); err == nil {
			t.Fatal("unobserved accepted")
		}
		if _, err := NewActionsPolicyChange(field, 7, float64(10)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NewActionsPolicyChange("actions.access_level", "none", "enterprise"); err == nil {
		t.Fatal("unknown access accepted")
	}
}
