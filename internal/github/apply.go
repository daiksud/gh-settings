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

// NewWriteClient retains the write timeout and rejects redirects for writes.
func NewWriteClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{
		Host: host, Timeout: 30 * time.Second,
		// Redirects can convert writes to GET and report success without updating.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	return &Client{rest: rest}, nil
}

func (client *Client) ApplyRepository(repo repository.Repository, desired settings.RepositoryDeclaration) error {
	payload := struct {
		Description              *string `json:"description,omitempty"`
		Homepage                 *string `json:"homepage,omitempty"`
		HasIssues                *bool   `json:"has_issues,omitempty"`
		HasProjects              *bool   `json:"has_projects,omitempty"`
		HasWiki                  *bool   `json:"has_wiki,omitempty"`
		AllowMergeCommit         *bool   `json:"allow_merge_commit,omitempty"`
		AllowSquashMerge         *bool   `json:"allow_squash_merge,omitempty"`
		AllowRebaseMerge         *bool   `json:"allow_rebase_merge,omitempty"`
		DeleteBranchOnMerge      *bool   `json:"delete_branch_on_merge,omitempty"`
		AllowAutoMerge           *bool   `json:"allow_auto_merge,omitempty"`
		AllowUpdateBranch        *bool   `json:"allow_update_branch,omitempty"`
		SquashMergeCommitTitle   *string `json:"squash_merge_commit_title,omitempty"`
		SquashMergeCommitMessage *string `json:"squash_merge_commit_message,omitempty"`
	}{
		Description: desired.Description, Homepage: desired.Homepage,
		HasIssues: desired.HasIssues, HasProjects: desired.HasProjects, HasWiki: desired.HasWiki,
		AllowMergeCommit: desired.AllowMergeCommit, AllowSquashMerge: desired.AllowSquashMerge, AllowRebaseMerge: desired.AllowRebaseMerge, DeleteBranchOnMerge: desired.DeleteBranchOnMerge, AllowAutoMerge: desired.AllowAutoMerge, AllowUpdateBranch: desired.AllowUpdateBranch,
		SquashMergeCommitTitle: desired.SquashMergeCommitTitle, SquashMergeCommitMessage: desired.SquashMergeCommitMessage,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode repository settings: %w", err)
	}
	if err := client.rest.Patch("repos/"+repo.Owner+"/"+repo.Name, bytes.NewReader(body), nil); err != nil {
		return fmt.Errorf("apply repository settings: %w", err)
	}
	return nil
}
