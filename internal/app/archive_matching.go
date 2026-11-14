package app

import (
	"crypto/sha256"
	"fmt"
	"reflect"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The archive read-only path admits only explicitly verified families. Any
// type change in Declaration, including a nil nested field with owned defaults,
// disables this path until its semantics and the digest are reviewed together.
// This guards the schema shape; it is not a proof of comparator behavior.
const archivedDeclarationShape = "ab2770cfad99c311e5dfeb70f009e0dffe367a695528ba2bc24ad7dd0b8f148f"

func declarationShape(root reflect.Type) (string, error) {
	return declarationShapeWithIdentity(root, canonicalDeclarationTypeIdentity)
}

// Composite types have no package path of their own. Follow their elements so
// pointers and slices of same-named types from distinct packages stay distinct.
func canonicalDeclarationTypeIdentity(t reflect.Type) string {
	if t.Name() != "" {
		return fmt.Sprintf("named(%q,%q,%d)", t.PkgPath(), t.Name(), t.Kind())
	}
	switch t.Kind() {
	case reflect.Pointer:
		return "pointer(" + canonicalDeclarationTypeIdentity(t.Elem()) + ")"
	case reflect.Slice:
		return "slice(" + canonicalDeclarationTypeIdentity(t.Elem()) + ")"
	case reflect.Array:
		return fmt.Sprintf("array(%d,%s)", t.Len(), canonicalDeclarationTypeIdentity(t.Elem()))
	case reflect.Map:
		return "map(" + canonicalDeclarationTypeIdentity(t.Key()) + "," + canonicalDeclarationTypeIdentity(t.Elem()) + ")"
	case reflect.Struct:
		identity := "struct("
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			identity += fmt.Sprintf("%q,%q,%q,%t,%s;", f.Name, f.PkgPath, string(f.Tag), f.Anonymous, canonicalDeclarationTypeIdentity(f.Type))
		}
		return identity + ")"
	default:
		// Unsupported kinds are rejected by declarationShapeWithIdentity.
		return fmt.Sprintf("unsupported(%d,%q)", t.Kind(), t.String())
	}
}

func declarationShapeWithIdentity(root reflect.Type, identity func(reflect.Type) string) (string, error) {
	h := sha256.New()
	seen := make(map[reflect.Type]bool)
	var visit func(reflect.Type) error
	visit = func(t reflect.Type) error {
		// Every edge contributes its type, including references to an already
		// traversed type. A later field changing *string to *bool must not be
		// hidden merely because both types occurred earlier in the graph.
		fmt.Fprintln(h, identity(t))
		if seen[t] {
			return nil
		}
		seen[t] = true
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice:
			return visit(t.Elem())
		case reflect.Array:
			fmt.Fprintln(h, t.Len())
			return visit(t.Elem())
		case reflect.Map:
			if err := visit(t.Key()); err != nil {
				return err
			}
			return visit(t.Elem())
		case reflect.Struct:
			fmt.Fprintln(h, t.NumField())
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				fmt.Fprintf(h, "%q %q %q %t\n", f.Name, f.PkgPath, string(f.Tag), f.Anonymous)
				if err := visit(f.Type); err != nil {
					return err
				}
			}
		case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		default:
			return fmt.Errorf("unsupported declaration type %s", t)
		}
		return nil
	}
	if err := visit(root); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// archivedMatchingScope rejects every managed declaration outside the admitted
// set. With no pruning, ordinary apply's archive guard treats some explicit
// empty collections as unmanaged; use the same typed predicate for the residue.
func archivedMatchingScope(desired settings.Declaration, prune settings.PruneScopes) bool {
	if !archivedSchemaCompatible() || prune != (settings.PruneScopes{}) {
		return false
	}
	remaining := desired
	remaining.Repository = settings.RepositoryDeclaration{}
	remaining.Topics.Names = nil
	remaining.Labels = nil
	remaining.PullRequestLimits = settings.PullRequestLimitsDeclaration{}
	remaining.PullRequestExemptions = nil
	remaining.CopilotReviewEffort = nil
	remaining.Actions.RetentionDays = nil
	remaining.Actions.ForkApprovalPolicy = nil
	remaining.Security = settings.SecurityDeclaration{}
	remaining.Pages = settings.PagesDeclaration{}
	return !archiveConflictsWithOtherManaged(remaining)
}

func archivedSchemaCompatible() bool {
	shape, err := declarationShape(reflect.TypeOf(settings.Declaration{}))
	return err == nil && shape == archivedDeclarationShape
}

// verifyArchivedMatching performs only reads. The enclosing identity checks
// do not make the independent observations an atomic snapshot.
func verifyArchivedMatching(repo repository.Repository, desired settings.Declaration, initial settings.ArchiveState, read Writer) error {
	declaration := desired.Repository
	declaration.Archived = nil
	if !declaration.Empty() {
		if read.ObserveRepository == nil {
			return fmt.Errorf("verify archived About: reader unavailable")
		}
		fields := settings.DeclaredRepositoryFields(declaration)
		// The visitor and comparison must cover every selected scalar. The
		// shape guard also requires review before declaration fields change.
		selected := 0
		v := reflect.ValueOf(declaration)
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsNil() {
				selected++
			}
		}
		unique := make(map[settings.FieldID]bool, len(fields))
		for _, field := range fields {
			unique[field] = true
		}
		if len(fields) != selected || len(unique) != selected {
			return fmt.Errorf("verify archived repository: incomplete field coverage")
		}
		observed, err := read.ObserveRepository(repo, fields)
		if err != nil {
			return fmt.Errorf("verify archived About: %w", err)
		}
		comparison, err := settings.CompareRepositoryObservation(observed, declaration)
		if err != nil {
			return fmt.Errorf("verify archived About: %w", err)
		}
		if comparison.Managed != selected || comparison.HasChanges() {
			return fmt.Errorf("archived repository About differs")
		}
	}
	if desired.Topics.Names != nil {
		if read.ReadTopics == nil {
			return fmt.Errorf("verify archived topics: reader unavailable")
		}
		names, err := read.ReadTopics(repo)
		if err == nil && names == nil {
			err = fmt.Errorf("missing complete topics observation")
		}
		if err == nil {
			err = settings.ValidateTopics(names)
		}
		if err != nil {
			return fmt.Errorf("verify archived topics: %w", err)
		}
		if settings.CompareTopics(names, desired.Topics).HasChanges() {
			return fmt.Errorf("archived repository topics differ")
		}
	}
	if desired.Labels != nil && desired.Labels.Len() > 0 {
		if read.ReadLabels == nil {
			return fmt.Errorf("verify archived labels: reader unavailable")
		}
		current, err := read.ReadLabels(repo)
		if err != nil {
			return fmt.Errorf("verify archived labels: %w", err)
		}
		comparison, err := settings.CompareLabels(current, desired.Labels)
		if err != nil {
			return fmt.Errorf("verify archived labels: %w", err)
		}
		if comparison.HasChanges() {
			return fmt.Errorf("archived repository labels differ")
		}
	}
	if err := verifyArchivedGroups(repo, desired, read); err != nil {
		return err
	}
	if err := verifyArchivedSecurity(repo, desired.Security, read); err != nil {
		return err
	}
	fresh, err := read.ReadArchive(repo)
	if err == nil {
		err = fresh.Validate(repo)
	}
	if err == nil && (fresh.ID != initial.ID || fresh.FullName != initial.FullName || !fresh.Archived) {
		err = fmt.Errorf("repository identity or archive state changed during verification")
	}
	if err != nil {
		return fmt.Errorf("verify archived repository identity: %w", err)
	}
	return nil
}
