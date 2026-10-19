package settings

import (
	"reflect"
	"strings"
	"testing"
)

func TestTopicsSetComparison(t *testing.T) {
	for _, test := range []struct {
		name             string
		before           []string
		desired          *[]string
		managed, changes int
	}{
		{"omitted", []string{"old"}, nil, 0, 0},
		{"clear", []string{"old"}, topicsPtr(), 1, 1},
		{"empty equal", nil, topicsPtr(), 1, 0},
		{"order independent", []string{"b", "a"}, topicsPtr("a", "b"), 1, 0},
		{"changed sorted", []string{"z", "a"}, topicsPtr("y", "b"), 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := CompareTopics(test.before, TopicsDeclaration{Names: test.desired})
			if got.Managed != test.managed || len(got.Changes) != test.changes {
				t.Fatalf("comparison=%+v", got)
			}
			if test.name == "changed sorted" && !reflect.DeepEqual(got.Changes, []Change{{field: FieldID{path: "topics.names"}, values: topicsPair{before: []string{"a", "z"}, after: []string{"b", "y"}}}}) {
				t.Fatalf("comparison=%+v", got)
			}
			if test.desired != nil && (Declaration{Topics: TopicsDeclaration{Names: test.desired}}).Empty() {
				t.Fatal("explicit list considered unmanaged")
			}
		})
	}
}
func topicsPtr(names ...string) *[]string {
	if names == nil {
		names = []string{}
	}
	return &names
}
func TestTopicNames(t *testing.T) {
	for _, names := range [][]string{{""}, {"Upper"}, {"a_b"}, {"a b"}, {"日本語"}, {strings.Repeat("a", 51)}, {"a", "a"}} {
		if ValidateTopics(names) == nil {
			t.Fatalf("accepted %q", names)
		}
	}
	names := make([]string, 21)
	for i := range names {
		names[i] = string(rune('a' + i))
	}
	if ValidateTopics(names) == nil {
		t.Fatal("accepted 21 topics")
	}
	if err := ValidateTopics([]string{strings.Repeat("a", 50), "github-cli", "123"}); err != nil {
		t.Fatal(err)
	}
}
