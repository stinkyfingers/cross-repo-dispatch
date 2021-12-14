package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	Status     string `json:"status"`     // “queued”, “in_progress”, or “completed”
	Conclusion string `json:"conclusion"` // “success”, “failure”, “neutral”, “cancelled”, “skipped”, “timed_out”, or “action_required”
}

// RunJobsResponse defines relevant fields from a github api /jobs response
type RunJobsResponse struct {
	Jobs []struct {
		Steps []struct {
			Name string `json:"name"`
		} `json:"steps"`
	} `json:"jobs"`
}

type RepositoryDispatchRequest struct {
	EventType     string        `json:"event_type"`
	ClientPayload ClientPayload `json:"client_payload"`
}

type ClientPayload struct {
	Sha string `json:"sha"`
}

var (
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrTimeout          = errors.New("timeout waiting for workflow to complete")
)

func main() {
	var err error
	action := githubactions.New()

	/* inputs */
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
	workflowStatusRetryInterval := 10 // default
	workflowStatusRetryIntervalString := action.GetInput("workflow_status_retry_interval")
	if workflowStatusRetryIntervalString != "" {
		workflowStatusRetryInterval, err = strconv.Atoi(workflowStatusRetryIntervalString)
		if err != nil {
			action.Fatalf("workflow_status_retry_interval must be int: %s", err.Error())
		}
	}
	workflowStatusTimeout := 600 // default
	workflowStatusTimeoutString := action.GetInput("workflow_status_timeout")
	if workflowStatusTimeoutString != "" {
		workflowStatusTimeout, err = strconv.Atoi(workflowStatusTimeoutString)
		if err != nil {
			action.Fatalf("workflow_status_timeout must be int: %s", err.Error())
		}
	}
	maxRuns := 10
	maxRunsString := action.GetInput("max_runs")
	if maxRunsString != "" {
		maxRuns, err = strconv.Atoi(maxRunsString)
		if err != nil {
			action.Fatalf("max_runs must be int: %s", err.Error())
		}
	}
	eventType := action.GetInput("event_type")
	if owner == "" {
		action.Fatalf("missing input 'event_type'")
	}
	clientPayload := action.GetInput("client_payload")
	if clientPayload == "" {
		action.Fatalf("missing input 'client_payload'")
	}
	testRepoRef := action.GetInput("test_repo_ref")
	if testRepoRef == "" {
		action.Fatalf("missing input 'test_repo_ref'")
	}

	/* end inputs */

	err = repositoryDispatch(owner, repo, user, pat, eventType, name, testRepoRef)
	if err != nil {
		action.Fatalf("error running repository dispatch: %s", err.Error())
		return
	}

	runID, err := findWorkflowRunWithStepName(owner, repo, user, pat, name, maxRuns)
	if err != nil {
		action.Fatalf("error getting runs: %s", err.Error())
		return
	}
	conclusion, err := getWorkflowRunConclusion(owner, repo, user, pat, runID, workflowStatusRetryInterval, workflowStatusTimeout)
	if err != nil {
		action.Fatalf("error getting runs: %s", err.Error())
		return
	}
	fmt.Println("STATUS: ", conclusion)
	action.SetOutput("status", conclusion)
}

// findWorkflowRunWithStepName gets jobs for the last <maxRuns> runs and returns the workflow ID
func findWorkflowRunWithStepName(owner, repo, user, pat, name string, maxRuns int) (int, error) {
	wrr, err := getRuns(owner, repo, user, pat)
	if err != nil {
		return 0, err
	}
	for i, run := range wrr.WorkflowRuns {
		if i == maxRuns {
			break
		}
		rjr, err := getJob(owner, repo, user, pat, run.ID)
		if err != nil {
			return 0, err
		}
		for _, job := range rjr.Jobs {
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
func getWorkflowRunConclusion(owner, repo, user, pat string, runID, workflowStatusRetryInterval, workflowStatusTimeout int) (string, error) {
	done := make(chan struct{})

	time.AfterFunc(time.Second*time.Duration(workflowStatusTimeout), func() {
		done <- struct{}{}
	})

	ticker := time.NewTicker(time.Second * time.Duration(workflowStatusRetryInterval))
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

func repositoryDispatch(owner, repo, user, pat, eventType, name, testRepoRef string) error {
	rdp := &RepositoryDispatchRequest{
		EventType: eventType,
		ClientPayload: ClientPayload{
			Sha: name,
		},
	}
	j, err := json.Marshal(rdp)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(j)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/repos/%s/%s/dispatches", owner, repo), buf)
	if err != nil {
		return err
	}
	req.URL.Query().Add("ref", testRepoRef)
	req.SetBasicAuth(user, pat)
	req.Header.Set("accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("expected status 204, got %d", resp.StatusCode)
	}
	return nil
}

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
