
/*
Copyright © 2020 Flanksource
This file is part of Flanksource github-app
*/
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flanksource/build-tools/pkg/junit"
	"github.com/google/go-github/v32/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

type CheckRunHandler struct {
	githubapp.ClientCreator

	preamble string
}

func (h *CheckRunHandler) Handles() []string {
	return []string{"check_run"}
}

func (h *CheckRunHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.CheckRunEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse issue comment event payload")
	}

	fmt.Printf("Event %v - %v\n", *event.CheckRun.Name, event.GetAction() )

	if event.GetAction() != "completed" {
		return nil //we only want to process the results at completion and ignore anything else
	}

	installationID := *event.Installation.ID
	client, err := h.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrapf(err, "failed to get github client from installationID %s given in event", installationID)
	}

	repo := event.GetRepo()
	repoOwner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()


	cr, _, err := client.Checks.GetCheckRun(ctx, repoOwner, repoName, *event.CheckRun.ID)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", *cr.Name)
	if !strings.HasPrefix(*cr.Name, "test ") {
		return nil
	}
	content := strings.ReplaceAll(*cr.Name, "test (","")
	content = strings.ReplaceAll(content, ")","")
	data := strings.Split(content,", ")
	k8s := data[0]
	suite := data[1]


	getStrRef := func (myString string) *string {
		return &myString
	}
	getIntRef := func (myInt int) *int {
		return &myInt
	}
	title := "Test Title"
	summary := "Test Summary"
	_, _, err = client.Checks.UpdateCheckRun(ctx, repoOwner, repoName, *cr.ID, github.UpdateCheckRunOptions{
		Output: &github.CheckRunOutput{
			Title:       &title,
			Summary:     &summary,
			Annotations: []*github.CheckRunAnnotation{{
				Path:            getStrRef("test/test.sh"),
				StartLine:       getIntRef(1),
				EndLine:         getIntRef(1),
				AnnotationLevel: getStrRef("notice"),
				Message:         getStrRef("Test Message"),
				Title:           getStrRef("Test Title"),
			}},
		},
	})
	if err != nil {
		return err
	}

	// ☹️ no easy direct way to go from the check run to the workflow run
	// so we get this first page of workflow runs for this owner/repo
	// and look for a matching head SHA commit on the branch for this event
	wfrList, _, err := client.Actions.ListRepositoryWorkflowRuns(ctx, repoOwner, repoName, &github.ListWorkflowRunsOptions{
		Branch:      *event.CheckRun.CheckSuite.HeadBranch,
		ListOptions: github.ListOptions{},
	})
	if err != nil {
		return err
	}


	var junitTestText string = ""
	for _, wfr := range wfrList.WorkflowRuns {
		if *wfr.HeadSHA != *event.CheckRun.HeadSHA {
			continue // ignore the run if it isn't for our commit, got to next workflowrun
		}
		if *wfr.CheckSuiteURL != *event.CheckRun.CheckSuite.URL {
			continue
		}

		fmt.Printf("%v\n",*wfr.ID)
		url, _ := url.Parse(*wfr.WorkflowURL)
		wfId , err := strconv.ParseInt(path.Base(url.Path), 10, 64)
		fmt.Printf("wf id = %v\n",wfId)

		wf, _, err := client.Actions.GetWorkflowByID(ctx, repoOwner, repoName, wfId)

		fmt.Printf("wf name %v\n", *wf.Name)


		//client.Actions.GetWorkflowByID(strings.wfr.WorkflowURL)

		// cool now for this workflow run we get the artifacts
		artifactList, _, err := client.Actions.ListWorkflowRunArtifacts(ctx, repoOwner, repoName, *wfr.ID,&github.ListOptions{})
		if err != nil {
			return err
		}
		for _, artifact := range artifactList.Artifacts {
			fmt.Printf("artifact name %v\n", *artifact.Name)
			if !strings.HasPrefix(*artifact.Name, "test-results") {
				continue // we only care about 'test-results*', skip this artifact
			}
			fmt.Printf("does '%s' == '%s':", *artifact.Name, "test-results-"+k8s+"-"+suite )
			fmt.Printf("... '%v'\n", *artifact.Name == "test-results-"+k8s+"-"+suite )
			if *artifact.Name == "test-results-"+k8s+"-"+suite {
					url, _, err := client.Actions.DownloadArtifact(ctx, repoOwner, repoName,*artifact.ID,true)
					if err != nil {
						//logger.Error().Err(err).Msg("failed to get artifact download url")
						continue //ignore error, try next artifact
					}
					tmpfile, err := ioutil.TempFile("/tmp", "downloadedzip")
					defer os.Remove(tmpfile.Name())
					if err != nil {
						continue //ignore error, try next artifact
					}
					err = downloadFile(url.String(),tmpfile)
					junitTestText, err = getUnzippedFileContents(tmpfile.Name(),"results.xml")
			}
		}


	}
	results :=  make([]string, 0, 20)
	results = append(results,junitTestText)


	r, err := junit.ParseJunitResultStrings(results...)
	if err != nil {
		return err
	}

	title = "Test Results"
	summary = r.String()

	client, err = h.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrapf(err, "failed to get github client from installationID %s given in event", installationID)
	}

	_, _, err = client.Checks.UpdateCheckRun(ctx, repoOwner, repoName, *cr.ID, github.UpdateCheckRunOptions{
		Output: &github.CheckRunOutput{
			Title:       &title,
			Summary:     &summary,
			Annotations: r.GetGithubAnnotations(),
		},
	})
	if err != nil {
		return err
	}



	return nil
}


