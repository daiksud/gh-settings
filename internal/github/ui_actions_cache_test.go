package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"testing"
)

func TestActionsCacheUIAdapter(t *testing.T) {
	days, gb := 7, 10
	native := settings.ActionsPolicyDeclaration{CacheRetentionDays: &days, CacheSizeGB: &gb}
	ui, err := WithActionsCachePolicyUI(uimodel.ActionsGeneral{}, native)
	if err != nil {
		t.Fatal(err)
	}
	if ui.Cache == nil || *ui.Cache.CacheRetention != 7 || *ui.Cache.CacheSizeEvictionLimit != 10 {
		t.Fatalf("wrong UI %#v", ui)
	}
	got, err := ActionsDeclaration(ui)
	if err != nil {
		t.Fatal(err)
	}
	if *got.CacheRetentionDays != 7 || *got.CacheSizeGB != 10 {
		t.Fatalf("wrong native %#v", got)
	}
	*ui.Cache.CacheRetention = 8
	if *native.CacheRetentionDays != 7 || *got.CacheRetentionDays != 7 {
		t.Fatal("pointer alias")
	}
	empty, err := WithActionsCachePolicyUI(uimodel.ActionsGeneral{}, settings.ActionsPolicyDeclaration{})
	if err != nil || empty.Cache != nil {
		t.Fatal("omission lost")
	}
	for _, value := range []int{0, 366} {
		if _, err := ActionsDeclaration(uimodel.ActionsGeneral{Cache: &uimodel.ActionsCache{CacheRetention: &value}}); err == nil {
			t.Fatal("invalid range accepted")
		}
	}
}
