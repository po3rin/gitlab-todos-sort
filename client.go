package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GitLabClient struct {
	host        string
	accessToken string
}

func NewGitLabClient(host, accessToken string) *GitLabClient {
	return &GitLabClient{
		host:        host,
		accessToken: accessToken,
	}
}

func (g *GitLabClient) todos() ([]ToDo, error) {
	res, err := http.Get(fmt.Sprintf("https://%s/api/v4/todos?access_token=%s", g.host, g.accessToken))
	if err != nil {
		return nil, fmt.Errorf("failed to get response from GitLab API: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from GitLab API: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get response from GitLab API: %s", body)
	}

	var todos []ToDo
	if err := json.Unmarshal(body, &todos); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body from GitLab API: %w", err)
	}

	return todos, nil
}

func (g *GitLabClient) commits(id int) ([]Commit, error) {
	targetDate := time.Now().AddDate(0, 0, -365).Format("2006-01-02T15:04:05Z")
	res, err := http.Get(fmt.Sprintf("https://%s/api/v4/projects/%v/repository/commits?since=%s&access_token=%s", g.host, id, targetDate, g.accessToken))
	if err != nil {
		return nil, fmt.Errorf("failed to get commits in project %v from GitLab API: %w", id, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from GitLab API: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get unexpected status in project %v from GitLab API: %s", id, string(body))
	}

	var commits []Commit
	if err := json.Unmarshal(body, &commits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body from GitLab API: %w", err)
	}

	return commits, nil
}

func (g *GitLabClient) diffs(id int, iid string) ([]Diff, error) {
	res, err := http.Get(fmt.Sprintf("https://%s/api/v4/projects/%v/merge_requests/%s/diffs?access_token=%s", g.host, id, iid, g.accessToken))
	if err != nil {
		return nil, fmt.Errorf("failed to get diffs in project %v from GitLab API: %w", id, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from GitLab API: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get unexpected status in project %v from GitLab API: %s", id, string(body))
	}

	var diffs []Diff
	if err := json.Unmarshal(body, &diffs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body from GitLab API: %w", err)
	}

	return diffs, nil
}
