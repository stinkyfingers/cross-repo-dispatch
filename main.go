package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sethvargo/go-githubactions"
)

func main() {
	user := githubactions.GetInput("user")
	if user == "" {
		githubactions.Fatalf("missing input 'user'")
	}
	owner := githubactions.GetInput("owner")
	if owner == "" {
		githubactions.Fatalf("missing input 'owner'")
	}
	repo := githubactions.GetInput("repo")
	if repo == "" {
		githubactions.Fatalf("missing input 'repo'")
	}
	sha := githubactions.GetInput("sha")
	if sha == "" {
		githubactions.Fatalf("missing input 'sha'")
	}
	pat := githubactions.GetInput("pat")
	if pat == "" {
		githubactions.Fatalf("missing input 'pat'")
	}
	githubactions.AddMask(pat)
	action := githubactions.New()

	wrr, err := getRuns(owner, repo, user, pat)
	if err != nil {
		action.Fatalf("error getting runs: %s", err.Error())
	}
	j, _ := json.Marshal(wrr)
	action.SetOutput("status", string(j))
}

type WorkflowRunsResponse struct {
	WorkflowRuns []struct {
		ID      int    `json:"id"`
		JobsURL string `json:"jobs_url"`
	}
}

func getRuns(owner, repo, user, pat string) (*WorkflowRunsResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs", owner, repo), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pat)
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	var wrr WorkflowRunsResponse
	err = json.NewDecoder(resp.Body).Decode(&wrr)
	if err != nil {
		return nil, err
	}
	return &wrr, err
}
