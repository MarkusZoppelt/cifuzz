package container_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
	"code-intelligence.com/cifuzz/internal/api"
)

const (
	prjNid = "prj-testtesttesttest"
	runNid = "run-testtesttesttest"
)

var containerRemoteRunWithoutConnectionTests = &[]e2e.TestCase{
	{
		Description:   "container remote-run command without connection to CI Sense fails if token is found",
		Command:       "container remote-run",
		CIUser:        e2e.LoggedInCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest"},
		SampleFolder:  []string{"../../../examples/maven"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Failed().ErrorContains("No connection to CI Sense")
		},
	},
}

var containerRemoteRunInvalidTokenTests = &[]e2e.TestCase{
	{
		Description:   "container remote-run command without connection to CI Sense fails if token is found",
		Command:       "container remote-run",
		CIUser:        e2e.InvalidTokenCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest"},
		SampleFolder:  []string{"../../../examples/maven"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Failed().ErrorContains("Please log in with a valid API access token")
		},
	},
}

var containerRemoteRunTests = &[]e2e.TestCase{
	{
		Description: "container remote-run command is available in --help output",
		Command:     "container remote-run",
		Args:        []string{"--help"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().OutputContains("container")
		},
	},
	{
		Description:   "container remote-run command in a maven/gradle example directory is available and pushes it to a registry",
		CIUser:        e2e.LoggedInCIUser,
		Command:       "container remote-run",
		Args:          []string{fmt.Sprintf(" --project %s com.example.FuzzTestCase::myFuzzTest -v", prjNid)},
		SampleFolder:  []string{"maven-default", "gradle-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().
				ErrorContains("Created fuzz container image with ID ").
				ErrorContains("Start uploading image ")
		},
	},
	{
		Description:   "container remote-run command in a maven/gradle example directory with --registry flag pushes it to a custom registry",
		CIUser:        e2e.LoggedInCIUser,
		Command:       "container remote-run",
		Args:          []string{fmt.Sprintf(" --project %s com.example.FuzzTestCase::myFuzzTest --registry localhost:5000/test/cifuzz -v", prjNid)},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().
				ErrorContains("Getting registry credentials from local docker config for registry localhost:5000/test/cifuzz").
				ErrorContains("Start uploading image ").
				ErrorContains("Tag used for upload: localhost:5000/cifuzz/")
		},
	},
	{
		Description:   "container remote-run command in a maven/gradle example directory with monitor mode runs successfully",
		CIUser:        e2e.LoggedInCIUser,
		Command:       "container remote-run",
		Args:          []string{fmt.Sprintf(" --project %s com.example.FuzzTestCase::myFuzzTest --monitor --monitor-duration 5m --monitor-interval 5s -v", prjNid)},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().
				ErrorContains("Created fuzz container image with ID ").
				ErrorContains("Max monitor duration is 300 seconds.").
				ErrorContains("Finding found: test_finding, NID: fdn-testtesttesttest").
				NoOutput()
		},
	},
	{
		Description:   "container remote-run command in a maven/gradle example directory with monitor mode and check for duration limit.",
		CIUser:        e2e.LoggedInCIUser,
		Command:       "container remote-run",
		Args:          []string{fmt.Sprintf(" --project %s com.example.FuzzTestCase::myFuzzTest --monitor --monitor-duration 10s --monitor-interval 5s -v", prjNid)},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().
				ErrorContains("Created fuzz container image with ID ").
				ErrorContains("Start uploading image ").
				ErrorContains("Max monitor duration is 10 seconds.").
				ErrorNotContains("Finding found: test_finding, NID: fdn-testtesttesttest")
		},
	},
	{
		Description:   "container remote-run command in a maven/gradle example directory with monitor mode and JSON output produces parsable JSON",
		CIUser:        e2e.LoggedInCIUser,
		Command:       "container remote-run",
		Args:          []string{fmt.Sprintf(" --project %s com.example.FuzzTestCase::myFuzzTest --monitor --monitor-duration 5m --monitor-interval 5s --json", prjNid)},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success()
			var response api.ContainerRunResponse
			err := json.Unmarshal([]byte(output.Stdout), &response)
			require.NoError(t, err)
			require.NotEmpty(t, response.Links)
			require.NotEmpty(t, response.Run.Nid)
			require.NotEmpty(t, response.Run.FuzzTests)
		},
	},
}

func TestContainerRemoteRunWithoutConnection(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusRequestTimeout)
	}

	e2e.RunTests(t, *containerRemoteRunWithoutConnectionTests, server)
}

func TestContainerRemoteRunWithInvalidToken(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = mockserver.ReturnResponseIfValidToken(t, "")

	e2e.RunTests(t, *containerRemoteRunInvalidTokenTests, server)
}

func TestContainerRemoteRun(t *testing.T) {
	mockServer := mockserver.New(t)
	mockServer.Handlers["/v1/projects"] = mockserver.ReturnResponse(t, mockserver.ProjectsJSON)
	mockServer.Handlers["/v2/docker_registry/authentication"] = mockserver.ReturnResponse(t, mockserver.ContainerRegstryCredentialsResponse)
	mockServer.Handlers[fmt.Sprintf("/v3/projects/%s/runs", prjNid)] = mockserver.ReturnResponse(t, mockserver.ContainerRemoteRunResponse)
	mockServer.Handlers[fmt.Sprintf("/v3/runs/%s/status", runNid)] = mockserver.ReturnResponse(t, mockserver.ContainerRemoteRunStatusResponse)

	requestCount := 0
	mockServer.Handlers[fmt.Sprintf("/v3/runs/%s/findings", runNid)] = func(w http.ResponseWriter, req *http.Request) {
		if requestCount >= 5 {
			_, err := io.WriteString(w, mockserver.ContainerRemoteRunFindingsResponse)
			require.NoError(t, err)
			requestCount = 0
		} else {
			_, err := io.WriteString(w, mockserver.ContainerRemoteRunNoFindingsResponse)
			require.NoError(t, err)
			requestCount++
		}
	}

	e2e.RunTests(t, *containerRemoteRunTests, mockServer)
}
