package main

import "strings"

func repositoryYAML(fields string) string {
	return "schemaVersion: 1\nsettings:\n  repository:\n    " + strings.ReplaceAll(strings.TrimSuffix(fields, "\n"), "\n", "\n    ") + "\n"
}
