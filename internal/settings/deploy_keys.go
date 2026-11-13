package settings

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/crypto/ssh"
)

const DeployKeysPath = "repository.settings.deploy_keys"
const DeployPublicKeyLimit = 64 << 10

// DeployPublicKey contains public data, but reports receive only its fingerprint.
// Construction is deliberately separate from private-key parsing and key generation.
type DeployPublicKey struct{ canonical, fingerprint string }

func ParseDeployPublicKey(input string) (DeployPublicKey, error) {
	invalid := func() (DeployPublicKey, error) {
		return DeployPublicKey{}, fmt.Errorf("deploy key must be one valid public SSH key without authorization options")
	}
	if len(input) == 0 || len(input) > DeployPublicKeyLimit {
		return invalid()
	}
	line := strings.TrimSuffix(input, "\n")
	if len(line) != len(input) {
		line = strings.TrimSuffix(line, "\r")
	}
	if strings.ContainsAny(line, "\r\n") {
		return invalid()
	}
	line = strings.Trim(line, " \t")
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return invalid()
	}
	key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil || key == nil || len(options) != 0 || len(bytes.TrimSpace(rest)) != 0 || fields[0] != key.Type() {
		return invalid()
	}
	return DeployPublicKey{strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(key)), "\n"), ssh.FingerprintSHA256(key)}, nil
}
func (k DeployPublicKey) Canonical() string   { return k.canonical }
func (k DeployPublicKey) Fingerprint() string { return k.fingerprint }
func (k DeployPublicKey) valid() bool         { return k.canonical != "" && k.fingerprint != "" }

type DeployKeyDeclaration struct {
	ID               *int64
	Title            string
	Key              DeployPublicKey
	AllowWriteAccess bool
}

func (d DeployKeyDeclaration) Validate() error {
	if d.ID != nil && *d.ID <= 0 {
		return fmt.Errorf("deploy key id must be positive")
	}
	if strings.ContainsFunc(d.Title, unicode.IsControl) {
		return fmt.Errorf("deploy key title contains control characters")
	}
	if !d.Key.valid() {
		return fmt.Errorf("deploy key public material is unavailable")
	}
	return nil
}
func cloneDeployDeclaration(d DeployKeyDeclaration) DeployKeyDeclaration {
	if d.ID != nil {
		id := *d.ID
		d.ID = &id
	}
	return d
}

type DeployKeyDeclarations struct{ values []DeployKeyDeclaration }

func NewDeployKeyDeclarations(values []DeployKeyDeclaration) (DeployKeyDeclarations, error) {
	result := DeployKeyDeclarations{values: make([]DeployKeyDeclaration, 0, len(values))}
	ids, keys, fingerprints := map[int64]bool{}, map[string]bool{}, map[string]bool{}
	for _, d := range values {
		if err := d.Validate(); err != nil {
			return DeployKeyDeclarations{}, err
		}
		if d.ID != nil {
			if ids[*d.ID] {
				return DeployKeyDeclarations{}, fmt.Errorf("duplicate deploy key id")
			}
			ids[*d.ID] = true
		}
		if keys[d.Key.canonical] || fingerprints[d.Key.fingerprint] {
			return DeployKeyDeclarations{}, fmt.Errorf("duplicate deploy key public identity")
		}
		keys[d.Key.canonical], fingerprints[d.Key.fingerprint] = true, true
		result.values = append(result.values, cloneDeployDeclaration(d))
	}
	return result, nil
}
func (d DeployKeyDeclarations) Len() int { return len(d.values) }
func (d DeployKeyDeclarations) Values() []DeployKeyDeclaration {
	v := make([]DeployKeyDeclaration, 0, len(d.values))
	for _, x := range d.values {
		v = append(v, cloneDeployDeclaration(x))
	}
	return v
}

type DeployKeyTarget struct {
	ID, OwnerID int64
	FullName    string
	Archived    bool
}

func (t DeployKeyTarget) Validate() error {
	if t.ID <= 0 || t.OwnerID <= 0 || t.FullName == "" {
		return fmt.Errorf("deploy key repository identity is incomplete")
	}
	return nil
}

// Optional strings preserve absent/null/value acquisition without becoming owned state.
type DeployKeyOptionalString struct {
	Present, Null bool
	Value         string
}
type DeployKeyState struct {
	ID                int64
	Title             string
	Key               DeployPublicKey
	ReadOnly          bool
	CreatedAt, APIURL string
	Enabled           *bool
	Verified          bool
	AddedBy, LastUsed DeployKeyOptionalString
}

func cloneDeployState(s DeployKeyState) DeployKeyState {
	if s.Enabled != nil {
		b := *s.Enabled
		s.Enabled = &b
	}
	return s
}
func (s DeployKeyState) Declaration() DeployKeyDeclaration {
	id := s.ID
	return DeployKeyDeclaration{ID: &id, Title: s.Title, Key: s.Key, AllowWriteAccess: !s.ReadOnly}
}
func sameDeployEnabled(a, b *bool) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// Dynamic usage, verification and attribution never identify a credential.
func DeployKeyStableEqual(a, b DeployKeyState) bool {
	return a.ID == b.ID && a.Title == b.Title && a.Key.canonical == b.Key.canonical && a.ReadOnly == b.ReadOnly && a.CreatedAt == b.CreatedAt && a.APIURL == b.APIURL && sameDeployEnabled(a.Enabled, b.Enabled)
}
func DeployKeyConfigurationMatches(s DeployKeyState, d DeployKeyDeclaration) bool {
	return d.Validate() == nil && s.Title == d.Title && s.Key.canonical == d.Key.canonical && s.ReadOnly == !d.AllowWriteAccess && (d.ID == nil || *d.ID == s.ID)
}

type DeployKeyObservation struct {
	acquired bool
	target   DeployKeyTarget
	values   []DeployKeyState
}

func NewDeployKeyObservation(target DeployKeyTarget, values []DeployKeyState) (DeployKeyObservation, error) {
	if err := target.Validate(); err != nil {
		return DeployKeyObservation{}, err
	}
	result := DeployKeyObservation{acquired: true, target: target, values: make([]DeployKeyState, 0, len(values))}
	ids, keys := map[int64]bool{}, map[string]bool{}
	for _, v := range values {
		if v.ID <= 0 || v.CreatedAt == "" || v.APIURL == "" || v.Declaration().Validate() != nil {
			return DeployKeyObservation{}, fmt.Errorf("deploy key observation is incomplete")
		}
		if ids[v.ID] || keys[v.Key.canonical] {
			return DeployKeyObservation{}, fmt.Errorf("deploy key observation repeats identity")
		}
		ids[v.ID], keys[v.Key.canonical] = true, true
		result.values = append(result.values, cloneDeployState(v))
	}
	slices.SortFunc(result.values, func(a, b DeployKeyState) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return result, nil
}
func (o DeployKeyObservation) Values() (DeployKeyTarget, []DeployKeyState, error) {
	if !o.acquired {
		return DeployKeyTarget{}, nil, fmt.Errorf("deploy key observation unavailable")
	}
	v := make([]DeployKeyState, 0, len(o.values))
	for _, s := range o.values {
		v = append(v, cloneDeployState(s))
	}
	return o.target, v, nil
}
func (o DeployKeyObservation) Declaration() (DeployKeyDeclarations, error) {
	_, v, e := o.Values()
	if e != nil {
		return DeployKeyDeclarations{}, e
	}
	d := make([]DeployKeyDeclaration, 0, len(v))
	for _, s := range v {
		d = append(d, s.Declaration())
	}
	return NewDeployKeyDeclarations(d)
}
func (o DeployKeyObservation) HasID(id int64) bool {
	for _, s := range o.values {
		if s.ID == id {
			return true
		}
	}
	return false
}
func (o DeployKeyObservation) State(id int64) (DeployKeyState, bool) {
	for _, s := range o.values {
		if s.ID == id {
			return cloneDeployState(s), true
		}
	}
	return DeployKeyState{}, false
}
func (o DeployKeyObservation) AfterOwnUnarchive() (DeployKeyObservation, error) {
	t, v, e := o.Values()
	if e != nil {
		return DeployKeyObservation{}, e
	}
	t.Archived = false
	return NewDeployKeyObservation(t, v)
}
func DeployKeyInventoryEqual(a, b DeployKeyObservation) bool {
	at, av, ae := a.Values()
	bt, bv, be := b.Values()
	if ae != nil || be != nil || at != bt || len(av) != len(bv) {
		return false
	}
	for i := range av {
		if !DeployKeyStableEqual(av[i], bv[i]) {
			return false
		}
	}
	return true
}
func (o DeployKeyObservation) WithCreated(s DeployKeyState) (DeployKeyObservation, error) {
	t, v, e := o.Values()
	if e != nil {
		return DeployKeyObservation{}, e
	}
	if o.HasID(s.ID) {
		return DeployKeyObservation{}, fmt.Errorf("deploy key creation reused identity")
	}
	return NewDeployKeyObservation(t, append(v, s))
}
func (o DeployKeyObservation) Without(id int64) (DeployKeyObservation, error) {
	t, v, e := o.Values()
	if e != nil {
		return DeployKeyObservation{}, e
	}
	if !o.HasID(id) {
		return DeployKeyObservation{}, fmt.Errorf("deploy key removal identity is missing")
	}
	return NewDeployKeyObservation(t, slices.DeleteFunc(v, func(s DeployKeyState) bool { return s.ID == id }))
}

// Reports contain safe display identity and the exact owned grant, never raw key material.
type DeployKeyDisplay struct {
	ID               *int64 `json:"id,omitempty"`
	Title            string `json:"title"`
	Fingerprint      string `json:"fingerprint"`
	AllowWriteAccess bool   `json:"allow_write_access"`
}

func DeployKeyDisplayFor(d DeployKeyDeclaration) DeployKeyDisplay {
	return DeployKeyDisplay{cloneDeployDeclaration(d).ID, d.Title, d.Key.fingerprint, d.AllowWriteAccess}
}
func (v DeployKeyDisplay) Validate() error {
	if v.ID != nil && *v.ID <= 0 || strings.ContainsFunc(v.Title, unicode.IsControl) || !strings.HasPrefix(v.Fingerprint, "SHA256:") {
		return fmt.Errorf("invalid deploy key report")
	}
	raw, err := base64.RawStdEncoding.Strict().DecodeString(strings.TrimPrefix(v.Fingerprint, "SHA256:"))
	if err != nil || len(raw) != 32 {
		return fmt.Errorf("invalid deploy key fingerprint")
	}
	return nil
}

type DeployKeyChange struct {
	operation     string
	before, after *DeployKeyDisplay
}

func NewDeployKeyChange(operation string, before, after *DeployKeyDisplay) (DeployKeyChange, error) {
	if operation != "create" && operation != "delete" || operation == "create" && (before != nil || after == nil || after.ID != nil) || operation == "delete" && (after != nil || before == nil || before.ID == nil) {
		return DeployKeyChange{}, fmt.Errorf("invalid deploy key change")
	}
	for _, v := range []*DeployKeyDisplay{before, after} {
		if v != nil {
			if err := v.Validate(); err != nil {
				return DeployKeyChange{}, err
			}
		}
	}
	clone := func(v *DeployKeyDisplay) *DeployKeyDisplay {
		if v == nil {
			return nil
		}
		x := *v
		if x.ID != nil {
			id := *x.ID
			x.ID = &id
		}
		return &x
	}
	return DeployKeyChange{operation, clone(before), clone(after)}, nil
}
func (c DeployKeyChange) Operation() string { return c.operation }
func (c DeployKeyChange) Values() (*DeployKeyDisplay, *DeployKeyDisplay) {
	x, _ := NewDeployKeyChange(c.operation, c.before, c.after)
	return x.before, x.after
}

type DeployKeySelection struct {
	desired DeployKeyDeclaration
	before  *DeployKeyState
}

func (s DeployKeySelection) Desired() DeployKeyDeclaration { return cloneDeployDeclaration(s.desired) }
func (s DeployKeySelection) Create() bool                  { return s.before == nil }
func PrepareDeployKeys(o DeployKeyObservation, d *DeployKeyDeclarations, prune bool) ([]DeployKeySelection, []DeployKeyChange, error) {
	if d == nil {
		return nil, nil, nil
	}
	_, values, err := o.Values()
	if err != nil {
		return nil, nil, err
	}
	if _, err := NewDeployKeyDeclarations(d.Values()); err != nil {
		return nil, nil, err
	}
	byID, byKey := map[int64]DeployKeyState{}, map[string]int64{}
	for _, v := range values {
		byID[v.ID], byKey[v.Key.canonical] = v, v.ID
	}
	selected := make([]DeployKeySelection, 0, d.Len())
	changes := []DeployKeyChange{}
	retained := map[int64]bool{}
	for _, want := range d.Values() {
		if want.ID != nil {
			before, found := byID[*want.ID]
			if !found {
				return nil, nil, fmt.Errorf("selected deploy key id is missing; no recreation is attempted")
			}
			if !DeployKeyConfigurationMatches(before, want) {
				return nil, nil, fmt.Errorf("deploy keys are immutable; prune the old ID before a separate replacement apply")
			}
			retained[before.ID] = true
			copy := cloneDeployState(before)
			selected = append(selected, DeployKeySelection{want, &copy})
			continue
		}
		if id, exists := byKey[want.Key.canonical]; exists {
			return nil, nil, fmt.Errorf("deploy key material already belongs to id %d; explicit ID binding is required", id)
		}
		selected = append(selected, DeployKeySelection{desired: want})
		display := DeployKeyDisplayFor(want)
		change, _ := NewDeployKeyChange("create", nil, &display)
		changes = append(changes, change)
	}
	if prune {
		for _, v := range values {
			if !retained[v.ID] {
				display := DeployKeyDisplayFor(v.Declaration())
				change, _ := NewDeployKeyChange("delete", &display, nil)
				changes = append(changes, change)
			}
		}
	}
	return selected, changes, nil
}

// DeployKeyWriteResult keeps recovery ID evidence separate from verified state.
type DeployKeyWriteResult struct {
	ID          int64
	Observation DeployKeyObservation
}
