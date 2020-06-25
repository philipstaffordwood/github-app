package handler

import (
	"context"
	"fmt"
	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

func GetWorkflowRun(ctx context.Context, client github.Client, repoOwner string, repoName string, branch string, headSHA string )  (*github.WorkflowRun, error) {
	// ☹️ no easy direct way to go from the check run to the workflow run
	// so we get this first page of workflow runs for this owner/repo
	// and look for a matching head SHA commit on the branch for this event
	wfrList, _, err := client.Actions.ListRepositoryWorkflowRuns(ctx, repoOwner, repoName, &github.ListWorkflowRunsOptions{
		Branch:      branch,
		ListOptions: github.ListOptions{},
	})
	if err != nil {
		return nil, err
	}

	for _, wfr := range wfrList.WorkflowRuns {
		if *wfr.HeadSHA != headSHA {
			continue // ignore the run if it isn't for our commit, got to next workflowrun
		}
		return wfr, nil
	}
	return nil, fmt.Errorf("no matching workflow run found")
}

// getPatClient returns a github client that uses the given
// Personal Access Token to authenticate
// NOTE: this is a workaround for issues experienced with using
//       githubapp.ClientCreator.NewAppClient()
func getPatClient(ctx context.Context, pat string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: pat},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

