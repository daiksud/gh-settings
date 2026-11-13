package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const classicTargetFields = `id nameWithOwner isArchived isPrivate owner { __typename } squashMergeAllowed rebaseMergeAllowed`
const classicOwnedFields = `id databaseId pattern requiresConversationResolution requiresCommitSignatures requiresLinearHistory allowsDeletions lockBranch isAdminEnforced requiresDeployments requiredDeploymentEnvironments allowsForcePushes`
const classicCompanionFields = `blocksCreations dismissesStaleReviews lockAllowsFetchAndMerge requireLastPushApproval requiresApprovingReviews requiresCodeOwnerReviews requiresStatusChecks requiresStrictStatusChecks restrictsPushes restrictsReviewDismissals requiredApprovingReviewCount requiredStatusCheckContexts requiredStatusChecks { context app { id } }`
const classicRuleFields = classicOwnedFields + ` ` + classicCompanionFields + ` repository { ` + classicTargetFields + ` }`
const classicInventoryQuery = `query ClassicProtectionInventory($owner: String!, $name: String!, $cursor: String) { repository(owner:$owner,name:$name,followRenames:false) { ` + classicTargetFields + ` branchProtectionRules(first:100,after:$cursor) { totalCount pageInfo { hasNextPage endCursor } nodes { ` + classicRuleFields + ` } } } }`
const classicNodeQuery = `query ClassicProtectionRule($id:ID!) { node(id:$id) { __typename ... on BranchProtectionRule { ` + classicRuleFields + ` } } }`

type ClassicProtectionClient struct{ graphql *api.GraphQLClient }

func NewClassicProtectionClient(host string) (*ClassicProtectionClient, error) {
	c, err := api.NewGraphQLClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, LogIgnoreEnv: true, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure classic protection API: %w", err)
	}
	return &ClassicProtectionClient{graphql: c}, nil
}

type classicPreWriteFailure struct{ error }

func (*classicPreWriteFailure) ReadFailedBeforeWrite() {}
func classicObject(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, error) {
	m, err := accessObject(raw, "classic protection object")
	if err != nil {
		return nil, err
	}
	for k := range m {
		if !slices.Contains(keys, k) {
			return nil, fmt.Errorf("unknown classic response member %q", k)
		}
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return nil, fmt.Errorf("missing classic response member %q", k)
		}
	}
	return m, nil
}
func classicBool(m map[string]json.RawMessage, k string) (bool, error) {
	var v *bool
	if json.Unmarshal(m[k], &v) != nil || v == nil {
		return false, fmt.Errorf("missing/null or invalid classic boolean %s", k)
	}
	return *v, nil
}
func classicInt(m map[string]json.RawMessage, k string) (int64, error) {
	var v *int64
	if json.Unmarshal(m[k], &v) != nil || v == nil || *v <= 0 {
		return 0, fmt.Errorf("missing/null or invalid classic identity %s", k)
	}
	return *v, nil
}
func classicNullable[T any](raw json.RawMessage) (settings.ClassicNullable[T], error) {
	var v *T
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return settings.ClassicNullable[T]{}, fmt.Errorf("missing or invalid nullable classic companion")
	}
	return settings.ClassicNullable[T]{Present: true, Value: v}, nil
}

type classicTarget struct {
	id                       string
	archived, squash, rebase bool
}

func decodeClassicTarget(raw json.RawMessage, repo repository.Repository) (classicTarget, error) {
	var t classicTarget
	m, err := classicObject(raw, "id", "nameWithOwner", "isArchived", "isPrivate", "owner", "squashMergeAllowed", "rebaseMergeAllowed")
	if err != nil {
		return t, err
	}
	t.id, err = validateAccessTarget(m, repo, "classic protection")
	if err != nil {
		return t, err
	}
	owner, err := classicObject(m["owner"], "__typename")
	if err != nil {
		return t, err
	}
	kind, err := accessString(owner, "__typename")
	if err != nil || kind != "User" {
		return t, fmt.Errorf("classic protection requires a personally owned repository")
	}
	private, err := classicBool(m, "isPrivate")
	if err != nil || private {
		return t, fmt.Errorf("classic protection requires a public repository")
	}
	t.archived, err = classicBool(m, "isArchived")
	if err != nil {
		return t, err
	}
	t.squash, err = classicBool(m, "squashMergeAllowed")
	if err != nil {
		return t, err
	}
	t.rebase, err = classicBool(m, "rebaseMergeAllowed")
	return t, err
}

var classicOwnedKeys = []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced", "requiresDeployments", "requiredDeploymentEnvironments", "allowsForcePushes", "repository"}
var classicCompanionKeys = []string{"blocksCreations", "dismissesStaleReviews", "lockAllowsFetchAndMerge", "requireLastPushApproval", "requiresApprovingReviews", "requiresCodeOwnerReviews", "requiresStatusChecks", "requiresStrictStatusChecks", "restrictsPushes", "restrictsReviewDismissals", "requiredApprovingReviewCount", "requiredStatusCheckContexts", "requiredStatusChecks"}

func decodeClassicRule(raw json.RawMessage, repo repository.Repository, complete bool) (settings.ClassicState, classicTarget, error) {
	var s settings.ClassicState
	var target classicTarget
	keys := slices.Clone(classicOwnedKeys)
	if complete {
		keys = append(keys, classicCompanionKeys...)
	}
	m, err := classicObject(raw, keys...)
	if err != nil {
		return s, target, err
	}
	s.NodeID, err = accessString(m, "id")
	if err != nil {
		return s, target, err
	}
	id, err := classicInt(m, "databaseId")
	if err != nil {
		return s, target, err
	}
	s.Rule.ID = &id
	s.Rule.BranchNamePattern, err = accessString(m, "pattern")
	if err != nil {
		return s, target, err
	}
	target, err = decodeClassicTarget(m["repository"], repo)
	if err != nil {
		return s, target, err
	}
	s.RepositoryID = target.id
	for k, dest := range map[string]**bool{"requiresConversationResolution": &s.Rule.RequireConversationResolutionBeforeMerging, "requiresCommitSignatures": &s.Rule.RequireSignedCommits, "requiresLinearHistory": &s.Rule.RequireLinearHistory, "allowsDeletions": &s.Rule.AllowDeletions, "lockBranch": &s.Rule.LockBranch, "isAdminEnforced": &s.Rule.DoNotAllowBypassingTheAboveSettings} {
		v, e := classicBool(m, k)
		if e != nil {
			return s, target, e
		}
		*dest = &v
	}
	s.Companions.RequiresDeployments, err = classicBool(m, "requiresDeployments")
	if err != nil {
		return s, target, err
	}
	s.Companions.Deployments, err = classicStringList(m["requiredDeploymentEnvironments"])
	if err != nil {
		return s, target, err
	}
	s.Rule.RequireDeploymentsToSucceedBeforeMerging, err = settings.ClassicDeploymentProjection(s.Companions.RequiresDeployments, s.Companions.Deployments)
	if err != nil {
		return s, target, err
	}
	if err := settings.ValidateClassicRules([]settings.ClassicRule{s.Rule}); err != nil {
		return s, target, err
	}
	s.Companions.AllowsForcePushes, err = classicBool(m, "allowsForcePushes")
	if err != nil {
		return s, target, err
	}
	if !complete {
		return s, target, nil
	}
	c := &s.Companions
	for k, dest := range map[string]*bool{"blocksCreations": &c.BlocksCreations, "dismissesStaleReviews": &c.DismissesStaleReviews, "lockAllowsFetchAndMerge": &c.LockAllowsFetchAndMerge, "requireLastPushApproval": &c.RequireLastPushApproval, "requiresApprovingReviews": &c.RequiresApprovingReviews, "requiresCodeOwnerReviews": &c.RequiresCodeOwnerReviews, "requiresStatusChecks": &c.RequiresStatusChecks, "requiresStrictStatusChecks": &c.RequiresStrictStatusChecks, "restrictsPushes": &c.RestrictsPushes, "restrictsReviewDismissals": &c.RestrictsReviewDismissals} {
		v, e := classicBool(m, k)
		if e != nil {
			return s, target, e
		}
		*dest = v
	}
	c.ReviewCount, err = classicNullable[int](m["requiredApprovingReviewCount"])
	if err != nil {
		return s, target, err
	}
	c.Contexts, err = classicStringList(m["requiredStatusCheckContexts"])
	if err != nil {
		return s, target, err
	}
	checks, err := classicNullable[[]json.RawMessage](m["requiredStatusChecks"])
	if err != nil {
		return s, target, err
	}
	c.Checks.Present = true
	if checks.Value != nil {
		values := make([]settings.ClassicCheck, 0, len(*checks.Value))
		for _, raw := range *checks.Value {
			m, e := classicObject(raw, "context", "app")
			if e != nil {
				return s, target, e
			}
			v := settings.ClassicCheck{}
			var context *string
			if json.Unmarshal(m["context"], &context) != nil || context == nil {
				return s, target, fmt.Errorf("missing or invalid classic check context")
			}
			v.Context = *context
			if !bytes.Equal(bytes.TrimSpace(m["app"]), []byte("null")) {
				a, e := classicObject(m["app"], "id")
				if e != nil {
					return s, target, e
				}
				id, e := accessString(a, "id")
				if e != nil {
					return s, target, e
				}
				v.AppID = &id
			}
			values = append(values, v)
		}
		c.Checks.Value = &values
	}
	return s, target, nil
}
func classicStringList(raw json.RawMessage) (settings.ClassicNullable[[]string], error) {
	values, err := classicNullable[[]*string](raw)
	if err != nil {
		return settings.ClassicNullable[[]string]{}, err
	}
	result := settings.ClassicNullable[[]string]{Present: true}
	if values.Value != nil {
		list := make([]string, 0, len(*values.Value))
		for _, v := range *values.Value {
			if v == nil {
				return result, fmt.Errorf("null classic companion list member")
			}
			list = append(list, *v)
		}
		result.Value = &list
	}
	return result, nil
}
func classicPage(raw json.RawMessage) ([]json.RawMessage, int, string, bool, error) {
	m, err := classicObject(raw, "totalCount", "nodes", "pageInfo")
	if err != nil {
		return nil, 0, "", false, err
	}
	var total *int
	var nodes *[]json.RawMessage
	if json.Unmarshal(m["totalCount"], &total) != nil || total == nil || *total < 0 || json.Unmarshal(m["nodes"], &nodes) != nil || nodes == nil || *nodes == nil {
		return nil, 0, "", false, fmt.Errorf("incomplete classic page")
	}
	page, err := classicObject(m["pageInfo"], "hasNextPage", "endCursor")
	if err != nil {
		return nil, 0, "", false, err
	}
	next, err := classicBool(page, "hasNextPage")
	if err != nil {
		return nil, 0, "", false, err
	}
	var cursor *string
	if json.Unmarshal(page["endCursor"], &cursor) != nil {
		return nil, 0, "", false, fmt.Errorf("invalid classic cursor")
	}
	value := ""
	if cursor != nil {
		value = *cursor
	}
	if next && (value == "" || len(*nodes) == 0) {
		return nil, 0, "", false, fmt.Errorf("incomplete classic pagination")
	}
	return *nodes, *total, value, next, nil
}
func (c *ClassicProtectionClient) Read(repo repository.Repository) (settings.ClassicObservation, error) {
	result := settings.ClassicObservation{Rules: []settings.ClassicState{}}
	cursor := ""
	seen := map[string]bool{}
	expected := -1
	for {
		var response json.RawMessage
		var after any
		if cursor != "" {
			after = cursor
		}
		if err := c.graphql.Do(classicInventoryQuery, map[string]any{"owner": repo.Owner, "name": repo.Name, "cursor": after}, &response); err != nil {
			return result, fmt.Errorf("read classic protection inventory: %w", err)
		}
		top, err := classicObject(response, "repository")
		if err != nil {
			return result, err
		}
		fields, err := accessObject(top["repository"], "classic repository")
		if err != nil {
			return result, err
		}
		connection, ok := fields["branchProtectionRules"]
		if !ok {
			return result, fmt.Errorf("missing classic rule inventory")
		}
		delete(fields, "branchProtectionRules")
		raw, _ := json.Marshal(fields)
		target, err := decodeClassicTarget(raw, repo)
		if err != nil {
			return result, err
		}
		if result.RepositoryID != "" && (result.RepositoryID != target.id || result.Archived != target.archived) {
			return result, fmt.Errorf("classic repository identity changed during pagination")
		}
		result.RepositoryID, result.Archived = target.id, target.archived
		nodes, total, end, next, err := classicPage(connection)
		if err != nil {
			return result, err
		}
		if expected >= 0 && expected != total {
			return result, fmt.Errorf("classic inventory count changed")
		}
		expected = total
		for _, raw := range nodes {
			s, t, err := decodeClassicRule(raw, repo, true)
			if err != nil {
				return result, err
			}
			if t.id != target.id || t.archived != target.archived {
				return result, fmt.Errorf("classic rule repository changed")
			}
			if err := c.readClassicActors(repo, &s, target); err != nil {
				return result, err
			}
			result.Rules = append(result.Rules, s)
		}
		if len(result.Rules) > total || (!next && len(result.Rules) != total) {
			return result, fmt.Errorf("incomplete classic inventory")
		}
		if !next {
			break
		}
		if seen[end] {
			return result, fmt.Errorf("repeated classic inventory cursor")
		}
		seen[end] = true
		cursor = end
	}
	return result, result.Validate()
}
func (c *ClassicProtectionClient) readClassicActors(repo repository.Repository, s *settings.ClassicState, expectedTarget classicTarget) error {
	groups := []struct {
		name   string
		values *[]settings.ClassicActor
	}{{"pushAllowances", &s.Companions.PushActors}, {"reviewDismissalAllowances", &s.Companions.ReviewDismissalActors}, {"bypassForcePushAllowances", &s.Companions.ForcePushActors}, {"bypassPullRequestAllowances", &s.Companions.PullRequestActors}}
	for _, group := range groups {
		values := []settings.ClassicActor{}
		cursor := ""
		expected := -1
		seen := map[string]bool{}
		query := `query ClassicProtectionActors($id:ID!,$cursor:String){node(id:$id){__typename ... on BranchProtectionRule{id databaseId repository{` + classicTargetFields + `} ` + group.name + `(first:100,after:$cursor){totalCount pageInfo{hasNextPage endCursor} nodes{actor{__typename ... on User{id} ... on Team{id} ... on App{id}}}}}}}`
		for {
			var response json.RawMessage
			var after any
			if cursor != "" {
				after = cursor
			}
			if err := c.graphql.Do(query, map[string]any{"id": s.NodeID, "cursor": after}, &response); err != nil {
				return fmt.Errorf("read classic actor inventory: %w", err)
			}
			top, err := classicObject(response, "node")
			if err != nil {
				return err
			}
			m, err := classicObject(top["node"], "__typename", "id", "databaseId", "repository", group.name)
			if err != nil {
				return err
			}
			kind, e := accessString(m, "__typename")
			if e != nil || kind != "BranchProtectionRule" {
				return fmt.Errorf("wrong classic node type")
			}
			id, e := accessString(m, "id")
			if e != nil || id != s.NodeID {
				return fmt.Errorf("classic actor parent changed")
			}
			db, e := classicInt(m, "databaseId")
			if e != nil || db != *s.Rule.ID {
				return fmt.Errorf("classic actor parent database identity changed")
			}
			target, e := decodeClassicTarget(m["repository"], repo)
			if e != nil || target.id != s.RepositoryID || target.archived != expectedTarget.archived {
				return fmt.Errorf("classic actor repository changed")
			}
			nodes, total, end, next, e := classicPage(m[group.name])
			if e != nil {
				return e
			}
			if expected >= 0 && expected != total {
				return fmt.Errorf("classic actor inventory changed")
			}
			expected = total
			for _, raw := range nodes {
				entry, e := classicObject(raw, "actor")
				if e != nil {
					return e
				}
				actor, e := classicObject(entry["actor"], "__typename", "id")
				if e != nil {
					return e
				}
				kind, e := accessString(actor, "__typename")
				if e != nil {
					return e
				}
				id, e := accessString(actor, "id")
				if e != nil {
					return e
				}
				values = append(values, settings.ClassicActor{Kind: kind, ID: id})
			}
			if len(values) > total || (!next && len(values) != total) {
				return fmt.Errorf("incomplete classic actor inventory")
			}
			if !next {
				break
			}
			if seen[end] {
				return fmt.Errorf("repeated classic actor cursor")
			}
			seen[end] = true
			cursor = end
		}
		slices.SortFunc(values, func(a, b settings.ClassicActor) int {
			if a.Kind != b.Kind {
				return strings.Compare(a.Kind, b.Kind)
			}
			return strings.Compare(a.ID, b.ID)
		})
		*group.values = values
	}
	var err error
	s.Rule.AllowForcePushes, err = settings.ClassicForcePushProjection(s.Companions.AllowsForcePushes, s.Companions.ForcePushActors)
	return err
}
func (c *ClassicProtectionClient) readOne(repo repository.Repository, id string) (settings.ClassicState, classicTarget, error) {
	var response json.RawMessage
	var empty settings.ClassicState
	var target classicTarget
	if err := c.graphql.Do(classicNodeQuery, map[string]any{"id": id}, &response); err != nil {
		return empty, target, err
	}
	top, err := classicObject(response, "node")
	if err != nil {
		return empty, target, err
	}
	m, err := accessObject(top["node"], "classic rule")
	if err != nil {
		return empty, target, err
	}
	kind, err := accessString(m, "__typename")
	if err != nil || kind != "BranchProtectionRule" {
		return empty, target, fmt.Errorf("wrong classic rule node type")
	}
	delete(m, "__typename")
	raw, _ := json.Marshal(m)
	s, target, err := decodeClassicRule(raw, repo, true)
	if err != nil {
		return s, target, err
	}
	if s.NodeID != id {
		return s, target, fmt.Errorf("classic rule node identity changed")
	}
	if err = c.readClassicActors(repo, &s, target); err != nil {
		return s, target, err
	}
	err = (settings.ClassicObservation{RepositoryID: target.id, Archived: target.archived, Rules: []settings.ClassicState{s}}).Validate()
	return s, target, err
}
func (c *ClassicProtectionClient) Apply(repo repository.Repository, selection settings.ClassicSelection) (settings.ClassicWriteResult, error) {
	result := settings.ClassicWriteResult{}
	pre := func(e error) (settings.ClassicWriteResult, error) { return result, &classicPreWriteFailure{e} }
	if selection.RepositoryID == "" {
		return pre(fmt.Errorf("missing acquired classic repository identity"))
	}
	desired := selection.Desired
	if err := settings.ValidateClassicRules([]settings.ClassicRule{desired}); err != nil {
		return pre(err)
	}
	var before *settings.ClassicState
	var target classicTarget
	known := map[int64]bool{}
	knownNodes := map[string]bool{}
	if selection.Before != nil {
		initial := selection.Before
		if initial.Rule.ID == nil || desired.ID == nil || *initial.Rule.ID != *desired.ID {
			return pre(fmt.Errorf("invalid classic update selection"))
		}
		fresh, t, err := c.readOne(repo, initial.NodeID)
		if err != nil {
			return pre(err)
		}
		target = t
		if target.id != selection.RepositoryID {
			return pre(fmt.Errorf("classic repository identity changed"))
		}
		if fresh.RepositoryID != initial.RepositoryID || *fresh.Rule.ID != *initial.Rule.ID || fresh.Rule.BranchNamePattern != initial.Rule.BranchNamePattern {
			return pre(fmt.Errorf("classic update identity or original pattern changed"))
		}
		before = &fresh
		result.ID, result.NodeID = *fresh.Rule.ID, fresh.NodeID
		if target.archived {
			return pre(fmt.Errorf("archived repository cannot update classic protection"))
		}
		if settings.ClassicMatches(fresh.Rule, desired) {
			return result, nil
		}
		if desired.BranchNamePattern != fresh.Rule.BranchNamePattern {
			inventory, err := c.Read(repo)
			if err != nil {
				return pre(err)
			}
			if inventory.RepositoryID != target.id || inventory.Archived != target.archived {
				return pre(fmt.Errorf("classic target changed before rename"))
			}
			if _, err := settings.PrepareClassicRules(inventory, []settings.ClassicRule{desired}); err != nil {
				return pre(err)
			}
		}
	} else {
		if desired.ID != nil {
			return pre(fmt.Errorf("classic creation cannot adopt an id"))
		}
		inventory, err := c.Read(repo)
		if err != nil {
			return pre(err)
		}
		if _, err := settings.PrepareClassicRules(inventory, []settings.ClassicRule{desired}); err != nil {
			return pre(err)
		}
		target.id, target.archived = inventory.RepositoryID, inventory.Archived
		for _, s := range inventory.Rules {
			known[*s.Rule.ID] = true
			knownNodes[s.NodeID] = true
		}
		// The complete repository query provides fresh merge-method prerequisites even
		// for an empty inventory. Fetch those values without relying on rule existence.
		var response json.RawMessage
		query := `query ClassicProtectionTarget($owner:String!,$name:String!){repository(owner:$owner,name:$name,followRenames:false){` + classicTargetFields + `}}`
		if err := c.graphql.Do(query, map[string]any{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
			return pre(err)
		}
		m, err := classicObject(response, "repository")
		if err != nil {
			return pre(err)
		}
		t, err := decodeClassicTarget(m["repository"], repo)
		if err != nil {
			return pre(err)
		}
		if t.id != target.id || t.archived != target.archived {
			return pre(fmt.Errorf("classic creation target changed"))
		}
		target = t
		if target.id != selection.RepositoryID {
			return pre(fmt.Errorf("classic repository identity changed"))
		}
		if target.archived {
			return pre(fmt.Errorf("archived repository cannot create classic protection"))
		}
	}
	if desired.RequireLinearHistory != nil && *desired.RequireLinearHistory && (before == nil || !*before.Rule.RequireLinearHistory) && !target.squash && !target.rebase {
		return pre(fmt.Errorf("linear history requires squash or rebase merging"))
	}
	if err := c.classicForcePushIdentities(desired.AllowForcePushes); err != nil {
		return pre(err)
	}
	var references map[string]string
	group := desired.RequireDeploymentsToSucceedBeforeMerging
	if settings.ClassicDeploymentsSelected(group) && *group.Enabled && len(*group.Environments) > 0 {
		var err error
		references, err = c.classicDeploymentReferences(repo, target, *group.Environments)
		if err != nil {
			return pre(err)
		}
	}
	input := map[string]any{}
	mutation, kind := "createBranchProtectionRule", "CreateBranchProtectionRuleInput"
	if before == nil {
		input["repositoryId"] = target.id
		input["pattern"] = desired.BranchNamePattern
	} else {
		mutation, kind = "updateBranchProtectionRule", "UpdateBranchProtectionRuleInput"
		input["branchProtectionRuleId"] = before.NodeID
		if before.Rule.BranchNamePattern != desired.BranchNamePattern {
			input["pattern"] = desired.BranchNamePattern
		}
	}
	names := []string{"requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced"}
	flags := settings.ClassicFlags(desired)
	var existing []*bool
	if before != nil {
		existing = settings.ClassicFlags(before.Rule)
	}
	for i, v := range flags {
		if v != nil && (before == nil || *v != *existing[i]) {
			input[names[i]] = *v
		}
	}
	if settings.ClassicDeploymentsSelected(group) {
		if before == nil || *group.Enabled != before.Companions.RequiresDeployments {
			input["requiresDeployments"] = *group.Enabled
		}
		if *group.Enabled && (before == nil || !classicEnvironmentSetsEqual(*group.Environments, before.Companions.Deployments.Value)) {
			input["requiredDeploymentEnvironments"] = *group.Environments
		}
	}
	classicForcePushInputs(input, before, desired.AllowForcePushes)
	query := `mutation ClassicProtectionWrite($input:` + kind + `!){` + mutation + `(input:$input){branchProtectionRule{` + classicOwnedFields + ` repository{` + classicTargetFields + `}}}}`
	var response json.RawMessage
	mutationErr := c.graphql.Do(query, map[string]any{"input": input}, &response)
	// go-gh decodes partial data before returning GraphQL resolver errors. Retain
	// reported identities for recovery even when the mutation is unconfirmed.
	if node, id := classicMutationRecovery(response, mutation); node != "" || id > 0 {
		if node != "" {
			result.NodeID = node
		}
		if id > 0 {
			result.ID = id
		}
	}
	if mutationErr != nil {
		return result, fmt.Errorf("classic mutation failed (effects may have applied; recovery node %q, id %d): %w", result.NodeID, result.ID, mutationErr)
	}
	result.Wrote = true
	top, err := classicObject(response, mutation)
	if err != nil {
		return result, err
	}
	body, err := classicObject(top[mutation], "branchProtectionRule")
	if err != nil {
		return result, err
	}
	ack, t, err := decodeClassicRule(body["branchProtectionRule"], repo, false)
	if err != nil {
		return result, fmt.Errorf("classic mutation acknowledgment invalid (recovery node %q, id %d): %w", result.NodeID, result.ID, err)
	}
	if t.id != target.id || t.archived || !classicAcknowledgmentMatches(ack, desired) {
		return result, fmt.Errorf("classic mutation acknowledgment differs from request")
	}
	if before != nil && (ack.NodeID != before.NodeID || *ack.Rule.ID != *before.Rule.ID) {
		return result, fmt.Errorf("classic update acknowledgment changed identity")
	}
	if before == nil && (known[*ack.Rule.ID] || knownNodes[ack.NodeID]) {
		return result, fmt.Errorf("classic creation returned an existing identity; refusing adoption")
	}
	actual, t, err := c.readOne(repo, ack.NodeID)
	if err != nil {
		return result, fmt.Errorf("verify classic mutation (effects may have applied): %w", err)
	}
	if t.id != target.id || t.archived || actual.NodeID != ack.NodeID || *actual.Rule.ID != *ack.Rule.ID || !settings.ClassicMatches(actual.Rule, desired) {
		return result, fmt.Errorf("classic mutation readback identity or requested controls differ")
	}
	if before != nil {
		if err := settings.VerifyClassicPreservation(*before, actual, desired); err != nil {
			return result, err
		}
	}
	if references != nil {
		fresh, err := c.classicDeploymentReferences(repo, target, *group.Environments)
		if err != nil {
			return result, fmt.Errorf("verify classic deployment references (effects may have applied): %w", err)
		}
		if !reflect.DeepEqual(references, fresh) {
			return result, fmt.Errorf("classic deployment references changed after mutation (effects may have applied)")
		}
	}
	return result, nil
}

// Recovery is not confirmation: require only the typed identity fields, then
// leave full acknowledgment validation and independent readback to Apply.
func classicMutationRecovery(response json.RawMessage, mutation string) (string, int64) {
	top, err := accessObject(response, "classic mutation")
	if err != nil {
		return "", 0
	}
	body, err := accessObject(top[mutation], "classic mutation payload")
	if err != nil {
		return "", 0
	}
	rule, err := accessObject(body["branchProtectionRule"], "classic mutation rule")
	if err != nil {
		return "", 0
	}
	node, _ := accessString(rule, "id")
	id, _ := classicInt(rule, "databaseId")
	return node, id
}
