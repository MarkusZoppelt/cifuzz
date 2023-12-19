package api

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type ContainerRun struct {
	Image     string      `json:"image"`
	FuzzTests []*FuzzTest `json:"fuzz_tests,omitempty"`

	// ProjectNid is the new project ID format used in responses from CI Sense.
	// Future responses from the /v3 API will use these nano IDs and are usually
	// prefixed with "prj", "job", "org", etc.
	ProjectNid string `json:"project_nid,omitempty"`

	// ProjectExternalID is the conventional project id used in the URL of CI Sense
	ProjectExternalID string `json:"project_external_id,omitempty"`
}

type ContainerRunResponse struct {
	Run   *Run    `json:"run,omitempty"`
	Links []*Link `json:"links,omitempty"`
}

type FuzzTest struct {
	Name string `json:"name"`
	Jobs []*Job `json:"jobs,omitempty"`
}

type Link struct {
	Href   string `json:"href,omitempty"`
	Rel    string `json:"rel,omitempty"`
	Method string `json:"method,omitempty"`
}

type Run struct {
	Nid       string      `json:"nid,omitempty"`
	Image     string      `json:"image,omitempty"`
	FuzzTests []*FuzzTest `json:"fuzz_tests,omitempty"`
}
type Job struct {
	Nid    string `json:"nid,omitempty"`
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
	Config string `json:"config,omitempty"`
}

// PostContainerRemoteRun posts a new container run to the CI Sense API at /v3/runs.
// project does not need to have a projects/ prefix and needs to be url encoded.
func (client *APIClient) PostContainerRemoteRun(image string, project string, fuzzTests []string, token string) (*ContainerRunResponse, error) {
	tests := []*FuzzTest{}
	for _, fuzzTest := range fuzzTests {
		tests = append(tests, &FuzzTest{Name: fuzzTest})
	}

	nid, err := ProjectNameToNID(project)
	if err != nil {
		return nil, err
	}

	containerRun := &ContainerRun{
		Image:      image,
		FuzzTests:  tests,
		ProjectNid: nid,
	}

	body, err := json.Marshal(containerRun)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return APIRequest[ContainerRunResponse](&RequestConfig{
		Client:       client,
		Method:       "POST",
		Body:         body,
		Token:        token,
		PathSegments: []string{"v3", "runs"},
	})
}

type ContainerRemoteRunStatus struct {
	Run ContainerRemoteRun `json:"run"`
}

type ContainerRemoteRun struct {
	Nid    string `json:"nid"`
	Status string `json:"status"`
}

// GetContainerRemoteRunStatus gets the status of a container run from the CI
// Sense API at /v3/runs/{run_nid}/status.
func (client *APIClient) GetContainerRemoteRunStatus(runNID string, token string) (*ContainerRemoteRunStatus, error) {
	return APIRequest[ContainerRemoteRunStatus](&RequestConfig{
		Client:       client,
		Method:       "GET",
		Token:        token,
		PathSegments: []string{"v3", "runs", runNID, "status"},
	})
}
