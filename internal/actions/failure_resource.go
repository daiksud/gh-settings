package actions

import (
	"github.com/daiksud/gh-settings/internal/app"
	"strings"
)

// Only fixed resource paths are retained: diagnostics must never echo arbitrary
// configuration values, API errors, credentials or resource names into Summary.
var failureResources = []string{
	"repository.settings.general.danger_zone.archive_this_repository",
	"repository.settings.general.releases.immutable_release",
	"repository.settings.advanced_security.private_vulnerability_reporting",
	"repository.settings.advanced_security",
	"repository.settings.actions.general", "repository.settings.environments",
	"repository.settings.rulesets", "repository.settings.pages", "repository.settings.general",
	"repository.about.topics", "repository.about", "repository.labels", "repository.settings", "repository",
}

func reportFailureResource(failure *app.PlanError) string {
	if failure.Stage != "invalid-config" && failure.Stage != "read-failure" {
		return ""
	}
	for _, path := range failureResources {
		if failure.Path == path {
			return path
		}
	}
	if failure.Stage == "invalid-config" {
		for _, path := range failureResources {
			if strings.Contains(failure.Err.Error(), path) {
				return path
			}
		}
		return "repository"
	}
	return ""
}

func validFailureResource(outcome, resource string) bool {
	if resource == "" {
		return true
	}
	if outcome != "invalid-config" && outcome != "read-failure" {
		return false
	}
	for _, path := range failureResources {
		if resource == path {
			return true
		}
	}
	return false
}
