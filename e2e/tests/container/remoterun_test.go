package container_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
	"code-intelligence.com/cifuzz/internal/api"
)

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
		Args:          []string{" --project test-project com.example.FuzzTestCase::myFuzzTest -v"},
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
		Description:   "container remote-run command in a maven/gradle example directory with monitor mode runs successfully",
		CIUser:        e2e.LoggedInCIUser,
		Command:       "container remote-run",
		Args:          []string{" --project test-project com.example.FuzzTestCase::myFuzzTest --monitor --monitor-duration 5m --monitor-interval 5s -v"},
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
		Args:          []string{" --project test-project com.example.FuzzTestCase::myFuzzTest --monitor --monitor-duration 10s --monitor-interval 5s -v"},
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
		Args:          []string{" --project test-project com.example.FuzzTestCase::myFuzzTest --monitor --monitor-duration 5m --monitor-interval 5s --json"},
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

func TestContainerRemoteRun(t *testing.T) {
	mockServer := mockserver.New(t)
	mockServer.Handlers["/v1/projects"] = mockserver.ReturnResponse(t, mockserver.ProjectsJSON)
	mockServer.Handlers["/v2/docker_registry/authentication"] = mockserver.ReturnResponse(t, mockserver.ContainerRegstryCredentialsResponse)
	mockServer.Handlers["/v3/runs"] = mockserver.ReturnResponse(t, mockserver.ContainerRemoteRunResponse)
	mockServer.Handlers["/v3/runs/run-testtesttesttest/status"] = mockserver.ReturnResponse(t, mockserver.ContainerRemoteRunStatusResponse)

	requestCount := 0
	mockServer.Handlers["/v3/runs/run-testtesttesttest/findings"] = func(w http.ResponseWriter, req *http.Request) {
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
