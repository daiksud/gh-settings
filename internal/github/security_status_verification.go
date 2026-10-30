package github

import "github.com/daiksud/gh-settings/internal/settings"

func sameRawSecretScanningStatuses(expected, actual settings.SecretScanningObservation) (bool, error) {
	scanning, push, err := expected.RawStatuses()
	if err != nil {
		return false, err
	}
	actualScanning, actualPush, err := actual.RawStatuses()
	if err != nil {
		return false, err
	}
	return scanning == actualScanning && push == actualPush, nil
}
