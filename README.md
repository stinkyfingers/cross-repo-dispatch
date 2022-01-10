# Cross Repo Dispatch
Story: I want to push code to one repository (a source repo) and have its Github Action trigger a test in another repository (a target repo) and have CI obtain the results of that test.

Triggering a Github Action from outside its repo is simple enough with a [repository or workflow dispatch](https://docs.github.com/en/actions/learn-github-actions/events-that-trigger-workflows#manual-events). Obtaining the result of that dispatch is a little trickier. Triggering a dispatch returns a 204, no content, with no indication as to the workflow ID. For this to work, you need to include a step in your test respository's workflow named with the "sha". This sha should be unique. Inside your "target repo," you will need something like:

```
jobs:
  identify-run:
    runs-on: ubuntu-latest
    steps:
      - name: ${{ github.event.client_payload.sha }}
        run: echo "Running this test ${{ github.event.client_payload.sha }}"
```
From your source repo, you can run the cross-repo-dispatch with a Github Action step like:

```
- name: run test-in-another-repo
	id: get_status
	uses: stinkyfingers/cross-repo-dispatch@v0.0
	with:
	  owner: 'my-repo-owner'
	  repo: 'my-target-repo'
	  ref: 'master'
	  pat: ${{ secrets.SHIPA_GITHUB_TOKEN }}
	  user: ${{ secrets.SHIPA_GITHUB_USERNAME }}
	  sha: ${{ github.sha }}
	  event_type: ${{ github.event.head_commit.message }}
	  client_payload: '{"misc_key":"misc_value"}'
	  workflow_status_timeout: 1200
```

### Options
```
  owner:
    description: 'target repo owner'
    required: true
  repo:
    description: 'target repo'
    required: true
  ref:
    description: 'ref of target repo to use'
    required: true
  pat:
    description: 'target repo personal access token'
    required: true
  user:
    description: 'target repo user'
    required: true
  sha:
    description: 'unique step name "sha" to search for in jobs'
    required: true
  client_payload:
    description: 'data to send to the workflow'
    required: false
  workflow_status_retry_interval:
    description: 'interval to retry getting workflow status'
    required: false
    default: 10
  workflow_status_timeout:
    description: 'timeout to fail getting workflow status'
    required: false
    default: 600
  max_runs:
    description: 'max number of runs (starting at most recent) back to check for matching step "name"'
    required: false
    default: 10
  event_type:
    description: 'event type label'
    required: true
```

### Output
```
  status:
    description: 'workflow status'
  jobs_url:
    description: 'url for workflow jobs'
```