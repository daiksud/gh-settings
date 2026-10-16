package main

import "strings"

const settingsPath = ".github/settings.yml"

func repositoryYAML(fields string) string {
	return "schemaVersion: 1\nsettings:\n  repository:\n    " + strings.ReplaceAll(strings.TrimSuffix(fields, "\n"), "\n", "\n    ") + "\n"
}
