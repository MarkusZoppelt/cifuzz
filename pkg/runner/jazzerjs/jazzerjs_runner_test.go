package jazzerjs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/internal/testutil"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer"
)

func TestRunner_FuzzerEnvironment(t *testing.T) {
	r := NewRunner(&RunnerOptions{
		LibfuzzerOptions: &libfuzzer.RunnerOptions{
			EngineArgs: []string{"-seed=1", "-run=1", "-timeout=10000"},
		},
	})

	env, err := r.FuzzerEnvironment()
	require.NoError(t, err)

	assert.Contains(t, env, "JAZZER_FUZZ=1")
	assert.Contains(t, env, "JAZZER_FUZZER_OPTIONS=[\"-seed=1\",\"-run=1\"]")
	assert.Contains(t, env, "JAZZER_TIMEOUT=10000")
}

// TestRunner_FuzzerEnvironmentWithJazzerJSRC checks that values set in a
// .jazzerjsrc are always prioritized over values set via the engine args.
func TestRunner_FuzzerEnvironmentWithJazzerJSRC(t *testing.T) {
	tempDir := testutil.MkdirTemp(t, "", "nodets-test-*")

	jazzerJSRCContent := `{
	"fuzzerOptions": ["-max_len=8192", "-seed=10"],
	"timeout": 1000
}`
	err := os.WriteFile(filepath.Join(tempDir, ".jazzerjsrc"), []byte(jazzerJSRCContent), 0o644)
	require.NoError(t, err)

	r := NewRunner(&RunnerOptions{
		LibfuzzerOptions: &libfuzzer.RunnerOptions{
			ProjectDir: tempDir,
			EngineArgs: []string{"-seed=1", "-run=1", "-timeout=10000"},
		},
	})

	env, err := r.FuzzerEnvironment()
	require.NoError(t, err)

	assert.Contains(t, env, "JAZZER_FUZZ=1")
	assert.Contains(t, env, "JAZZER_FUZZER_OPTIONS=[\"-max_len=8192\",\"-seed=10\",\"-run=1\"]")
}

// TestRunner_FuzzerEnvironmentWithJazzerJSRC_OnlyDuplicates tests that the
// environment variables for engine args are only set if they add an argument
// that is not present in the .jazzerjsrc.
func TestRunner_FuzzerEnvironmentWithJazzerJSRC_OnlyDuplicates(t *testing.T) {
	tempDir := testutil.MkdirTemp(t, "", "nodets-test-*")

	jazzerJSRCContent := `{
	"fuzzerOptions": ["-seed=1"],
	"timeout": 1000
}`
	err := os.WriteFile(filepath.Join(tempDir, ".jazzerjsrc"), []byte(jazzerJSRCContent), 0o644)
	require.NoError(t, err)

	r := NewRunner(&RunnerOptions{
		LibfuzzerOptions: &libfuzzer.RunnerOptions{
			ProjectDir: tempDir,
			EngineArgs: []string{"-seed=10", "-timeout=100"},
		},
	})

	env, err := r.FuzzerEnvironment()
	require.NoError(t, err)

	assert.Len(t, env, 1)
	assert.Contains(t, env, "JAZZER_FUZZ=1")
}
