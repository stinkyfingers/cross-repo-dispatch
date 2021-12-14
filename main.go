package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sethvargo/go-githubactions"
)

// WorkflowRunsResponse defines relevant fields from a github api /runs response
type WorkflowRunsResponse struct {
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

type WorkflowRun struct {
	ID         int    `json:"id"`
	JobsURL    string `json:"jobs_url"`
	Status     string `json:"status"`    // “queued”, “in_progress”, or “completed”
	Conclusion string `json:"completed"` // “success”, “failure”, “neutral”, “cancelled”, “skipped”, “timed_out”, or “action_required”
}

// RunJobsResponse defines relevant fields from a github api /jobs response
type RunJobsResponse struct {
	Jobs []struct {
		Steps []struct {
			Name string `json:"name"`
		} `json:"steps"`
	} `json:"jobs"`
}

// type WorkflowDispatchRequest struct {
// 	Ref    string `json:"ref"`
// 	Inputs struct {
// 		SHA string `json:"sha"`
// 	} `json:"inputs"`
// }

var (
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrTimeout          = errors.New("timeout waiting for workflow to complete")
)

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
	name := action.GetInput("name")
	if name == "" {
		action.Fatalf("missing input 'name'")
	}
	pat := action.GetInput("pat")
	if pat == "" {
		action.Fatalf("missing input 'pat'")
	}
	action.AddMask(pat)

	runID, err := findWorkflowRunWithStepName(owner, repo, user, pat, name)
	if err != nil {
		action.Fatalf("error getting runs: %s", err.Error())
		return
	}
	conclusion, err := getWorkflowRunConclusion(owner, repo, user, pat, runID)
	if err != nil {
		action.Fatalf("error getting runs: %s", err.Error())
		return
	}
	fmt.Println("STATUS", conclusion)
	action.SetOutput("status", conclusion)
}

// findWorkflowRunWithStepName gets jobs for the last <maxRuns> runs and returns the workflow ID
func findWorkflowRunWithStepName(owner, repo, user, pat, name string) (int, error) {
	maxRuns := 10 // TODO configure
	wrr, err := getRuns(owner, repo, user, pat)
	if err != nil {
		return 0, err
	}
	for i, run := range wrr.WorkflowRuns {
		if i == maxRuns {
			break
		}
		fmt.Println("RUNID", run.ID)
		rjr, err := getJob(owner, repo, user, pat, run.ID)
		if err != nil {
			return 0, err
		}
		for _, job := range rjr.Jobs {
			fmt.Println("JOB", job.Steps)
			for _, step := range job.Steps {
				if step.Name == name {
					return run.ID, nil
				}
			}
		}
	}
	return 0, ErrWorkflowNotFound
}

// getWorkflowRunConclusion retries getting a workflow by ID until the Status is "completed". It returns the Conclusion
func getWorkflowRunConclusion(owner, repo, user, pat string, runID int) (string, error) {
	waitSeconds := 10     // TODO configure
	timeoutSeconds := 600 // TODO configure
	done := make(chan struct{})

	time.AfterFunc(time.Second*time.Duration(timeoutSeconds), func() {
		done <- struct{}{}
	})

	ticker := time.NewTicker(time.Second * time.Duration(waitSeconds))
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return "", ErrTimeout
		case <-ticker.C:
			run, err := getRun(owner, repo, user, pat, runID)
			if err != nil {
				return "", err
			}
			if run.Status == "completed" {
				return run.Conclusion, nil
			}
		}
	}
}

/* API Calls */

// func repositoryDispatch(owner, repo, user, pat string) error {
// 	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/dispatches", owner, repo), nil)
// 	if err != nil {
// 		return err
// 	}
// 	req.URL.Query().Add("ref", "main") // TODO configure
// 	req.SetBasicAuth(user, pat)
// 	req.Header.Set("accept", "application/vnd.github.v3+json")
// 	req.Header.Set("Content-Type", "application/json")
// 	cli := &http.Client{}
// 	resp, err := cli.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	if resp.StatusCode != http.StatusNoContent {
// 		return fmt.Errorf("expected status 204, got %d", resp.StatusCode)
// 	}
// 	return nil
// }

func getRun(owner, repo, user, pat string, runID int) (*WorkflowRun, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%d", owner, repo, runID), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pat)
	req.Header.Set("accept", "application/vnd.github.v3+json")
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	var run WorkflowRun
	err = json.NewDecoder(resp.Body).Decode(&run)
	if err != nil {
		return nil, err
	}
	return &run, err
}

func getRuns(owner, repo, user, pat string) (*WorkflowRunsResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs", owner, repo), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pat)
	req.Header.Set("accept", "application/vnd.github.v3+json")
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

func getJob(owner, repo, user, pat string, runID int) (*RunJobsResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%d/jobs", owner, repo, runID), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pat)
	req.Header.Set("accept", "application/vnd.github.v3+json")
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	var rjr RunJobsResponse
	err = json.NewDecoder(resp.Body).Decode(&rjr)
	if err != nil {
		return nil, err
	}
	return &rjr, err
}
