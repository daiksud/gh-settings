package actions

import (
	"encoding/json"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestDeployKeyReportValidation(t *testing.T) {
	key, _ := settings.ParseDeployPublicKey("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB")
	display := settings.DeployKeyDisplayFor(settings.DeployKeyDeclaration{Title: "same <title>", Key: key, AllowWriteAccess: false})
	entry := DeployKeyPlanEntry{Operation: "create", After: &display}
	rows, err := validateDeployKeyReport([]DeployKeyPlanEntry{entry})
	if err != nil || len(rows) != 1 {
		t.Fatal(err)
	}
	data, _ := json.Marshal(entry)
	if strings.Contains(string(data), key.Canonical()) || strings.Contains(string(data), "created_at") {
		t.Fatal("private state in report")
	}
	for _, bad := range []string{
		`{"operation":"create","after":{"title":"x","fingerprint":"` + key.Fingerprint() + `"}}`,
		`{"operation":"create","after":{"title":"x","fingerprint":"` + key.Fingerprint() + `","allow_write_access":null}}`,
		`{"operation":"create","after":{"title":"x","fingerprint":"` + key.Fingerprint() + `","allow_write_access":false,"key":"private"}}`,
		`{"operation":"update","after":{"title":"x","fingerprint":"` + key.Fingerprint() + `","allow_write_access":false}}`,
		`{"operation":"delete","before":{"title":"x","fingerprint":"` + key.Fingerprint() + `","allow_write_access":false}}`,
		`{"operation":"create","after":{"id":4,"title":"x","fingerprint":"` + key.Fingerprint() + `","allow_write_access":false}}`,
		`{"operation":"create","after":{"title":"x","fingerprint":"SHA256:bad","allow_write_access":false}}`,
		`{"operation":"create","after":{"title":"x","fingerprint":"` + key.Fingerprint() + `","allow_write_access":false},"private_snapshot":{}}`,
	} {
		var row DeployKeyPlanEntry
		if err := json.Unmarshal([]byte(bad), &row); err != nil {
			continue
		}
		if _, err := validateDeployKeyReport([]DeployKeyPlanEntry{row}); err == nil {
			t.Fatal("invalid report accepted", bad)
		}
	}
	if _, err := validateDeployKeyReport([]DeployKeyPlanEntry{entry, entry}); err == nil {
		t.Fatal("duplicate identity")
	}
	// JSON report boundaries retain a numeric ID exactly, including beyond float64 precision.
	var exact DeployKeyPlanEntry
	raw := `{"operation":"delete","before":{"id":9007199254740993,"title":"","fingerprint":"` + key.Fingerprint() + `","allow_write_access":false}}`
	if err := json.Unmarshal([]byte(raw), &exact); err != nil {
		t.Fatal(err)
	}
	changes, err := validateDeployKeyReport([]DeployKeyPlanEntry{exact})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := changes[0].Values()
	if *before.ID != 9007199254740993 {
		t.Fatal("rounded identity")
	}
	report := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, DeployKeys: []DeployKeyPlanEntry{entry, exact}}
	if _, err := validatePlanReport(report, "owner/current"); err == nil {
		t.Fatal("same material cannot create and delete together")
	}
	report.DeployKeys = []DeployKeyPlanEntry{entry}
	if _, err := validatePlanReport(report, "owner/current"); err != nil {
		t.Fatal(err)
	}
	report.Managed = 0
	if _, err := validatePlanReport(report, "owner/current"); err == nil {
		t.Fatal("collection uncounted")
	}
}

func TestDeployKeyAutolinkOIDCAndClassicDeletionReportCounts(t *testing.T) {
	key, err := settings.ParseDeployPublicKey("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB")
	if err != nil {
		t.Fatal(err)
	}
	display := settings.DeployKeyDisplayFor(settings.DeployKeyDeclaration{Title: "release key", Key: key})
	id, yes, no := int64(7), true, false
	classic := settings.ClassicRule{ID: &id, BranchNamePattern: "old/*", RequireConversationResolutionBeforeMerging: &no, RequireSignedCommits: &no, RequireLinearHistory: &no, AllowDeletions: &no, LockBranch: &no, DoNotAllowBypassingTheAboveSettings: &no, RequireDeploymentsToSucceedBeforeMerging: &settings.ClassicDeployments{Enabled: &no}, RequireAPullRequestBeforeMerging: &settings.ClassicPullRequestReviews{Enabled: &no}, RequireStatusChecksToPassBeforeMerging: &settings.ClassicStatusChecks{Enabled: &no}, AllowForcePushes: &settings.ClassicForcePush{Enabled: &no}}
	claims := []string{"repo", "context"}
	report := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 4, Changes: []PlanChange{}, DeployKeys: []DeployKeyPlanEntry{{Operation: "create", After: &display}}, OIDCSubject: []PlanChange{{Field: settings.OIDCSubjectPath, Before: settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes}, After: settings.OIDCSubjectTemplate{UseDefaultTemplate: &no, SubjectClaimTemplate: &claims}}}, BranchProtectionRules: []settings.ClassicChange{{Before: &classic}}}
	report.Autolinks = []settings.AutolinkChange{{After: &settings.Autolink{ReferencePrefix: "EXT-", TargetURL: "https://example.test/<num>", ReferenceFormat: "numeric"}}}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PlanReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, err := validatePlanReport(decoded, "owner/current"); err != nil {
		t.Fatal(err)
	}
	decoded.Managed = 3
	if _, err := validatePlanReport(decoded, "owner/current"); err == nil {
		t.Fatal("combined scalar and collection count underreported")
	}
}
