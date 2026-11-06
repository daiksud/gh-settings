package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func WithActionsCacheAccessPolicyUI(ui uimodel.ActionsGeneral, policy settings.ActionsPolicyDeclaration) (uimodel.ActionsGeneral, error) {
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
	if policy.AccessLevel != nil {
		choice, _ := settings.ActionsAccessChoiceForValue(*policy.AccessLevel)
		ui.Access = &choice
	}
	return ui, nil
}
