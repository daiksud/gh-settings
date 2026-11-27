package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"go.yaml.in/yaml/v4"
)

type currentLedger struct {
	Shape     yaml.Node
	Choices   map[string][]string
	Contracts map[string]string
	Resources map[string]struct{ Identity, Ownership, Prune string }
	Lists     map[string]string
	LegacyV0  []string `yaml:"legacy_v0"`
}

func readCurrentLedger(t *testing.T) currentLedger {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "development", "settings-schema", "current.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var ledger currentLedger
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	return ledger
}

// Exercise the actual public encoder with every optional control present, including
// the alternate controls that cannot be selected together in a valid policy.
// Domain choice validation is tested separately with individual valid declarations.
func populateSchemaExample(value reflect.Value) {
	switch value.Kind() {
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		populateSchemaExample(value.Elem())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			// This removed input exists only to diagnose old declarations.
			if value.Type() == reflect.TypeFor[uimodel.Ruleset]() && value.Type().Field(i).Name == "Delete" {
				continue
			}
			populateSchemaExample(value.Field(i))
		}
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
		populateSchemaExample(value.Index(0))
	case reflect.Bool:
		value.SetBool(true)
	case reflect.String:
		value.SetString("example")
	case reflect.Int, reflect.Int64:
		value.SetInt(1)
	case reflect.Float64:
		value.SetFloat(0.5)
	}
}

type ledgerField struct{ path, kind string }

func orderedLedgerFields(t *testing.T, node *yaml.Node, path string, shape bool) []ledgerField {
	t.Helper()
	switch node.Kind {
	case yaml.MappingNode:
		fields := []ledgerField{{path, "mapping"}}
		for i := 0; i < len(node.Content); i += 2 {
			child := strings.TrimPrefix(path+"."+node.Content[i].Value, ".")
			fields = append(fields, orderedLedgerFields(t, node.Content[i+1], child, shape)...)
		}
		return fields
	case yaml.SequenceNode:
		if len(node.Content) != 1 {
			t.Fatalf("schema probe needs one item to check %s", path)
		}
		return append([]ledgerField{{path, "sequence"}}, orderedLedgerFields(t, node.Content[0], path+"[]", shape)...)
	case yaml.ScalarNode:
		kind := node.Value
		if !shape {
			kind = map[string]string{"!!str": "string", "!!int": "integer", "!!float": "number", "!!bool": "boolean"}[node.Tag]
		}
		if !slices.Contains([]string{"string", "string|null", "integer", "boolean", "number"}, kind) {
			t.Fatalf("unexpected schema scalar %s at %s", kind, path)
		}
		return []ledgerField{{path, kind}}
	default:
		t.Fatalf("unexpected schema node kind %v at %s", node.Kind, path)
		return nil
	}
}

func reverseSchemaMappingOrder(node *yaml.Node) {
	for _, child := range node.Content {
		reverseSchemaMappingOrder(child)
	}
	if node.Kind == yaml.MappingNode {
		for i, j := 0, len(node.Content)-2; i < j; i, j = i+2, j-2 {
			node.Content[i], node.Content[j] = node.Content[j], node.Content[i]
			node.Content[i+1], node.Content[j+1] = node.Content[j+1], node.Content[i+1]
		}
	}
}

func TestCurrentSchemaDescribesExportOrderAndInputTypes(t *testing.T) {
	ledger := readCurrentLedger(t)
	want := orderedLedgerFields(t, &ledger.Shape, "", true)

	var example uimodel.Repository
	populateSchemaExample(reflect.ValueOf(&example).Elem())
	encoded, err := EncodeConfig(example)
	if err != nil {
		t.Fatal(err)
	}
	var output yaml.Node
	if err := yaml.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}
	got := orderedLedgerFields(t, output.Content[0], "", false)
	// The export probe supplies a concrete domain. Its other possible type is
	// exercised by the nullable-scalar cases in TestCurrentSchemaScalarTypes.
	for i, field := range want {
		if field.kind == "string|null" {
			want[i].kind = "string"
		}
	}
	for i := range min(len(want), len(got)) {
		if want[i] != got[i] {
			t.Fatalf("canonical field %d: ledger %+v, export %+v", i+1, want[i], got[i])
		}
	}
	if len(want) != len(got) {
		t.Fatalf("ledger has %d nodes, export has %d", len(want), len(got))
	}

	reverseSchemaMappingOrder(&output)
	reversed, err := yaml.Marshal(&output)
	if err != nil {
		t.Fatal(err)
	}
	read, err := DecodeConfig(reversed)
	if err != nil {
		t.Fatal(err)
	}
	again, err := EncodeConfig(read)
	if err != nil || !bytes.Equal(encoded, again) {
		t.Fatalf("reordered input did not round trip to the original canonical export: %v", err)
	}
}

func TestCurrentSchemaInventoryContracts(t *testing.T) {
	ledger := readCurrentLedger(t)
	types := make(map[string]string)
	for _, field := range orderedLedgerFields(t, &ledger.Shape, "", true) {
		types[field.path] = field.kind
		if field.kind == "sequence" && ledger.Lists[field.path] == "" && ledger.Resources[field.path].Identity == "" {
			t.Errorf("specify item identity and ownership for %s", field.path)
		}
	}
	for path, entry := range ledger.Resources {
		if types[path] != "sequence" || entry.Identity == "" || entry.Ownership == "" || entry.Prune == "" {
			t.Errorf("invalid resource contract for %s", path)
		}
	}
	for path, entry := range ledger.Lists {
		if types[path] != "sequence" || entry == "" {
			t.Errorf("invalid list contract for %s", path)
		}
	}
	for path, ref := range ledger.Contracts {
		name := filepath.Join("..", "..", "docs", "references", "settings", ref)
		if _, err := os.Stat(name); err != nil || types[path] == "" {
			t.Errorf("invalid contract %s for %s: %v", ref, path, err)
		}
	}
	for path, choices := range ledger.Choices {
		if types[path] != "string" || len(choices) == 0 {
			t.Errorf("invalid closed menu %s", path)
		}
		seen := map[string]bool{}
		for _, choice := range choices {
			if choice == "" || seen[choice] {
				t.Errorf("invalid or repeated choice %q at %s", choice, path)
			}
			seen[choice] = true
		}
	}
}

func TestCurrentSchemaScalarTypes(t *testing.T) {
	ledger := readCurrentLedger(t)
	for _, field := range orderedLedgerFields(t, &ledger.Shape, "", true) {
		if field.path == "" || field.kind == "sequence" || field.kind == "mapping" {
			continue
		}
		t.Run(field.path, func(t *testing.T) {
			for _, sample := range []struct {
				kind  string
				value any
			}{
				{"integer", 1}, {"number", 0.5}, {"boolean", true},
				{"string", "example"}, {"null", nil}, {"sequence", []string{}}, {"mapping", map[string]any{}},
			} {
				if field.path == "version" && sample.kind == "string" {
					sample.value = "daiksud/gh-settings@v0"
				}
				accept := field.kind == sample.kind ||
					(field.kind == "number" && sample.kind == "integer") ||
					(field.kind == "string|null" && (sample.kind == "string" || sample.kind == "null"))
				_, err := DecodeConfig(schemaInput(t, field.path, sample.value, nil))
				if (err == nil) != accept {
					t.Fatalf("ledger type %s, input type %s: %v", field.kind, sample.kind, err)
				}
			}
		})
	}
}

// Wrap one value in an otherwise minimal declaration, with a single element at
// each [] component. A list path can identify either the list or its item.
func schemaInput(t *testing.T, path string, value any, companions func(string) map[string]any) []byte {
	t.Helper()
	keys := strings.Split(path, ".")
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		if strings.HasSuffix(key, "[]") {
			key = strings.TrimSuffix(key, "[]")
			value = []any{value}
		}
		section := map[string]any{key: value}
		if companions != nil {
			for k, v := range companions(strings.Join(keys[:i], ".")) {
				if k != key {
					section[k] = v
				}
			}
		}
		value = section
	}
	doc := value.(map[string]any)
	if path != "version" {
		doc["version"] = "daiksud/gh-settings@v0"
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A menu can live in an ID-selected object or a conditional subgroup. Supply
// the smallest valid surrounding declaration so an unrelated missing field
// cannot make an unsupported choice appear rejected.
func schemaChoiceCompanions(parent, path, choice string) map[string]any {
	switch parent {
	case "repository.settings.rulesets[]":
		return map[string]any{"ruleset_name": "main", "enforcement_status": "active", "target_branches": map[string]any{"include_default_branch": true}}
	case "repository.settings.rulesets[].bypass_list[]":
		return map[string]any{"role": "repository_admin", "bypass_mode": "always"}
	case "repository.settings.rulesets[].rules.require_code_scanning_results.code_scanning_tools[]":
		return map[string]any{"tool_name": "CodeQL", "alerts": "errors", "security_alerts": "high_or_higher"}
	case "repository.settings.branch_protection_rules[]":
		return map[string]any{"id": 1, "branch_name_pattern": "main"}
	case "repository.settings.branch_protection_rules[].allow_force_pushes":
		if strings.HasSuffix(path, ".who_can_force_push") && choice == "everyone" {
			return map[string]any{"enabled": true}
		}
		return map[string]any{"enabled": true, "who_can_force_push": "specified_actors", "actors": []any{map[string]any{"type": "user", "id": "user-node"}}}
	case "repository.settings.branch_protection_rules[].allow_force_pushes.actors[]":
		return map[string]any{"id": "actor-node"}
	case "repository.settings.actions.policies[]":
		return map[string]any{"id": 1, "policy_name": "test", "enforcement_status": "active"}
	case "repository.settings.webhooks[]":
		return map[string]any{"id": 1}
	case "repository.settings.environments[]":
		if strings.Contains(path, ".deployment_branch_or_tag_rules[]") {
			return map[string]any{"name": "test", "deployment_branches_and_tags": "selected_branches_and_tags"}
		}
		return map[string]any{"name": "test"}
	case "repository.settings.environments[].deployment_branch_or_tag_rules[]":
		return map[string]any{"name_pattern": "release"}
	case "repository.settings.pages.build_and_deployment":
		if !strings.HasSuffix(path, ".source") || choice == "deploy_from_a_branch" {
			return map[string]any{"source": "deploy_from_a_branch", "branch": "main", "select_folder": "root"}
		}
	case "repository.settings.advanced_security.code_security.codeql_analysis.scan_settings":
		if strings.HasSuffix(path, ".runner_type") && choice == "labeled_runner" {
			return map[string]any{"runner_label": "build"}
		}
	case "repository.settings.autolinks[]":
		return map[string]any{"reference_prefix": "TICKET-", "target_url": "https://example.com/<num>"}
	}
	return nil
}

func schemaChoiceError(t *testing.T, path, choice string) error {
	t.Helper()
	input := schemaInput(t, path, choice, func(parent string) map[string]any {
		return schemaChoiceCompanions(parent, path, choice)
	})
	ui, err := DecodeConfig(input)
	if err != nil {
		return err
	}
	_, err = github.DeclarationFromUI(ui)
	return err
}

func TestCurrentSchemaClosedMenus(t *testing.T) {
	ledger := readCurrentLedger(t)
	for path, values := range ledger.Choices {
		t.Run(path, func(t *testing.T) {
			for _, value := range values {
				if err := schemaChoiceError(t, path, value); err != nil {
					t.Errorf("recorded choice %q was rejected: %v", value, err)
				}
			}
			if schemaChoiceError(t, path, "unrecognized_schema_choice") == nil {
				t.Error("accepted an unrecognized choice in a closed menu")
			}
		})
	}
}

type futureSchemaField struct {
	Path, Type, Status, Extension, UI, Before, After, Items string
	Evidence                                                any
	Choices                                                 []string
}

func readFutureLedger(t *testing.T) []futureSchemaField {
	t.Helper()
	var result []futureSchemaField
	files, err := filepath.Glob("../../docs/development/settings-schema/future/*.yml")
	if err != nil || len(files) == 0 {
		t.Fatalf("load proposed fields: %v", err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var records []futureSchemaField
		if err := yaml.Unmarshal(data, &records); err != nil {
			t.Fatal(file, err)
		}
		result = append(result, records...)
	}
	return result
}

func TestFutureSchemaFitsTheRecordedTree(t *testing.T) {
	ledger := readCurrentLedger(t)
	kind := map[string]string{}
	for _, field := range orderedLedgerFields(t, &ledger.Shape, "", true) {
		kind[field.path] = field.kind
	}
	rows := readFutureLedger(t)
	seen := map[string]bool{}
	absolute := regexp.MustCompile(`^repository(\.[a-z][a-z0-9_]*(\[\])?)+$`)
	for _, row := range rows {
		if !absolute.MatchString(row.Path) || seen[row.Path] || strings.TrimSpace(row.UI) == "" {
			t.Fatalf("duplicate, incomplete or invalid UI record %q", row.Path)
		}
		evidence, list := row.Evidence.([]any)
		if !list {
			evidence = []any{row.Evidence}
		}
		if len(evidence) == 0 {
			t.Errorf("missing UI evidence for %s", row.Path)
		}
		for _, item := range evidence {
			if ref, ok := item.(string); !ok || strings.TrimSpace(ref) == "" {
				t.Errorf("invalid UI evidence for %s", row.Path)
			}
		}
		seen[row.Path] = true
		if !slices.Contains([]string{"boolean", "integer", "number", "enum", "string", "mapping", "list"}, row.Type) {
			t.Errorf("invalid type %q at %s", row.Type, row.Path)
		}
		if !slices.Contains([]string{"reserved", "reshape", "unconfirmed"}, row.Status) {
			t.Errorf("invalid status %q at %s", row.Status, row.Path)
		}
		if row.Type == "enum" || row.Items == "enum" {
			if len(row.Choices) == 0 {
				t.Errorf("enum needs at least one choice at %s", row.Path)
			}
		}
		if row.Extension == "enum" {
			if kind[row.Path] != "string" || row.Type != "enum" || len(ledger.Choices[row.Path]) == 0 || len(row.Choices) == 0 {
				t.Errorf("invalid enum extension at %s", row.Path)
			}
			continue
		}
		if row.Extension != "" {
			t.Errorf("unknown extension %q at %s", row.Extension, row.Path)
		}
		if (kind[row.Path] != "") != (row.Status == "reshape") {
			t.Errorf("expected reshape only for a supported path: %s", row.Path)
		}
		kind[row.Path] = row.Type
		if row.Type == "list" {
			if !slices.Contains([]string{"boolean", "integer", "number", "string", "enum", "mapping", "unconfirmed"}, row.Items) {
				t.Errorf("invalid or missing list item type at %s", row.Path)
			}
			kind[row.Path] = "sequence"
			kind[row.Path+"[]"] = row.Items
		}
	}
	for _, row := range rows {
		parent := row.Path[:strings.LastIndex(row.Path, ".")]
		if kind[parent] != "mapping" {
			t.Errorf("missing or non-mapping parent %s for %s", parent, row.Path)
		}
		for _, sibling := range []string{row.Before, row.After} {
			if sibling != "" && (strings.Contains(sibling, ".") || parent+"."+sibling == row.Path || kind[parent+"."+sibling] == "") {
				t.Errorf("%s refers to missing or invalid sibling %q", row.Path, sibling)
			}
		}
	}
}

func TestFutureSchemaStaysOutsideOperationalInput(t *testing.T) {
	ledger := readCurrentLedger(t)
	for _, row := range readFutureLedger(t) {
		t.Run(row.Path, func(t *testing.T) {
			if row.Extension == "enum" {
				current := ledger.Choices[row.Path]
				if len(current) == 0 || len(row.Choices) == 0 {
					t.Fatal("enum extension needs current and reserved choices")
				}
				for _, choice := range row.Choices {
					if slices.Contains(current, choice) || schemaChoiceError(t, row.Path, choice) == nil {
						t.Errorf("reserved choice %q is already accepted", choice)
					}
				}
				return
			}
			var value any
			switch row.Type {
			case "boolean":
				value = true
			case "integer":
				value = 1
			case "number":
				value = 0.5
			case "string":
				value = "example"
			case "enum":
				if len(row.Choices) == 0 {
					t.Fatal("reserved enum needs its UI choices")
				}
				value = row.Choices[0]
			case "mapping":
				value = map[string]any{}
			case "list":
				value = []any{}
			default:
				t.Fatalf("unknown future type %q", row.Type)
			}
			_, err := DecodeConfig(schemaInput(t, row.Path, value, nil))
			if err == nil {
				t.Fatalf("operational codec accepted the proposed %s", row.Type)
			}
		})
	}
}

func TestRemovedRulesetDeleteIsOutsideCurrentSchema(t *testing.T) {
	const path = "repository.settings.rulesets[].delete"
	for _, enabled := range []bool{true, false} {
		input := schemaInput(t, path, enabled, func(parent string) map[string]any {
			return schemaChoiceCompanions(parent, path, "")
		})
		ui, err := DecodeConfig(input)
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted the removed per-ruleset delete declaration")
		}
	}
}
