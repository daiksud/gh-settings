package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// NewWriteClient retains the write timeout and rejects redirects throughout
// repository reconciliation, including its fresh reads.
func NewWriteClient(host string) (*Client, error) {
	return newWriteClient(api.ClientOptions{Host: host})
}

func newWriteClient(options api.ClientOptions) (*Client, error) {
	options.Timeout = 30 * time.Second
	// Redirects can change the target or convert writes to GET.
	options.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	rest, err := api.NewRESTClient(options)
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	graphql, err := api.NewGraphQLClient(options)
	if err != nil {
		return nil, fmt.Errorf("configure GitHub commit message client: %w", err)
	}
	return &Client{rest: rest, commitMessages: &MergeMethodsClient{graphql: graphql}}, nil
}

// ApplyRepository reconciles the selected REST controls with fresh observations.
// The bool records a successful PATCH acknowledgment, including a later failure.
// A false result with an error does not establish that no mutation occurred.
func (client *Client) ApplyRepository(repo repository.Repository, desired settings.RepositoryDeclaration) (bool, error) {
	if desired.BaseEmpty() && desired.MergeMethodsEmpty() {
		return false, nil
	}
	before, beforeMenus, err := client.readRepositoryPatch(repo, desired)
	if err != nil {
		return false, &repositoryPreWriteFailure{err}
	}
	patch, err := settings.RepositoryPatchChanges(before, desired)
	if err != nil {
		return false, &repositoryPreWriteFailure{err}
	}
	client.reportCommitMessageNormalizations(beforeMenus)
	if patch.Empty() {
		return false, nil
	}
	return client.patchRepository(repo, desired, patch, beforeMenus)
}

func (client *Client) readRepositoryPatch(repo repository.Repository, desired settings.RepositoryDeclaration) (settings.RepositoryDeclaration, repositoryCommitObservations, error) {
	var observed settings.RepositoryDeclaration
	if !desired.BaseEmpty() {
		base, err := client.ReadRepository(repo)
		if err != nil {
			return settings.RepositoryDeclaration{}, nil, err
		}
		observed = base.Declaration()
	}
	if desired.MergeMethodsEmpty() {
		return observed, nil, nil
	}
	if client.commitMessages == nil || client.commitMessages.graphql == nil {
		return settings.RepositoryDeclaration{}, nil, fmt.Errorf("repository merge verification client is not configured")
	}
	return client.commitMessages.readRepositoryPatchMerge(repo, desired, observed)
}

func (client *Client) patchRepository(repo repository.Repository, original, desired settings.RepositoryDeclaration, beforeMenus repositoryCommitObservations) (bool, error) {
	payload := struct {
		Description              *string `json:"description,omitempty"`
		Homepage                 *string `json:"homepage,omitempty"`
		DefaultBranch            *string `json:"default_branch,omitempty"`
		HasIssues                *bool   `json:"has_issues,omitempty"`
		HasProjects              *bool   `json:"has_projects,omitempty"`
		HasWiki                  *bool   `json:"has_wiki,omitempty"`
		HasDiscussions           *bool   `json:"has_discussions,omitempty"`
		WebCommitSignoffRequired *bool   `json:"web_commit_signoff_required,omitempty"`
		IsTemplate               *bool   `json:"is_template,omitempty"`
		AllowMergeCommit         *bool   `json:"allow_merge_commit,omitempty"`
		AllowSquashMerge         *bool   `json:"allow_squash_merge,omitempty"`
		AllowRebaseMerge         *bool   `json:"allow_rebase_merge,omitempty"`
		DeleteBranchOnMerge      *bool   `json:"delete_branch_on_merge,omitempty"`
		AllowAutoMerge           *bool   `json:"allow_auto_merge,omitempty"`
		AllowUpdateBranch        *bool   `json:"allow_update_branch,omitempty"`
		SquashMergeCommitTitle   *string `json:"squash_merge_commit_title,omitempty"`
		SquashMergeCommitMessage *string `json:"squash_merge_commit_message,omitempty"`
		MergeCommitTitle         *string `json:"merge_commit_title,omitempty"`
		MergeCommitMessage       *string `json:"merge_commit_message,omitempty"`
	}{
		Description: desired.Description, Homepage: desired.Homepage, DefaultBranch: desired.DefaultBranch,
		HasIssues: desired.HasIssues, HasProjects: desired.HasProjects, HasWiki: desired.HasWiki, HasDiscussions: desired.HasDiscussions, WebCommitSignoffRequired: desired.WebCommitSignoffRequired, IsTemplate: desired.IsTemplate,
		AllowMergeCommit: desired.AllowMergeCommit, AllowSquashMerge: desired.AllowSquashMerge, AllowRebaseMerge: desired.AllowRebaseMerge, DeleteBranchOnMerge: desired.DeleteBranchOnMerge, AllowAutoMerge: desired.AllowAutoMerge, AllowUpdateBranch: desired.AllowUpdateBranch,
		SquashMergeCommitTitle: desired.SquashMergeCommitTitle, SquashMergeCommitMessage: desired.SquashMergeCommitMessage,
		MergeCommitTitle: desired.MergeCommitTitle, MergeCommitMessage: desired.MergeCommitMessage,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, &repositoryPreWriteFailure{fmt.Errorf("encode repository settings: %w", err)}
	}
	var response map[string]json.RawMessage
	var result any
	if desired.HasDiscussions != nil || desired.DefaultBranch != nil {
		result = &response
	}
	if err := client.rest.Patch("repos/"+repo.Owner+"/"+repo.Name, bytes.NewReader(body), result); err != nil {
		return false, fmt.Errorf("apply repository settings: %w", err)
	}
	if desired.HasDiscussions != nil {
		var actual *bool
		value, present := response["has_discussions"]
		if !present || json.Unmarshal(value, &actual) != nil || actual == nil {
			return true, fmt.Errorf("apply repository settings: GitHub response lacks boolean has_discussions (repository writes may already have applied)")
		}
		if *actual != *desired.HasDiscussions {
			return true, fmt.Errorf("apply repository settings: GitHub returned has_discussions=%t, expected %t (repository writes may already have applied)", *actual, *desired.HasDiscussions)
		}
	}
	if desired.DefaultBranch != nil {
		var actual *string
		value, present := response["default_branch"]
		if !present || json.Unmarshal(value, &actual) != nil || actual == nil || !settings.ValidDefaultBranch(*actual) {
			return true, fmt.Errorf("apply repository settings: GitHub response lacks valid default_branch (repository writes may already have applied)")
		}
		if *actual != *desired.DefaultBranch {
			return true, fmt.Errorf("apply repository settings: GitHub returned default_branch=%q, expected %q (repository writes may already have applied)", *actual, *desired.DefaultBranch)
		}
	}
	actual, actualMenus, err := client.readRepositoryPatch(repo, original)
	if err != nil {
		return true, fmt.Errorf("verify repository settings (repository writes may already have applied): %w", err)
	}
	client.reportCommitMessageNormalizations(actualMenus)
	remaining, err := settings.RepositoryPatchChanges(actual, original)
	if err != nil {
		return true, fmt.Errorf("verify repository settings (repository writes may already have applied): %w", err)
	}
	if !remaining.Empty() || !repositoryCommitMessagesMatch(beforeMenus, actualMenus, original) {
		return true, fmt.Errorf("GitHub repository settings readback does not match declared values or preserved companions (repository writes may already have applied)")
	}
	return true, nil
}

type repositoryPreWriteFailure struct{ cause error }

func (failure *repositoryPreWriteFailure) Error() string  { return failure.cause.Error() }
func (failure *repositoryPreWriteFailure) Unwrap() error  { return failure.cause }
func (*repositoryPreWriteFailure) ReadFailedBeforeWrite() {}

func (client *Client) reportCommitMessageNormalizations(observed repositoryCommitObservations) {
	if client.CommitMessageNormalizationWarning == nil {
		return
	}
	for _, menu := range []settings.CommitMessageMenu{settings.SquashCommitMenu, settings.MergeCommitMenu} {
		if normalization := observed[menu].Normalization; normalization != (settings.CommitMessageNormalization{}) {
			client.CommitMessageNormalizationWarning(normalization)
		}
	}
}
