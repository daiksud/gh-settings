package githubactions

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

const workspaceLockBridge = "$/.github/actions/lock-workspace-dependencies"

type lockInventoryStep struct {
	Uses string         `yaml:"uses"`
	If   yaml.Node      `yaml:"if"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
	Env  map[string]any `yaml:"env"`
}

type lockInventoryDocument struct {
	Jobs map[string]struct {
		Uses  string              `yaml:"uses"`
		Steps []lockInventoryStep `yaml:"steps"`
	} `yaml:"jobs"`
	Runs struct {
		Using string              `yaml:"using"`
		Steps []lockInventoryStep `yaml:"steps"`
	} `yaml:"runs"`
}

type lockInventoryRead func(string) ([]byte, error)

func lockInventoryDiskRead(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(root(), filepath.FromSlash(name)))
}

func lockInventoryDocumentRead(read lockInventoryRead, name string) (lockInventoryDocument, error) {
	data, err := read(name)
	if err != nil {
		return lockInventoryDocument{}, err
	}
	var doc lockInventoryDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return doc, fmt.Errorf("%s: %w", name, err)
	}
	return doc, nil
}

// These are checkout destinations of this same frozen source, not repository
// directories. The CI/release identity and consumer-isolation tests verify the
// corresponding source checks; the dependency walk must inspect root action.yml.
func lockInventoryLocalDirectory(uses string) (string, bool) {
	var local string
	switch {
	case strings.HasPrefix(uses, "$/"):
		local = strings.TrimPrefix(uses, "$/")
	case strings.HasPrefix(uses, "./"):
		local = strings.TrimPrefix(uses, "./")
	default:
		return "", false
	}
	for _, checkout := range []string{".action", ".published"} {
		if local == checkout {
			return ".", true
		}
		if strings.HasPrefix(local, checkout+"/") {
			local = strings.TrimPrefix(local, checkout+"/")
			break
		}
	}
	return path.Clean(local), true
}

var lockInventorySHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Lock entries identify source repositories/commits. Distinct action subpaths
// remain distinct in the bridge inventory but share a repository lock entry.
func lockInventoryRemotePin(uses string) (string, error) {
	name, sha, ok := strings.Cut(uses, "@")
	parts := strings.Split(name, "/")
	if !ok || len(parts) < 2 || !lockInventorySHA.MatchString(sha) {
		return "", fmt.Errorf("remote Action must retain a full SHA: %q", uses)
	}
	return strings.Join(parts[:2], "/") + "@" + sha, nil
}

func lockInventorySorted(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// Walk only actual uses edges, including arbitrarily nested composites. A new
// local dependency must be resolved rather than silently excluded from coverage.
func lockInventoryClosure(read lockInventoryRead, workflow string) (map[string]bool, map[string]bool, error) {
	remote, visited := map[string]bool{}, map[string]bool{}
	var walk func(string, bool) error
	walk = func(name string, workflowFile bool) error {
		if visited[name] {
			return nil
		}
		visited[name] = true
		doc, err := lockInventoryDocumentRead(read, name)
		if err != nil {
			return err
		}
		steps := doc.Runs.Steps
		if workflowFile {
			steps = nil
			for _, job := range doc.Jobs {
				if job.Uses != "" {
					return fmt.Errorf("reusable workflow needs separate lock coverage: %s", name)
				}
				steps = append(steps, job.Steps...)
			}
		}
		for _, step := range steps {
			if step.Uses == "" {
				continue
			}
			if step.Uses == workspaceLockBridge {
				if !workflowFile {
					return fmt.Errorf("runtime composite must not execute bridge: %s", name)
				}
				continue
			}
			if local, ok := lockInventoryLocalDirectory(step.Uses); ok {
				bridgeDir, _ := lockInventoryLocalDirectory(workspaceLockBridge)
				if local == bridgeDir {
					return fmt.Errorf("runtime local Action must not execute bridge: %s", name)
				}
				if local == ".." || strings.HasPrefix(local, "../") {
					return fmt.Errorf("local Action leaves source: %s", step.Uses)
				}
				action := path.Join(local, "action.yml")
				if _, err := read(action); os.IsNotExist(err) {
					action = path.Join(local, "action.yaml")
				}
				if err := walk(action, false); err != nil {
					return fmt.Errorf("%s uses %s: %w", name, step.Uses, err)
				}
			} else {
				if _, err := lockInventoryRemotePin(step.Uses); err != nil {
					return err
				}
				remote[step.Uses] = true
			}
		}
		return nil
	}
	err := walk(workflow, true)
	return remote, visited, err
}

func validateWorkspaceActionLock(read lockInventoryRead, workflow string) error {
	doc, err := lockInventoryDocumentRead(read, workflow)
	if err != nil {
		return err
	}

	isPR := strings.HasSuffix(workflow, "/pr.yml")
	bridgeCount := 0
	for jobName, job := range doc.Jobs {
		for _, step := range job.Steps {
			if step.Uses != workspaceLockBridge {
				continue
			}
			if isPR {
				return fmt.Errorf("%s: inert dependency inventory not allowed in Screening", workflow)
			}
			bridgeCount++
			condition := strings.TrimSpace(step.If.Value)
			if jobName != "resolve" || (condition != "${{ false }}" && condition != "false") ||
				step.Run != "" || len(step.With) != 0 || len(step.Env) != 0 {
				return fmt.Errorf("%s: dependency bridge must be inert in resolve", workflow)
			}
		}
	}
	if (isPR && bridgeCount != 0) || (!isPR && bridgeCount != 1) {
		return fmt.Errorf("%s: unexpected dependency inventory count %d", workflow, bridgeCount)
	}
	runtime, _, err := lockInventoryClosure(read, workflow)
	if err != nil {
		return err
	}
	if !isPR {
		bridgeDir, _ := lockInventoryLocalDirectory(workspaceLockBridge)
		bridge, err := lockInventoryDocumentRead(read, path.Join(bridgeDir, "action.yml"))
		if err != nil {
			return err
		}
		if bridge.Runs.Using != "composite" {
			return fmt.Errorf("bridge is not composite")
		}
		listed := map[string]bool{}
		for _, step := range bridge.Runs.Steps {
			if _, local := lockInventoryLocalDirectory(step.Uses); local || step.Uses == "" ||
				step.Run != "" || step.If.Kind != 0 || len(step.With) != 0 || len(step.Env) != 0 {
				return fmt.Errorf("bridge must contain only unconditional remote inventory entries")
			}
			if _, err := lockInventoryRemotePin(step.Uses); err != nil {
				return err
			}
			if listed[step.Uses] {
				return fmt.Errorf("duplicate bridge entry: %s", step.Uses)
			}
			listed[step.Uses] = true
		}
		if !reflect.DeepEqual(lockInventorySorted(runtime), lockInventorySorted(listed)) {
			return fmt.Errorf("%s: bridge differs from runtime dependencies", workflow)
		}
	}
	data, err := read(".github/workflows/actions.lock")
	if err != nil {
		return err
	}
	var lock struct {
		Workflows    map[string][]string `yaml:"workflows"`
		Dependencies map[string]struct {
			Commit string `yaml:"commit"`
		} `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return err
	}
	pins, ok := lock.Workflows[workflow]
	if !ok {
		return fmt.Errorf("%s: missing workflow lock inventory", workflow)
	}
	want, got := map[string]bool{}, map[string]bool{}
	for uses := range runtime {
		pin, _ := lockInventoryRemotePin(uses)
		want[pin] = true
	}
	for _, pin := range pins {
		if got[pin] {
			return fmt.Errorf("duplicate workflow lock pin: %s", pin)
		}
		got[pin] = true
		_, sha, _ := strings.Cut(pin, "@")
		if lock.Dependencies[pin].Commit != "sha1-"+sha {
			return fmt.Errorf("%s: lock dependency commit missing or different", pin)
		}
	}
	if !reflect.DeepEqual(lockInventorySorted(want), lockInventorySorted(got)) {
		return fmt.Errorf("%s: lock inventory differs from runtime dependencies: want=%v got=%v", workflow, lockInventorySorted(want), lockInventorySorted(got))
	}
	return nil
}

func TestWorkspaceActionLockInventoriesEveryWorkflow(t *testing.T) {
	for _, name := range []string{"pr", "ci-push", "release"} {
		t.Run(name, func(t *testing.T) {
			workflow := ".github/workflows/" + name + ".yml"
			if err := validateWorkspaceActionLock(lockInventoryDiskRead, workflow); err != nil {
				t.Fatal(err)
			}
			_, visited, err := lockInventoryClosure(lockInventoryDiskRead, workflow)
			if err != nil {
				t.Fatal(err)
			}
			required := []string{".github/actions/lint/action.yml", ".github/actions/unit-test/action.yml", ".github/actions/build/action.yml"}
			if name != "pr" {
				required = append(required, "action.yml")
			}
			for _, source := range required {
				if !visited[source] {
					t.Fatalf("%s did not traverse %s", workflow, source)
				}
			}
		})
	}
}

func TestWorkspaceActionLockDetectsInventoryDrift(t *testing.T) {
	const workflow = ".github/workflows/pr.yml"
	for _, test := range []struct{ name, file, from, to string }{
		{"changed-workflow-pin", workflow, `jdx/mise-action@[0-9a-f]{40}`, "jdx/mise-action@1111111111111111111111111111111111111111"},
		{"changed-nested-pin", ".github/actions/restore-go-cache/action.yml", `actions/cache/restore@[0-9a-f]{40}`, "actions/cache/restore@1111111111111111111111111111111111111111"},
		{"unexpected-remote-pin", workflow, `(?m)^      - name: Build executable\n`, "      - uses: actions/extra@1111111111111111111111111111111111111111\n      - name: Build executable\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			original, err := lockInventoryDiskRead(test.file)
			if err != nil {
				t.Fatal(err)
			}
			changed := regexp.MustCompile(test.from).ReplaceAllLiteralString(string(original), test.to)
			if changed == string(original) {
				t.Fatalf("missing mutation target in %s", test.file)
			}
			read := func(name string) ([]byte, error) {
				if name == test.file {
					return []byte(changed), nil
				}
				return lockInventoryDiskRead(name)
			}
			if err := validateWorkspaceActionLock(read, workflow); err == nil || !strings.Contains(err.Error(), "lock inventory differs") {
				t.Fatalf("dependency drift not rejected: %v", err)
			}
		})
	}
}

func TestWorkspaceActionLockRejectsMissingWorkflowPinDespiteGlobalEntry(t *testing.T) {
	const workflow = ".github/workflows/pr.yml"
	data, err := lockInventoryDiskRead(".github/workflows/actions.lock")
	if err != nil {
		t.Fatal(err)
	}
	var lock map[string]any
	if err := yaml.Unmarshal(data, &lock); err != nil {
		t.Fatal(err)
	}
	workflows := lock["workflows"].(map[string]any)
	pins := workflows[workflow].([]any)
	if len(pins) < 2 {
		t.Fatal("missing fixture workflow pins")
	}
	missing := pins[0].(string)
	if _, present := lock["dependencies"].(map[string]any)[missing]; !present {
		t.Fatal("missing pin must remain in global inventory")
	}
	workflows[workflow] = pins[1:]
	changed, err := yaml.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) ([]byte, error) {
		if name == ".github/workflows/actions.lock" {
			return changed, nil
		}
		return lockInventoryDiskRead(name)
	}
	if err := validateWorkspaceActionLock(read, workflow); err == nil || !strings.Contains(err.Error(), "lock inventory differs") {
		t.Fatalf("missing workflow pin accepted: %v", err)
	}
}
