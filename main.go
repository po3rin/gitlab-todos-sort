package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

type ToDo struct {
	TargetURL string    `json:"target_url"`
	CreatedAt time.Time `json:"created_at"`
	State     string    `json:"state"`
	Body      string    `json:"body"`
	Target    Target    `json:"target"`
	Project   Project   `json:"project"`

	// additional meta data
	Score float64
}

type Target struct {
	IID         int    `json:"iid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Draft       bool   `json:"draft"`
	Autor       Autor  `json:"author"`
}

type Autor struct {
	Username string `json:"username"`
}

type Project struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
}

type Commit struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	CommitterEmail string    `json:"committer_email"`
}

type Diff struct {
	NewPath string
}

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

func onlyOpenState(todos []ToDo) []ToDo {
	openTodos := []ToDo{}
	for _, todo := range todos {
		if todo.Target.State == "opened" {
			openTodos = append(openTodos, todo)
		}
	}
	return openTodos
}

func rmDraft(todos []ToDo) []ToDo {
	openTodos := []ToDo{}
	for _, todo := range todos {
		if !todo.Target.Draft {
			openTodos = append(openTodos, todo)
		}
	}
	return openTodos
}

var urgentKeywords = []string{"URGENT", "EMERGENCY", "緊急", "重要", "急ぎ"}

func addUrgentScore(todos []ToDo) {
	for i := range todos {
		for _, keyword := range urgentKeywords {
			if strings.Contains(todos[i].Body, keyword) || strings.Contains(todos[i].Target.Title, keyword) {
				todos[i].Score += 1000
			}
		}
	}
}

func addCreatedAtScore(todos []ToDo) {
	for i := range todos {
		score := math.Exp(time.Since(todos[i].CreatedAt).Hours())
		if score > 300 {
			score = 300
		}
		todos[i].Score += score
	}
}

func addUserNamePosScore(todos []ToDo, userName string) {
	for i := range todos {
		userNamePos := strings.LastIndex(todos[i].Body, fmt.Sprintf("@%s", userName))
		if userNamePos == -1 {
			continue
		}
		userNameOrder := float64(strings.Count(string([]rune(todos[i].Body)[:userNamePos]), "@"))
		userNum := float64(strings.Count(todos[i].Body, "@"))
		todos[i].Score += (30 * (1 - (userNameOrder / userNum))) + (50 / userNum)
	}
}

func addCommitScore(gitlabClient *GitLabClient, userName string, todos []ToDo) error {
	for i := range todos {
		commits, err := gitlabClient.commits(todos[i].Project.ID)
		if err != nil {
			return fmt.Errorf("failed to get commits: %w", err)
		}
		var commitiByUserNum float64
		for _, commit := range commits {
			if strings.Contains(commit.CommitterEmail, userName) {
				commitiByUserNum += 1
			}
		}
		commitsNum := float64(len(commits))
		todos[i].Score += 100 * commitiByUserNum / commitsNum
	}
	return nil
}

func addDiffScore(gitlabClient *GitLabClient, todos []ToDo, priorityFileExt []string) error {
	for i := range todos {
		diffs, err := gitlabClient.diffs(todos[i].Project.ID, fmt.Sprintf("%d", todos[i].Target.IID))
		if err != nil {
			return fmt.Errorf("failed to get diffs: %w", err)
		}

		var diffScore int
		for _, diff := range diffs {
			for _, ext := range priorityFileExt {
				if strings.HasSuffix(diff.NewPath, ext) {
					diffScore += 10
				}
			}
		}

		if diffScore > 50 {
			diffScore = 50
		}
		todos[i].Score += float64(diffScore)
	}
	return nil
}

func sortByScore(todos []ToDo) {
	sort.Slice(todos, func(i, j int) bool {
		return todos[i].Score > todos[j].Score
	})
}

func main() {
	token, ok := os.LookupEnv("GITLAB_TOKEN")
	if !ok {
		log.Fatal("GITLAB_TOKEN is not set")
	}
	host, ok := os.LookupEnv("GITLAB_HOST")
	if !ok {
		log.Fatal("GITLAB_TOKEN is not set")
	}
	user, ok := os.LookupEnv("GITLAB_USER_NAME")
	if !ok {
		log.Fatal("GITLAB_TOKEN is not set")
	}

	g := NewGitLabClient(host, token)

	todos, err := g.todos()
	if err != nil {
		log.Fatal(err)
	}

	todos = rmDraft(todos)
	todos = onlyOpenState(todos)
	addUserNamePosScore(todos, user)
	addUrgentScore(todos)
	addCreatedAtScore(todos)

	err = addDiffScore(g, todos, []string{".go", ".tf", ".py"}) // TODO: configurable
	if err != nil {
		log.Fatal(err)
	}

	err = addCommitScore(g, user, todos)
	if err != nil {
		log.Fatal(err)
	}

	sortByScore(todos)

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', tabwriter.TabIndent)
	w.Write([]byte("url\tscore\n"))
	for _, todo := range todos {
		fmt.Fprintf(w, "%s\t%v\n", todo.TargetURL, todo.Score)
	}
	w.Flush()
}
