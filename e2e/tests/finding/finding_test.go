package finding_test

import (
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
)

var findingWithoutConnectionTests = &[]e2e.TestCase{
	{
		Description:  "finding command without connection to CI Sense runs with a warning if token is found",
		Command:      "finding",
		CIUser:       e2e.LoggedInCIUser,
		SampleFolder: []string{"project-with-empty-cifuzz-yaml"},
		SkipOnOS:     "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().ErrorContains("No connection to CI Sense. Only local findings are shown.")
		},
	},
}

var findingWithInvalidTokenTests = &[]e2e.TestCase{
	{
		Description:  "finding command fails with invalid token",
		Command:      "finding",
		CIUser:       e2e.InvalidTokenCIUser,
		SampleFolder: []string{"project-with-empty-cifuzz-yaml"},
		SkipOnOS:     "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Failed().ErrorContains("Invalid token")
		},
	},
}

var findingTests = &[]e2e.TestCase{
	{
		Description:  "finding command in an empty folder prints error saying it is not a cifuzz project",
		Command:      "finding",
		SampleFolder: []string{"empty"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Failed().NoOutput().ErrorContains("not a cifuzz project")
		},
	},
	{
		Description:  "finding command in a project without findings",
		Command:      "finding",
		SampleFolder: []string{"project-with-empty-cifuzz-yaml"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().NoOutput().ErrorContains("This project doesn't have any findings yet")
		},
	},
	{
		Description:  "finding command ran in a project with findings prints findings table with severity score and fuzz test name",
		Command:      "finding",
		Args:         []string{"--interactive=false"},
		SampleFolder: []string{"project-with-findings-and-error-details"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.NotContains(t, output.Stdout, "n/a")
			assert.Contains(t, output.Stdout, "9.0")
			assert.Contains(t, output.Stdout, "heap buffer overflow")
			assert.Contains(t, output.Stdout, "src/explore_me.cpp:18:11")
		},
	},
	{
		Description:  "finding command with finding name argument ran in a project with findings print findings table",
		Command:      "finding",
		Args:         []string{"funky_angelfish --interactive=false"},
		SampleFolder: []string{"project-with-findings-and-error-details"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.Contains(t, output.Stdout, "my_fuzz_test")
			assert.Contains(t, output.Stdout, "heap buffer overflow")
			assert.Contains(t, output.Stdout, "src/explore_me.cpp:18:11")
			assert.Contains(t, output.Stderr, "cifuzz found more extensive information about this finding:")
			assert.Contains(t, output.Stderr, "| Severity Level       | Critical                                                                         |")
			assert.Contains(t, output.Stderr, "| Severity Score       | 9.0                                                                              |")
			assert.Contains(t, output.Stderr, "| ASan Example         | https://github.com/google/sanitizers/wiki/AddressSanitizerExampleHeapOutOfBounds |")
			assert.Contains(t, output.Stderr, "| ASan Example         | https://github.com/google/sanitizers/wiki/AddressSanitizerExampleHeapOutOfBounds |")
			assert.Contains(t, output.Stderr, "| CWE: Overflow writes | https://cwe.mitre.org/data/definitions/787.html                                  |")
			assert.Contains(t, output.Stderr, "| CWE: Overflow reads  | https://cwe.mitre.org/data/definitions/125.html                                  |")
		},
	},
}

func TestFindingWithoutConnection(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusRequestTimeout)
	}

	e2e.RunTests(t, *findingWithoutConnectionTests, server)
}

func TestFindingWithInvalidToken(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = mockserver.ReturnResponseIfValidToken(t, "")

	e2e.RunTests(t, *findingWithInvalidTokenTests, server)
}

func TestFindingList(t *testing.T) {
	// skipping test on Windows because the 'host.docker.internal' address does
	// not work on Windows.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = mockserver.ReturnResponseIfValidToken(t, mockserver.ProjectsJSON)

	e2e.RunTests(t, *findingTests, server)
}
