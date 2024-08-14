package main

import (
	"flag"
	"fmt"
	"log"
	"math"
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

func urgentScore(todos []ToDo) map[int]float64 {
	result := make(map[int]float64)
	for _, t := range todos {
		for _, keyword := range urgentKeywords {
			if strings.Contains(t.Body, keyword) || strings.Contains(t.Target.Title, keyword) {
				result[t.Target.IID] = 1000
			}
		}
	}
	return result
}

func createdAtScore(todos []ToDo) map[int]float64 {
	result := make(map[int]float64)
	for _, t := range todos {
		score := math.Exp(time.Since(t.CreatedAt).Hours())
		if score > 300 {
			score = 300
		}
		result[t.Target.IID] = score
	}
	return result
}

func userNamePosScore(todos []ToDo, userName string) map[int]float64 {
	result := make(map[int]float64)
	for _, t := range todos {
		userNamePos := strings.LastIndex(t.Body, fmt.Sprintf("@%s", userName))
		if userNamePos == -1 {
			continue
		}
		userNameOrder := float64(strings.Count(string([]rune(t.Body)[:userNamePos]), "@"))
		userNum := float64(strings.Count(t.Body, "@"))
		result[t.Target.IID] = (30 * (1 - (userNameOrder / userNum))) + (50 / userNum)
	}
	return result
}

func commitScore(gitlabClient *GitLabClient, userName string, todos []ToDo) (map[int]float64, error) {
	result := make(map[int]float64)
	ProjectScoreMap := make(map[int]float64)
	for _, t := range todos {
		score, ok := ProjectScoreMap[t.Project.ID]
		if !ok {
			commits, err := gitlabClient.commits(t.Project.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get commits: %w", err)
			}

			var commitiByUserNum float64
			for _, commit := range commits {
				if strings.Contains(commit.CommitterEmail, userName) {
					commitiByUserNum += 1
				}
			}

			commitsNum := float64(len(commits))

			score = 100 * commitiByUserNum / commitsNum
			ProjectScoreMap[t.Project.ID] = score
		}
		result[t.Target.IID] = score
	}
	return result, nil
}

func diffScore(gitlabClient *GitLabClient, todos []ToDo, priorityFileExt []string) (map[int]float64, error) {
	result := make(map[int]float64)
	for _, t := range todos {
		diffs, err := gitlabClient.diffs(t.Project.ID, fmt.Sprintf("%d", t.Target.IID))
		if err != nil {
			return nil, fmt.Errorf("failed to get diffs: %w", err)
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
		result[t.Target.IID] = float64(diffScore)
	}
	return result, nil
}

func mergeScore(scores ...map[int]float64) map[int]float64 {
	result := make(map[int]float64)
	for _, score := range scores {
		for k, v := range score {
			result[k] += v
		}
	}
	return result
}

func setScore(todos []ToDo, scoreMap map[int]float64) {
	for i, t := range todos {
		todos[i].Score = scoreMap[t.Target.IID]
	}
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

	b := flag.Bool("debug", false, "debug flag")
	flag.Parse()

	g := NewGitLabClient(host, token)

	todos, err := g.todos()
	if err != nil {
		log.Fatal(err)
	}

	todos = rmDraft(todos)
	todos = onlyOpenState(todos)

	urgentScoreMap := urgentScore(todos)
	userNamePosScoreMap := userNamePosScore(todos, user)
	createdAtScoreMap := createdAtScore(todos)
	diffScoreMap, err := diffScore(g, todos, []string{".go", ".tf", ".py"}) // TODO: configurable
	if err != nil {
		log.Fatal(err)
	}
	commitScoreMap, err := commitScore(g, user, todos)
	if err != nil {
		log.Fatal(err)
	}

	if *b {
		fmt.Println("urgentScore", urgentScoreMap)
		fmt.Println("userNamePosScore", userNamePosScoreMap)
		fmt.Println("createdAtScore", createdAtScoreMap)
		fmt.Println("diffScore", diffScoreMap)
		fmt.Println("commitScore", commitScoreMap)
	}

	scoreMap := mergeScore(urgentScoreMap, userNamePosScoreMap, createdAtScoreMap, diffScoreMap, commitScoreMap)
	setScore(todos, scoreMap)
	sortByScore(todos)

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', tabwriter.TabIndent)
	w.Write([]byte("url\tscore\n"))
	for _, todo := range todos {
		fmt.Fprintf(w, "%s\t%v\n", todo.TargetURL, todo.Score)
	}
	w.Flush()
}
