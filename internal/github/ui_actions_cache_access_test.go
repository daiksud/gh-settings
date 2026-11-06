package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"testing"
)

func TestActionsCacheAccessUIAdapter(t *testing.T) {
	days, gb, level := 7, 10, "user"
	native := settings.ActionsPolicyDeclaration{CacheRetentionDays: &days, CacheSizeGB: &gb, AccessLevel: &level}
	ui, err := WithActionsCacheAccessPolicyUI(uimodel.ActionsGeneral{}, native)
	if err != nil {
		t.Fatal(err)
	}
	if ui.Cache == nil || *ui.Cache.CacheRetention != 7 || *ui.Cache.CacheSizeEvictionLimit != 10 || ui.Access == nil || *ui.Access != uimodel.AccessibleFromRepositoriesOwnedByUser {
		t.Fatalf("wrong UI %#v", ui)
	}
	got, err := ActionsDeclaration(ui)
	if err != nil {
		t.Fatal(err)
	}
	if *got.CacheRetentionDays != 7 || *got.CacheSizeGB != 10 || *got.AccessLevel != "user" {
		t.Fatalf("wrong native %#v", got)
	}
	*ui.Cache.CacheRetention = 8
	if *native.CacheRetentionDays != 7 || *got.CacheRetentionDays != 7 {
		t.Fatal("pointer alias")
	}
	empty, err := WithActionsCacheAccessPolicyUI(uimodel.ActionsGeneral{}, settings.ActionsPolicyDeclaration{})
	if err != nil || empty.Cache != nil || empty.Access != nil {
		t.Fatal("omission lost")
	}
	for _, choice := range []uimodel.ActionsAccessChoice{"none", "user", "organization", "enterprise"} {
		if _, err := ActionsDeclaration(uimodel.ActionsGeneral{Access: &choice}); err == nil {
			t.Fatal("API alias accepted")
		}
	}
	for _, value := range []int{0, 366} {
		if _, err := ActionsDeclaration(uimodel.ActionsGeneral{Cache: &uimodel.ActionsCache{CacheRetention: &value}}); err == nil {
			t.Fatal("invalid range accepted")
		}
	}
}
