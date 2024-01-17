package run_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
)

var runWithoutConnectionTests = &[]e2e.TestCase{
	{
		Description:   "run command without connection to CI Sense runs with a warning if token is found",
		Command:       "run",
		CIUser:        e2e.LoggedInCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest"},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().ErrorContains("No connection to CI Sense. Findings are not uploaded")
		},
	},
}

var runWithInvalidTokenTests = &[]e2e.TestCase{
	{
		Description:   "run command fails with invalid token",
		Command:       "run",
		CIUser:        e2e.InvalidTokenCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest"},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Failed().ErrorContains("Invalid token")
		},
	},
}

var runModeTests = &[]e2e.TestCase{
	{
		Description:   "run in local mode does not communicate with CI Sense",
		Command:       "run",
		CIUser:        e2e.AnonymousCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest -v"},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.NotContains(t, output.Stderr, "Sending HTTP request:")
			assert.EqualValues(t, 0, output.ExitCode)
		},
	},
	{
		Description:   "run in remote mode communicates with CI Sense",
		Command:       "run",
		CIUser:        e2e.LoggedInCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest --project my_fuzz_test-bac40407 -v"},
		SampleFolder:  []string{"maven-default"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.Contains(t, output.Stderr, "Sending HTTP request:")
			assert.EqualValues(t, 0, output.ExitCode)
		},
	},
}

func TestRunWithoutConnection(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusRequestTimeout)
	}

	e2e.RunTests(t, *runWithoutConnectionTests, server)
}

func TestRunWithInvalidToken(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = mockserver.ReturnResponseIfValidToken(t, "")

	e2e.RunTests(t, *runWithInvalidTokenTests, server)
}

func TestRunLocalRemoteMode(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects/my_fuzz_test-bac40407/campaign_runs"] = mockserver.ReturnResponseIfValidToken(t, "{}")
	server.Handlers["/v1/projects/my_fuzz_test-bac40407/findings"] = mockserver.ReturnResponseIfValidToken(t, "{}")
	server.Handlers["/v1/projects"] = mockserver.ReturnResponseIfValidToken(t, mockserver.ProjectsJSON)

	e2e.RunTests(t, *runModeTests, server)
}
