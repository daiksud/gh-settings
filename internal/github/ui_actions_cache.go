package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func WithActionsCachePolicyUI(ui uimodel.ActionsGeneral, policy settings.ActionsPolicyDeclaration) (uimodel.ActionsGeneral, error) {
	if policy.AccessLevel != nil || !policy.PrivateFork.Empty() {
		return uimodel.ActionsGeneral{}, fmt.Errorf("private-only Actions controls are unsupported")
	}
	if err := policy.Validate(); err != nil {
		return uimodel.ActionsGeneral{}, err
	}
	if policy.CacheRetentionDays != nil || policy.CacheSizeGB != nil {
		cache := uimodel.ActionsCache{}
		if ui.Cache != nil {
			cache = *ui.Cache
		}
		if policy.CacheRetentionDays != nil {
			cache.CacheRetention = copyUIActionsValue(policy.CacheRetentionDays)
		}
		if policy.CacheSizeGB != nil {
			cache.CacheSizeEvictionLimit = copyUIActionsValue(policy.CacheSizeGB)
		}
		ui.Cache = &cache
	}
	return ui, nil
}
