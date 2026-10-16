package main

import "github.com/daiksud/gh-settings/internal/app"

func exportCurrent() ([]byte, error) {
	return nil, app.Export(readRepository)
}
