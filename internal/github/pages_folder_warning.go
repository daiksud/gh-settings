package github

import "github.com/daiksud/gh-settings/internal/settings"

func (c *PagesClient) warnFolderNormalization(o settings.PagesObservation, d settings.PagesDeclaration) {
	if d.Source == nil || c.NormalizationWarning == nil {
		return
	}
	if _, err := o.Values(); err != nil {
		return
	}
	if record := o.FolderNormalization(); record != (settings.ObservationNormalization{}) {
		c.NormalizationWarning(record)
	}
}
