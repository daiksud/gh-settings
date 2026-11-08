package settings

import "testing"

func TestAIScanPullRequestsObservationAndIndependentDeclaration(t *testing.T) {
	if _, err := (AIScanPullRequestsObservation{}).Value(); err == nil {
		t.Fatal("unacquired state became false")
	}
	for _, current := range []bool{false, true} {
		for _, desired := range []*bool{nil, securityBool(false), securityBool(true)} {
			d := SecurityDeclaration{AIScanPullRequests: desired}
			if !d.SecretScanningEmpty() || d.Empty() != (desired == nil) {
				t.Fatal("family selection coupled", d)
			}
			managed, changes, err := CompareAIScanPullRequests(NewAIScanPullRequestsObservation(current), d)
			wantManaged, wantChanges := 0, 0
			if desired != nil {
				wantManaged = 1
				if current != *desired {
					wantChanges = 1
				}
			}
			if err != nil || managed != wantManaged || len(changes) != wantChanges {
				t.Fatal(current, desired, managed, changes, err)
			}
			if len(changes) > 0 {
				before, after := changes[0].Values()
				if before != current || after != *desired || changes[0].Field() != "security.ai_scan_pull_requests" {
					t.Fatal(changes)
				}
			}
		}
	}
	secret, _ := NewSecurityChange("security.secret_scanning", true, false)
	reporting, _ := NewSecurityChange("security.ai_scan_pull_requests", false, true)
	if err := ValidateSecurityChanges([]SecurityChange{secret, reporting}); err != nil {
		t.Fatal("independent desired values coupled", err)
	}
	if _, err := AfterSecretScanningChange(NewSecretScanningObservation(true, false), reporting); err == nil {
		t.Fatal("PVR entered Secret scanning transition")
	}
}
