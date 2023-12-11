package reproduce

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/integration-tests/shared"
	"code-intelligence.com/cifuzz/integration-tests/shared/mockserver"
	"code-intelligence.com/cifuzz/internal/builder"
	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/internal/testutil"
	"code-intelligence.com/cifuzz/pkg/dependencies"
	"code-intelligence.com/cifuzz/pkg/finding"
	"code-intelligence.com/cifuzz/pkg/runfiles"
)

func TestReproduceCmdFailsIfNoCIFuzzProject(t *testing.T) {
	// Set finder install dir to project root. This way the
	// finder finds the required error-details.json in the
	// project dir instead of the cifuzz install dir.
	sourceDir, err := builder.FindProjectDir()
	if err != nil {
		log.Fatalf("Failed to find cifuzz project dir")
	}
	runfiles.Finder = runfiles.RunfilesFinderImpl{InstallDir: sourceDir}
	projectDir := testutil.MkdirTemp(t, "", "reproduce-cmd-test-")

	opts := &options{
		ProjectDir: projectDir,
		ConfigDir:  projectDir,
	}
	_, stdErr, err := cmdutils.ExecuteCommand(t, newWithOptions(opts), os.Stdin, "test_finding")
	require.Error(t, err)
	assert.Contains(t, stdErr, "Failed to parse cifuzz.yaml")
}

func TestClangMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("clang is not needed on windows and will be provided by Visual Studio")
	}

	dependencies.TestMockAllDeps(t)
	// let the clang dep fail
	dependencies.OverwriteUninstalled(dependencies.GetDep(dependencies.Clang))

	// clone the example project because this command needs to parse an actual
	// project config... if there is none it will fail before the dependency check
	testutil.BootstrapExampleProjectForTest(t, "run-cmd-test", config.BuildSystemCMake)

	_, stdErr, err := cmdutils.ExecuteCommand(t, New(), os.Stdin, "my_fuzz_test")
	require.Error(t, err)
	assert.Contains(t, stdErr, fmt.Sprintf(dependencies.MessageMissing, "clang"))
}

func TestVisualStudioMissing(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only needed on windows")
	}

	dependencies.TestMockAllDeps(t)

	dep := dependencies.GetDep(dependencies.VisualStudio)
	version := dependencies.OverwriteGetVersionWith0(dep)

	// clone the example project because this command needs to parse an actual
	// project config... if there is none it will fail before the dependency check
	testutil.BootstrapExampleProjectForTest(t, "run-cmd-test", config.BuildSystemCMake)

	_, stdErr, err := cmdutils.ExecuteCommand(t, New(), os.Stdin, "my_fuzz_test")
	require.Error(t, err)
	assert.Contains(t, stdErr,
		fmt.Sprintf(dependencies.MessageVersion, "Visual Studio", dep.MinVersion.String(), version))
}

func TestIntegration_ReproduceFailsForWrongBuildSystem(t *testing.T) {
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip()
	}

	// Set finder install dir to project root. This way the
	// finder finds the required error-details.json in the
	// project dir instead of the cifuzz install dir.
	sourceDir, err := builder.FindProjectDir()
	if err != nil {
		log.Fatalf("Failed to find cifuzz project dir")
	}
	runfiles.Finder = runfiles.RunfilesFinderImpl{InstallDir: sourceDir}
	projectDir := testutil.BootstrapExampleProjectForTest(t, "reproduce-cmd-test-", config.BuildSystemCMake)

	// create a local finding
	finding := &finding.Finding{
		Origin:    "Local",
		Name:      "test_finding",
		InputData: []byte("test"),
	}
	err = finding.Save(projectDir)
	require.NoError(t, err)

	opts := &options{
		ProjectDir: projectDir,
		ConfigDir:  projectDir,
	}
	_, stderr, err := cmdutils.ExecuteCommand(t, newWithOptions(opts), os.Stdin, "test_finding")
	require.Error(t, err)
	assert.Contains(t, stderr, "Only other build systems are supported for now.")
}

func TestIntegration_ReproduceLocalFinding(t *testing.T) {
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip()
	}

	// install cifuzz
	testutil.RegisterTestDepOnCIFuzz()
	installDir := shared.InstallCIFuzzInTemp(t)
	cifuzz := builder.CIFuzzExecutablePath(filepath.Join(installDir, "bin"))

	projectDir := testutil.BootstrapExampleProjectForTest(t, "reproduce-cmd-test-", config.BuildSystemOther)

	// create a local finding
	finding := &finding.Finding{
		Origin:    "Local",
		Name:      "test_finding",
		FuzzTest:  "my_fuzz_test",
		InputData: []byte("test"),
	}
	err := finding.Save(projectDir)
	require.NoError(t, err)

	cmd := exec.Command(cifuzz, "reproduce", "test_finding")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output), "Running: .cifuzz-findings/test_finding/crashing-input")
}

func TestIntegration_ReproduceRemoteFinding(t *testing.T) {
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip()
	}

	// install cifuzz
	testutil.RegisterTestDepOnCIFuzz()
	installDir := shared.InstallCIFuzzInTemp(t)
	cifuzz := builder.CIFuzzExecutablePath(filepath.Join(installDir, "bin"))

	testutil.BootstrapExampleProjectForTest(t, "reproduce-cmd-test-", config.BuildSystemOther)

	// setup mock server
	t.Setenv("CIFUZZ_API_TOKEN", mockserver.ValidToken)
	server := mockserver.New(t)
	server.Handlers["/v1/projects"] = mockserver.ReturnResponse(t, mockserver.ProjectsJSON)
	server.Handlers["/v1/projects/my-project/findings"] = mockserver.ReturnResponse(t, mockserver.RemoteFindingsJSON)
	server.Start(t)

	// run reproduce with invalid finding
	cmd := exec.Command(cifuzz, "reproduce", "--server", server.AddressOnHost(), "--project", "my-project", "test_finding")
	output, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(output), "test_finding not found in CI Sense project: my-project")

	// run reproduce with valid finding
	cmd = exec.Command(cifuzz, "reproduce", "--server", server.AddressOnHost(), "--project", "my-project", "pensive_flamingo")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output), "Running: .cifuzz-findings/pensive_flamingo/crashing-input")
}
