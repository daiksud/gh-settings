package github

import "github.com/daiksud/gh-settings/internal/settings"

func (c *PagesClient) warnPagesNormalizations(o settings.PagesObservation, d settings.PagesDeclaration) {
	if c.NormalizationWarning == nil || d.Delete() {
		return
	}
	if _, err := o.Values(); err != nil {
		return
	}
	if record := o.ModeNormalization(); d.BuildType != nil && record != (settings.ObservationNormalization{}) {
		c.NormalizationWarning(record)
	}
	if record := o.FolderNormalization(); d.Source != nil && record != (settings.ObservationNormalization{}) {
		c.NormalizationWarning(record)
	}
}
