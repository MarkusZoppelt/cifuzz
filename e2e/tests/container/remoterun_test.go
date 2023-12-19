package container_test

import (
	"testing"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
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
		Args:          []string{" --project test-project --registry localhost:5000/test/cifuzz com.example.FuzzTestCase::myFuzzTest -v"},
		SampleFolder:  []string{"../../../examples/maven", "../../../examples/gradle"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().
				ErrorContains("Created fuzz container image with ID ").
				ErrorContains("Start uploading image ").
				ErrorContains("to localhost:5000/test/cifuzz").
				ErrorContains("The push refers to repository [localhost:5000/test/cifuzz]")
		},
	},
	{
		Description:   "container remote-run command in a maven/gradle example directory with monitor mode runs successfully",
		CIUser:        e2e.LoggedInCIUser,
		Command:       "container remote-run",
		Args:          []string{" --project test-project --registry localhost:5000/test/cifuzz com.example.FuzzTestCase::myFuzzTest --monitor --monitor-duration 5m --monitor-interval 5s -v"},
		SampleFolder:  []string{"../../../examples/maven"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().
				ErrorContains("Created fuzz container image with ID ").
				ErrorContains("Start uploading image ").
				ErrorContains("to localhost:5000/test/cifuzz").
				ErrorContains("The push refers to repository [localhost:5000/test/cifuzz]").
				ErrorContains("Max monitor duration is 300 seconds.").
				ErrorContains("Finding found: test_finding, NID: fdn-testtesttesttest")
		},
	},
}

func TestContainerRemoteRun(t *testing.T) {
	mockServer := mockserver.New(t)
	mockServer.Handlers["/v1/projects"] = mockserver.ReturnResponse(t, mockserver.ProjectsJSON)
	mockServer.Handlers["/v3/runs"] = mockserver.ReturnResponse(t, mockserver.ContainerRemoteRunResponse)
	mockServer.Handlers["/v3/runs/run-testtesttesttest/status"] = mockserver.ReturnResponse(t, mockserver.ContainerRemoteRunStatusResponse)
	mockServer.Handlers["/v3/runs/run-testtesttesttest/findings"] = mockserver.ReturnResponse(t, mockserver.ContainerRemoteRunFindingsResponse)
	e2e.RunTests(t, *containerRemoteRunTests, mockServer)
}
