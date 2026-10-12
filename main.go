package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"go.yaml.in/yaml/v4"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, exportCurrent))
}

func run(args []string, stdout, stderr io.Writer, export func() ([]byte, error)) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := fmt.Fprintln(stdout, "Usage: gh settings export\nExport the current repository's five basic settings as YAML to standard output."); err != nil {
			fmt.Fprintf(stderr, "gh settings: write help: %v\n", err)
			return 1
		}
		return 0
	}
	if len(args) != 1 || args[0] != "export" {
		fmt.Fprintln(stderr, "gh settings: unsupported command or arguments; usage: gh settings export")
		return 1
	}
	data, err := export()
	if err != nil {
		fmt.Fprintf(stderr, "gh settings: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "gh settings: write output: %v\n", err)
		return 1
	}
	return 0
}

func exportCurrent() ([]byte, error) {
	repo, err := currentRepository()
	if err != nil {
		return nil, err
	}
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	return exportSettings(repo, client)
}

func exportSettings(repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	var response map[string]json.RawMessage
	if err := client.Get("repos/"+repo.Owner+"/"+repo.Name, &response); err != nil {
		return nil, fmt.Errorf("read repository settings: %w", err)
	}
	var result settings
	fields := []struct {
		name   string
		target any
	}{
		{"description", &result.Description}, {"homepage", &result.Homepage},
		{"has_issues", &result.HasIssues}, {"has_projects", &result.HasProjects}, {"has_wiki", &result.HasWiki},
	}
	for _, field := range fields {
		value, present := response[field.name]
		if !present {
			return nil, fmt.Errorf("GitHub response is missing %s", field.name)
		}
		if string(value) == "null" {
			if field.name == "description" || field.name == "homepage" {
				// GET uses null for unset strings; export uses the update schema's empty string.
				continue
			}
			return nil, fmt.Errorf("GitHub returned null for %s", field.name)
		}
		if err := json.Unmarshal(value, field.target); err != nil {
			return nil, fmt.Errorf("decode %s: %w", field.name, err)
		}
	}
	return yaml.Marshal(result)
}

type settings struct {
	Description string `yaml:"description"`
	Homepage    string `yaml:"homepage"`
	HasIssues   bool   `yaml:"has_issues"`
	HasProjects bool   `yaml:"has_projects"`
	HasWiki     bool   `yaml:"has_wiki"`
}

func currentRepository() (repository.Repository, error) {
	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return repository.Repository{}, fmt.Errorf("resolve current repository: a GitHub origin remote is required: %w", err)
	}
	remoteURL := strings.TrimSpace(string(remote))
	if !strings.HasPrefix(remoteURL, "https://") && !strings.HasPrefix(remoteURL, "ssh://") && !strings.HasPrefix(remoteURL, "git@github.com:") {
		return repository.Repository{}, fmt.Errorf("origin must be a GitHub HTTPS or SSH URL")
	}
	repo, err := repository.Parse(remoteURL)
	if err != nil {
		return repository.Repository{}, fmt.Errorf("parse origin remote: %w", err)
	}
	if repo.Host != "github.com" {
		return repository.Repository{}, fmt.Errorf("origin must point to github.com, got %q", repo.Host)
	}
	return repo, nil
}
