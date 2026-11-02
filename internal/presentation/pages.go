package presentation

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strconv"
)

func PagesLine(c settings.PagesChange) string {
	b, a := c.Values()
	if c.Field() == "pages.lifecycle" {
		if a == "absent" {
			return "repository.settings.pages: delete site (source: deploy_from_a_branch, branch: none)"
		}
		return "repository.settings.pages: create site"
	}
	if record := c.Normalization(); record != (settings.ObservationNormalization{}) {
		value, _ := a.(string)
		choice := ""
		if c.Field() == "pages.build_type" {
			mode, _ := settings.PagesModeChoiceForValue(value)
			choice = string(mode)
		} else {
			folder, _ := settings.PagesFolderChoiceForValue(value)
			choice = string(folder)
		}
		line := NormalizationLine(record) + "\n" + fmt.Sprintf("%s: %s (normalized observation) → %s (write required)", record.Path, strconv.Quote(record.Fallback), strconv.Quote(choice))
		if c.Field() == "pages.build_type" && value == "workflow" {
			line += " (branch source becomes inactive)"
		}
		return line
	}
	b = pagesDisplayValue(c.Field(), b)
	a = pagesDisplayValue(c.Field(), a)
	before, _ := json.Marshal(b)
	after, _ := json.Marshal(a)
	if c.BeforeAbsent() {
		before = []byte("absent")
	}
	if c.BeforeInactive() {
		before = []byte("inactive")
	}
	line := fmt.Sprintf("%s: %s → %s", SettingPath(c.Field()), before, after)
	if c.Field() == "pages.build_type" && a == "github_actions" {
		line += " (branch source becomes inactive)"
	}
	return line
}

func pagesDisplayValue(field string, value any) any {
	v, ok := value.(string)
	if !ok {
		return value
	}
	switch field {
	case "pages.build_type":
		if choice, valid := settings.PagesModeChoiceForValue(v); valid {
			return string(choice)
		}
	case "pages.source.path":
		if choice, valid := settings.PagesFolderChoiceForValue(v); valid {
			return string(choice)
		}
	}
	return value
}
