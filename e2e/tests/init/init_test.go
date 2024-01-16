package init_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"code-intelligence.com/cifuzz/e2e"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
)

var initTests = &[]e2e.TestCase{
	{
		Description:  "init command in empty CMake project succeeds and creates a config file",
		Command:      "init",
		SampleFolder: []string{"cmake"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.Contains(t, output.Stdall, "Configuration saved in cifuzz.yaml")
		},
	},
	{
		Description:  "init command with a 'maven' argument should create a config file for java",
		Command:      "init",
		Args:         []string{"maven"},
		SampleFolder: []string{"node-typescript", "nodejs", "cmake", "empty"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.Contains(t, output.Stdall, "Configuration saved in cifuzz.yaml")
			assert.Contains(t, output.Stdall, "<artifactId>cifuzz-maven-extension</artifactId>")
			assert.NotContains(t, output.Stdall, "Failed to create config")
			output.FileExists("cifuzz.yaml")
		},
	},
	{
		Description: "init command in empty project with arg 'maven' and --interactive=false runs in local-only mode",
		Command:     "init",
		CIUser:      e2e.AnonymousCIUser,
		// note: we are always --interactive=false in CI
		Args:         []string{"maven --interactive=false"},
		SampleFolder: []string{"empty"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.NotContains(t, output.Stdall, "Do you want to initialize cifuzz for this project in remote (recommended) or local-only mode?")
			assert.Contains(t, output.Stdall, "Running in local-only mode.")
			assert.Contains(t, output.Stdall, "Configuration saved in cifuzz.yaml")
			assert.NotContains(t, output.Stdall, "Failed to create config")
			output.FileContains("cifuzz.yaml", []string{"#server:", "#project:"})
		},
	},
	{
		Description:  "init command in empty project with arg 'maven' and only '--project' throws an error",
		Command:      "init",
		CIUser:       e2e.AnonymousCIUser,
		Args:         []string{"maven --project=prj-testtesttesttest"},
		SampleFolder: []string{"empty"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 1, output.ExitCode)
			assert.NotContains(t, output.Stdall, "Do you want to initialize cifuzz for this project in remote (recommended) or local-only mode?")
			assert.NotContains(t, output.Stdall, "Running in remote mode.")
			assert.NotContains(t, output.Stdall, "Configuration saved in cifuzz.yaml")
		},
	},
}

var remoteModeTests = &[]e2e.TestCase{
	{
		Description: "init command in empty project with arg 'maven' and '--server --project' flags runs in remote mode",
		Command:     "init",
		CIUser:      e2e.LoggedInCIUser,
		// note: --server is set by the e2e runner, because it is passed to e2e.RunTests
		Args:         []string{"maven --project=prj-testtesttesttest"},
		SampleFolder: []string{"empty"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.NotContains(t, output.Stdall, "Do you want to initialize cifuzz for this project in remote (recommended) or local-only mode?")
			assert.Contains(t, output.Stdall, "Running in remote mode.")
			assert.Contains(t, output.Stdall, "Configuration saved in cifuzz.yaml")
			assert.NotContains(t, output.Stdall, "Failed to create config")
			output.FileExists("cifuzz.yaml")
			output.FileContains("cifuzz.yaml", []string{"server:", "project: prj-testtesttesttest"})
		},
	},
}

var nodeInitTests = &[]e2e.TestCase{
	{
		Description:  "init command in Node.js (JS) project succeeds and creates a config file",
		Command:      "init",
		Args:         []string{"js"},
		SampleFolder: []string{"nodejs"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.Contains(t, output.Stdall, "To enable fuzz testing in your project, add a dev-dependency to @jazzer.js/jest-runner")
			assert.Contains(t, output.Stdall, "Configuration saved in cifuzz.yaml")
			assert.NotContains(t, output.Stdall, "Failed to create config")
			output.FileExists("cifuzz.yaml")
		},
	},
	{
		Description:  "init command in Node.js (TS) project succeeds and creates a config file",
		Command:      "init",
		Args:         []string{"ts"},
		SampleFolder: []string{"node-typescript"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			assert.EqualValues(t, 0, output.ExitCode)
			assert.Contains(t, output.Stdall, "To enable fuzz testing in your project, add a dev-dependency to @jazzer.js/jest-runner")
			assert.Contains(t, output.Stdall, "'jest.config.ts'")
			assert.Contains(t, output.Stdall, "To introduce the fuzz function types globally, add the following import to globals.d.ts:")
			assert.Contains(t, output.Stdall, "Configuration saved in cifuzz.yaml")
			assert.NotContains(t, output.Stdall, "Failed to create config")
			output.FileExists("cifuzz.yaml")
		},
	},
}

func TestInit(t *testing.T) {
	e2e.RunTests(t, *initTests, nil)
}

func TestInitWithServer(t *testing.T) {
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = mockserver.ReturnResponseIfValidToken(t, mockserver.ProjectsJSON)
	e2e.RunTests(t, *remoteModeTests, server)
}

func TestInitForNodejs(t *testing.T) {
	e2e.RunTests(t, *nodeInitTests, nil)
}
