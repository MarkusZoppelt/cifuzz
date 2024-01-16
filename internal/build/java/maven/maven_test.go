package maven

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/integration-tests/shared"
)

func Test_GetTestDirs(t *testing.T) {
	projectDir := shared.CopyTestdataDir(t, "maven")
	t.Logf("Project dir: %s", projectDir)

	testDirs, err := GetTestDirs(projectDir)
	require.NoError(t, err)
	require.Len(t, testDirs, 1)
	assert.Equal(t, filepath.Join(projectDir, "src", "test", "java"), testDirs[0])

	// adjust pom.xml to include tag <testSourceDirectory>
	newTestDir := "fuzztests"
	shared.AddLinesToFileAtBreakPoint(t,
		filepath.Join(projectDir, "pom.xml"),
		[]string{fmt.Sprintf("<testSourceDirectory>%s</testSourceDirectory>", newTestDir)},
		"<build>",
		true,
	)
	testDirs, err = GetTestDirs(projectDir)
	require.NoError(t, err)
	require.Len(t, testDirs, 1)
	assert.Equal(t, filepath.Join(projectDir, newTestDir), testDirs[0])
}

func Test_GetSourceDirs(t *testing.T) {
	projectDir := shared.CopyTestdataDir(t, "maven")

	sourceDirs, err := GetSourceDirs(projectDir)
	require.NoError(t, err)
	require.Len(t, sourceDirs, 1)
	assert.Equal(t, filepath.Join(projectDir, "src", "main", "java"), sourceDirs[0])

	// adjust pom.xml to include tag <sourceDirectory>
	newSourceDir := "example"
	shared.AddLinesToFileAtBreakPoint(t,
		filepath.Join(projectDir, "pom.xml"),
		[]string{fmt.Sprintf("<sourceDirectory>%s</sourceDirectory>", newSourceDir)},
		"<build>",
		true,
	)
	sourceDirs, err = GetSourceDirs(projectDir)
	require.NoError(t, err)
	require.Len(t, sourceDirs, 1)
	assert.Equal(t, filepath.Join(projectDir, newSourceDir), sourceDirs[0])
}
