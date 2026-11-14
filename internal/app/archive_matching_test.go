package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	shapeA "github.com/daiksud/gh-settings/internal/app/testdata/archive_shape_a"
	shapeB "github.com/daiksud/gh-settings/internal/app/testdata/archive_shape_b"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedAboutMatchingApplyIsReadOnly(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	archived := true
	description := "same"
	issues := true
	topics := []string{"alpha", "beta"}
	prune, err := settings.ParsePruneScopes("labels")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		which   string
		failure string
		prune   settings.PruneScopes
	}{
		{"description", "description", "", settings.PruneScopes{}},
		{"topics", "topics", "", settings.PruneScopes{}},
		{"both", "both", "", settings.PruneScopes{}},
		{"different description", "both", "description differs", settings.PruneScopes{}},
		{"different topics", "both", "topics differ", settings.PruneScopes{}},
		{"description unavailable", "both", "description unavailable", settings.PruneScopes{}},
		{"description incomplete", "both", "description incomplete", settings.PruneScopes{}},
		{"topics unavailable", "both", "topics unavailable", settings.PruneScopes{}},
		{"topics incomplete", "both", "topics incomplete", settings.PruneScopes{}},
		{"invalid topics", "both", "invalid topics", settings.PruneScopes{}},
		{"identity changed", "both", "identity changed", settings.PruneScopes{}},
		{"archive changed", "both", "archive changed", settings.PruneScopes{}},
		{"final read denied", "both", "final read denied", settings.PruneScopes{}},
		{"other intent", "other", "other intent", settings.PruneScopes{}},
		{"prune selected", "both", "prune selected", prune},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}}
			if tc.which != "topics" {
				d.Repository.Description = &description
			}
			if tc.which == "topics" || tc.which == "both" {
				d.Topics.Names = &topics
			}
			if tc.which == "other" {
				d.CopilotMCP = &issues
			}
			reads, writes := 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					reads++
					state := settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}
					if reads == 2 {
						switch tc.failure {
						case "identity changed":
							state.ID = 2
						case "archive changed":
							state.Archived = false
						case "final read denied":
							return settings.ArchiveState{}, errors.New("denied")
						}
					}
					return state, nil
				},
				ObserveRepository: func(_ repository.Repository, _ []settings.FieldID) (settings.RepositoryObservation, error) {
					if tc.failure == "description unavailable" {
						return settings.RepositoryObservation{}, errors.New("denied")
					}
					if tc.failure == "description incomplete" {
						return settings.RepositoryObservation{}, nil
					}
					value := "same"
					if tc.failure == "description differs" {
						value = "different"
					}
					return settings.NewRepositoryObservation(&settings.RepositoryState{Description: value}, nil, nil), nil
				},
				ReadTopics: func(repository.Repository) ([]string, error) {
					switch tc.failure {
					case "topics unavailable":
						return nil, errors.New("denied")
					case "topics incomplete":
						return nil, nil
					case "topics differ":
						return []string{"other"}, nil
					case "invalid topics":
						return []string{"INVALID"}, nil
					}
					return []string{"beta", "alpha"}, nil
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { writes++; return true, nil },
				Topics:     func(repository.Repository, settings.TopicsDeclaration) (bool, error) { writes++; return true, nil },
				Archive:    func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, writer, tc.prune)
			if (err != nil) != (tc.failure != "") || writes != 0 {
				t.Fatalf("result=%+v error=%v writes=%d", result, err, writes)
			}
			if tc.failure == "" && (result.Outcome != ApplyMatchingLabels || reads != 2) {
				t.Fatalf("matching result=%+v reads=%d", result, reads)
			}
			if tc.failure == "other intent" || tc.failure == "prune selected" {
				if reads != 1 || !strings.Contains(err.Error(), "without explicit archived: false") {
					t.Fatalf("existing guard changed: reads=%d err=%v", reads, err)
				}
			}
		})
	}
}

func TestArchivedMatchingLabelOwnership(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	archived := true
	id := int64(7)
	color := "112233"
	for _, tc := range []struct {
		name, observedDescription string
		id                        *int64
		readErr                   bool
		missingObservation        bool
		wantSuccess               bool
	}{
		{"omitted description equals empty", "", &id, false, false, true},
		{"omitted description owns empty", "existing", &id, false, false, false},
		{"ID-less selection cannot bind", "", nil, false, false, false},
		{"observation denied", "", &id, true, false, false},
		{"observation incomplete", "", &id, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			labels, err := settings.NewLabelDeclarations([]settings.LabelDeclaration{{ID: tc.id, Name: "bug", Color: &color}})
			if err != nil {
				t.Fatal(err)
			}
			declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Labels: &labels}
			reads, writes := 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					reads++
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
					if tc.readErr {
						return settings.LabelObservation{}, errors.New("denied")
					}
					if tc.missingObservation {
						return settings.LabelObservation{}, nil
					}
					return settings.NewLabelObservation([]settings.LabelState{{ID: 7, Name: "bug", Color: color, Description: tc.observedDescription}})
				},
				CreateLabel: func(repository.Repository, settings.LabelChange) (int64, error) { writes++; return 1, nil },
				UpdateLabel: func(repository.Repository, settings.LabelChange) error { writes++; return nil },
				DeleteLabel: func(repository.Repository, settings.LabelChange) error { writes++; return nil },
				CreatedID:   func(string, string, int64) error { writes++; return nil },
				Archive:     func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
			}
			result, err := Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, writer)
			if (err == nil) != tc.wantSuccess || writes != 0 {
				t.Fatalf("outcome=%v err=%v writes=%d", result.Outcome, err, writes)
			}
			if tc.wantSuccess && reads != 2 {
				t.Fatalf("expected two identity reads, got %d", reads)
			}
		})
	}
}

func TestArchivedDeclarationShapeFailsOnNestedNilFieldChange(t *testing.T) {
	type current struct{ N *string }
	type future struct {
		N     *string
		Added *bool
	}
	got, err := declarationShape(reflect.TypeOf(settings.Declaration{}))
	if err != nil || got != archivedDeclarationShape {
		t.Fatalf("declaration schema changed: %s %v", got, err)
	}
	a, err := declarationShape(reflect.TypeOf(current{}))
	if err != nil {
		t.Fatal(err)
	}
	b, err := declarationShape(reflect.TypeOf(future{}))
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("future nil field was not detected")
	}
}

func TestDeclarationShapeTracksRepeatedFieldTypeEdges(t *testing.T) {
	// The two root types model the same named declaration before and after a
	// field changes to another type that is already referenced by that struct.
	type before struct {
		A *string
		B *bool
		C *string
	}
	type after struct {
		A *string
		B *bool
		C *bool
	}
	a, b := reflect.TypeOf(before{}), reflect.TypeOf(after{})
	identity := func(typ reflect.Type) string {
		if typ == a || typ == b {
			return "same.package.M"
		}
		return canonicalDeclarationTypeIdentity(typ)
	}
	previous, err := declarationShapeWithIdentity(a, identity)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := declarationShapeWithIdentity(b, identity)
	if err != nil {
		t.Fatal(err)
	}
	if previous == changed {
		t.Fatal("changing a later field to an already-seen type was not detected")
	}
}

func TestDeclarationShapeDistinguishesPointerElementPackagePaths(t *testing.T) {
	// Both Value types have the same package name and type name, but different
	// import paths. Both pointer types were encountered before C changes.
	type before struct {
		A *shapeA.Value
		B *shapeB.Value
		C *shapeA.Value
	}
	type after struct {
		A *shapeA.Value
		B *shapeB.Value
		C *shapeB.Value
	}
	type sliceBefore struct {
		A []*shapeA.Value
		B []*shapeB.Value
		C []*shapeA.Value
	}
	type sliceAfter struct {
		A []*shapeA.Value
		B []*shapeB.Value
		C []*shapeB.Value
	}
	type mapBefore struct {
		A map[string]*shapeA.Value
		B map[string]*shapeB.Value
		C map[string]*shapeA.Value
	}
	type mapAfter struct {
		A map[string]*shapeA.Value
		B map[string]*shapeB.Value
		C map[string]*shapeB.Value
	}
	type arrayBefore struct {
		A [2]*shapeA.Value
		B [2]*shapeB.Value
		C [2]*shapeA.Value
	}
	type arrayAfter struct {
		A [2]*shapeA.Value
		B [2]*shapeB.Value
		C [2]*shapeB.Value
	}
	for _, tc := range []struct {
		name          string
		before, after reflect.Type
	}{
		{"pointer", reflect.TypeOf(before{}), reflect.TypeOf(after{})},
		{"slice", reflect.TypeOf(sliceBefore{}), reflect.TypeOf(sliceAfter{})},
		{"map", reflect.TypeOf(mapBefore{}), reflect.TypeOf(mapAfter{})},
		{"array", reflect.TypeOf(arrayBefore{}), reflect.TypeOf(arrayAfter{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			identity := func(typ reflect.Type) string {
				if typ == tc.before || typ == tc.after {
					return "same.package.M"
				}
				return canonicalDeclarationTypeIdentity(typ)
			}
			previous, err := declarationShapeWithIdentity(tc.before, identity)
			if err != nil {
				t.Fatal(err)
			}
			changed, err := declarationShapeWithIdentity(tc.after, identity)
			if err != nil {
				t.Fatal(err)
			}
			if previous == changed {
				t.Fatal("changing an edge between same-named packages was not detected")
			}
		})
	}
}

func TestArchivedOnlyPruneOfOmittedLabelsIsUnmanaged(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	archived := true
	desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}}
	prune, err := settings.ParsePruneScopes("labels")
	if err != nil {
		t.Fatal(err)
	}
	reads, labelReads, writes := 0, 0, 0
	writer := Writer{
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			reads++
			return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
		},
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			labelReads++
			return settings.LabelObservation{}, nil
		},
		DeleteLabel: func(repository.Repository, settings.LabelChange) error { writes++; return nil },
		Archive:     func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
	}
	result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, writer, prune)
	if err != nil || result.Outcome != ApplyMatchingLabels || reads != 1 || labelReads != 0 || writes != 0 {
		t.Fatalf("result=%+v error=%v archive reads=%d label reads=%d writes=%d", result, err, reads, labelReads, writes)
	}
}
