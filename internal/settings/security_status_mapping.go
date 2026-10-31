package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

type SecurityStatusControl string

const (
	SecretProtectionControl SecurityStatusControl = "secret_protection"
	PushProtectionControl   SecurityStatusControl = "push_protection"
)

func (control SecurityStatusControl) Path() string {
	switch control {
	case SecretProtectionControl:
		return "repository.settings.advanced_security.secret_protection.enabled"
	case PushProtectionControl:
		return "repository.settings.advanced_security.secret_protection.push_protection"
	default:
		return ""
	}
}

func (control SecurityStatusControl) Field() string {
	switch control {
	case SecretProtectionControl:
		return "security.secret_scanning"
	case PushProtectionControl:
		return "security.secret_scanning_push_protection"
	default:
		return ""
	}
}

func (control SecurityStatusControl) DefaultValue() (bool, bool) {
	switch control {
	case SecretProtectionControl:
		return uimodel.DefaultSecretProtection, true
	case PushProtectionControl:
		return uimodel.DefaultPushProtection, true
	default:
		return false, false
	}
}

func SecurityValueForStatus(status string) (bool, bool) {
	switch status {
	case "enabled":
		return true, true
	case "disabled":
		return false, true
	default:
		return false, false
	}
}

func SecurityStatusForValue(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}
