package workspace

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestSchemaRecordsObservedEditorOrder(t *testing.T) {
	base := filepath.Join("..", "..", "docs", "development", "settings-schema")
	data, err := os.ReadFile(filepath.Join(base, "export-order.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var orders []struct {
		Path, Evidence string
		Fields         []string
		Implemented    bool
	}
	if err := yaml.Unmarshal(data, &orders); err != nil {
		t.Fatal(err)
	}
	ledger := readCurrentLedger(t)
	children := map[string][]string{}
	for _, field := range orderedLedgerFields(t, &ledger.Shape, "", true) {
		i := strings.LastIndex(field.path, ".")
		if i >= 0 && !strings.HasSuffix(field.path, "[]") {
			parent, key := field.path[:i], field.path[i+1:]
			children[parent] = append(children[parent], key)
		}
	}
	seen := map[string]bool{}
	for _, row := range orders {
		if seen[row.Path] || len(row.Fields) == 0 || row.Evidence == "" {
			t.Errorf("duplicate or incomplete editor order for %q", row.Path)
			continue
		}
		seen[row.Path] = true
		current := children[row.Path]
		want, have := slices.Clone(row.Fields), slices.Clone(current)
		slices.Sort(want)
		slices.Sort(have)
		if !slices.Equal(want, have) {
			t.Errorf("%s: editor and operational schema list different fields: %v and %v", row.Path, want, have)
		}
		if row.Implemented != slices.Equal(current, row.Fields) {
			t.Errorf("%s: recorded implementation status disagrees with actual export order", row.Path)
		}
	}
}

func TestSchemaEvidenceGapsIdentifyFreezeImpact(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "docs", "development", "settings-schema", "gaps-*.yml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("cannot read schema evidence gaps: %v", err)
	}
	seen := map[string]bool{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var gaps []struct {
			Control, Impact, Resolution string
			Evidence                    yaml.Node
		}
		if err := yaml.Unmarshal(data, &gaps); err != nil {
			t.Fatal(err)
		}
		for _, gap := range gaps {
			if gap.Control == "" || seen[gap.Control] || gap.Resolution == "" || gap.Evidence.IsZero() ||
				!slices.Contains([]string{"additive", "blocking", "excluded"}, gap.Impact) {
				t.Errorf("%s: incomplete or duplicate gap %q", file, gap.Control)
			}
			seen[gap.Control] = true
		}
	}
}
