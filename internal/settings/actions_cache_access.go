package settings

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const (
	CacheRetentionPath = "repository.settings.actions.general.cache.cache_retention"
	CacheSizePath      = "repository.settings.actions.general.cache.cache_size_eviction_limit"
	ActionsAccessPath  = "repository.settings.actions.general.access"
)

func ValidateCacheRetentionDays(days int) error {
	if days < 1 || days > 365 {
		return fmt.Errorf("%s must be an integer from 1 through 365", CacheRetentionPath)
	}
	return nil
}
func ValidateCacheSizeGB(gb int) error {
	if gb < 1 || gb > 10000 {
		return fmt.Errorf("%s must be an integer from 1 through 10000", CacheSizePath)
	}
	return nil
}
func ActionsAccessChoiceForValue(value string) (uimodel.ActionsAccessChoice, bool) {
	switch value {
	case "none":
		return uimodel.NotAccessible, true
	case "user":
		return uimodel.AccessibleFromRepositoriesOwnedByUser, true
	case "organization":
		return uimodel.AccessibleFromRepositoriesInOrganization, true
	}
	return "", false
}
func ActionsAccessValueForChoice(choice uimodel.ActionsAccessChoice) (string, bool) {
	switch choice {
	case uimodel.NotAccessible:
		return "none", true
	case uimodel.AccessibleFromRepositoriesOwnedByUser:
		return "user", true
	case uimodel.AccessibleFromRepositoriesInOrganization:
		return "organization", true
	}
	return "", false
}
func ValidActionsAccessLevel(value string) bool {
	_, valid := ActionsAccessChoiceForValue(value)
	return valid
}
func ValidateActionsAccessLevel(value string) error {
	if !ValidActionsAccessLevel(value) {
		return fmt.Errorf("%s must be a supported choice", ActionsAccessPath)
	}
	return nil
}
func (d ActionsPolicyDeclaration) CacheRetentionEmpty() bool { return d.CacheRetentionDays == nil }
func (d ActionsPolicyDeclaration) CacheStorageEmpty() bool   { return d.CacheSizeGB == nil }
func (d ActionsPolicyDeclaration) AccessEmpty() bool         { return d.AccessLevel == nil }

// CacheRetentionObservation preserves acquired visibility as an unowned limit.
// A repository visibility change can leave configured retention above its current
// limit; preparation and export validate desired days against that limit.
type CacheRetentionObservation struct {
	acquired   bool
	days       int
	visibility string
}

func NewCacheRetentionObservation(days int, visibility string) (CacheRetentionObservation, error) {
	if err := ValidateCacheRetentionDays(days); err != nil {
		return CacheRetentionObservation{}, err
	}
	if visibility != "public" && visibility != "private" && visibility != "internal" {
		return CacheRetentionObservation{}, fmt.Errorf("cache retention requires a known repository visibility")
	}
	return CacheRetentionObservation{true, days, visibility}, nil
}
func (o CacheRetentionObservation) Values() (int, string, error) {
	if !o.acquired {
		return 0, "", fmt.Errorf("cache retention observation was not acquired")
	}
	return o.days, o.visibility, nil
}
func PrepareCacheRetention(o CacheRetentionObservation, days int) error {
	if err := ValidateCacheRetentionDays(days); err != nil {
		return err
	}
	_, visibility, err := o.Values()
	if err != nil {
		return err
	}
	if visibility == "public" && days > 90 {
		return fmt.Errorf("%s exceeds public repository maximum of 90 days", CacheRetentionPath)
	}
	return nil
}
func (o CacheRetentionObservation) Declaration() (ActionsPolicyDeclaration, error) {
	days, _, err := o.Values()
	if err != nil {
		return ActionsPolicyDeclaration{}, err
	}
	if err := PrepareCacheRetention(o, days); err != nil {
		return ActionsPolicyDeclaration{}, err
	}
	return ActionsPolicyDeclaration{CacheRetentionDays: &days}, nil
}
func CompareCacheRetention(o CacheRetentionObservation, d ActionsPolicyDeclaration) (int, []ActionsPolicyChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	before, _, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	if d.CacheRetentionDays == nil {
		return 0, []ActionsPolicyChange{}, nil
	}
	if err := PrepareCacheRetention(o, *d.CacheRetentionDays); err != nil {
		return 0, nil, err
	}
	return compareActionsGeneralValue("actions.cache_retention_days", before, *d.CacheRetentionDays)
}

// CacheStorageObservation distinguishes configured capacity from an absent read.
type CacheStorageObservation struct {
	acquired bool
	gb       int
}

func NewCacheStorageObservation(gb int) (CacheStorageObservation, error) {
	if err := ValidateCacheSizeGB(gb); err != nil {
		return CacheStorageObservation{}, err
	}
	return CacheStorageObservation{true, gb}, nil
}
func (o CacheStorageObservation) Value() (int, error) {
	if !o.acquired {
		return 0, fmt.Errorf("cache storage observation was not acquired")
	}
	return o.gb, nil
}
func (o CacheStorageObservation) Declaration() (ActionsPolicyDeclaration, error) {
	gb, err := o.Value()
	if err != nil {
		return ActionsPolicyDeclaration{}, err
	}
	return ActionsPolicyDeclaration{CacheSizeGB: &gb}, nil
}
func CompareCacheStorage(o CacheStorageObservation, d ActionsPolicyDeclaration) (int, []ActionsPolicyChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	before, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	if d.CacheSizeGB == nil {
		return 0, []ActionsPolicyChange{}, nil
	}
	return compareActionsGeneralValue("actions.cache_size_gb", before, *d.CacheSizeGB)
}

// ActionsAccessObservation retains the private repository and owner context
// needed to validate the sharing choice before comparison or mutation.
type ActionsAccessObservation struct {
	acquired                     bool
	level, visibility, ownerType string
}

func NewActionsAccessObservation(level, visibility, ownerType string) (ActionsAccessObservation, error) {
	if err := ValidateActionsAccessLevel(level); err != nil {
		return ActionsAccessObservation{}, err
	}
	if visibility != "private" {
		return ActionsAccessObservation{}, fmt.Errorf("%s requires a private repository", ActionsAccessPath)
	}
	if ownerType != "User" && ownerType != "Organization" {
		return ActionsAccessObservation{}, fmt.Errorf("%s requires a known repository owner type", ActionsAccessPath)
	}
	o := ActionsAccessObservation{true, level, visibility, ownerType}
	if err := PrepareActionsAccess(o, level); err != nil {
		return ActionsAccessObservation{}, err
	}
	return o, nil
}
func (o ActionsAccessObservation) Values() (string, string, string, error) {
	if !o.acquired {
		return "", "", "", fmt.Errorf("Actions access observation was not acquired")
	}
	return o.level, o.visibility, o.ownerType, nil
}
func PrepareActionsAccess(o ActionsAccessObservation, level string) error {
	if err := ValidateActionsAccessLevel(level); err != nil {
		return err
	}
	_, _, owner, err := o.Values()
	if err != nil {
		return err
	}
	if (level == "user" && owner != "User") || (level == "organization" && owner != "Organization") {
		return fmt.Errorf("%s choice is incompatible with repository owner type %s", ActionsAccessPath, owner)
	}
	return nil
}
func (o ActionsAccessObservation) Declaration() (ActionsPolicyDeclaration, error) {
	level, _, _, err := o.Values()
	if err != nil {
		return ActionsPolicyDeclaration{}, err
	}
	return ActionsPolicyDeclaration{AccessLevel: &level}, nil
}
func CompareActionsAccess(o ActionsAccessObservation, d ActionsPolicyDeclaration) (int, []ActionsPolicyChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	before, _, _, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	if d.AccessLevel == nil {
		return 0, []ActionsPolicyChange{}, nil
	}
	if err := PrepareActionsAccess(o, *d.AccessLevel); err != nil {
		return 0, nil, err
	}
	return compareActionsGeneralValue("actions.access_level", before, *d.AccessLevel)
}
func compareActionsGeneralValue[T comparable](field string, before, after T) (int, []ActionsPolicyChange, error) {
	changes := []ActionsPolicyChange{}
	if before != after {
		c, err := NewActionsPolicyChange(field, before, after)
		if err != nil {
			return 0, nil, err
		}
		changes = append(changes, c)
	}
	return 1, changes, nil
}
