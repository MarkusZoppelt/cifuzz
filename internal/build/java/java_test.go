package java

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/integration-tests/shared"
	"code-intelligence.com/cifuzz/internal/config"
)

func Test_RootDirectory(t *testing.T) {
	projectDir := shared.CopyTestdataDir(t, "maven-multi")
	t.Logf("Project dir: %s", projectDir)

	rootDir, err := RootDirectory(projectDir, config.BuildSystemMaven)
	require.NoError(t, err)
	assert.Equal(t, projectDir, rootDir)
}
