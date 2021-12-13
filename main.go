package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sethvargo/go-githubactions"
)

// WorkflowRunsResponse defines relevant fields from a github api /runs response
type WorkflowRunsResponse struct {
	WorkflowRuns []struct {
		ID      int    `json:"id"`
		JobsURL string `json:"jobs_url"`
	} `json:"workflow_runs"`
}

func main() {
	action := githubactions.New()
	user := action.GetInput("user")
	if user == "" {
		action.Fatalf("missing input 'user'")
	}
	owner := action.GetInput("owner")
	if owner == "" {
		action.Fatalf("missing input 'owner'")
	}
	repo := action.GetInput("repo")
	if repo == "" {
		action.Fatalf("missing input 'repo'")
	}
	sha := action.GetInput("sha")
	if sha == "" {
		action.Fatalf("missing input 'sha'")
	}
	pat := action.GetInput("pat")
	if pat == "" {
		action.Fatalf("missing input 'pat'")
	}
	action.AddMask(pat)

	wrr, err := getRuns(owner, repo, user, pat)
	if err != nil {
		action.Fatalf("error getting runs: %s", err.Error())
		return
	}
	j, _ := json.Marshal(wrr)
	action.SetOutput("status", string(j))
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
