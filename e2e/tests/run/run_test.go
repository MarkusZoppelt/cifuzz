package run_test

import (
	"net/http"
	"testing"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
)

var runWithoutConnectionTests = &[]e2e.TestCase{
	{
		Description:   "finding command without connection to CI Sense runs with a warning if token is found",
		Command:       "run",
		CIUser:        e2e.LoggedInCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest"},
		SampleFolder:  []string{"../../../examples/maven"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().ErrorContains("No connection to CI Sense. Findings are not uploaded")
		},
	},
}

var runWithInvalidTokenTests = &[]e2e.TestCase{
	{
		Description:   "finding command fails with invalid token",
		Command:       "run",
		CIUser:        e2e.InvalidTokenCIUser,
		Args:          []string{"com.example.FuzzTestCase::myFuzzTest"},
		SampleFolder:  []string{"../../../examples/maven"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Failed().ErrorContains("Invalid token")
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
