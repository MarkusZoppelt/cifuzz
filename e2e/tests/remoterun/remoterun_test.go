package remoterrun_test

import (
	"net/http"
	"testing"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
)

var remoteRunWithoutConnectionTests = &[]e2e.TestCase{
	{
		Description:   "remote-run command without connection to CI Sense fails if token is found",
		Command:       "remote-run",
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

func TestRemoteRunWithoutConnection(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusRequestTimeout)
	}

	e2e.RunTests(t, *remoteRunWithoutConnectionTests, server)
}
