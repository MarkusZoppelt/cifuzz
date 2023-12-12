package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectNameToNID tests the conversion of project names to NIDs
func TestProjectNameToNID(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		// For old projects this guarantees to find the NID
		"projects%2Ftest-73d94c96": "prj-000073d94c96",
		"projects/test-73d94c96":   "prj-000073d94c96",
		"test-73d94c96":            "prj-000073d94c96",
		"prj-000073d94c96":         "prj-000073d94c96",

		// For new projects the external-id equals the NID
		"prj-ow6h1UIwHXTr":          "prj-ow6h1UIwHXTr",
		"projects/prj-ow6h1UIwHXTr": "prj-ow6h1UIwHXTr",
	}

	for projectName, expectedNID := range testCases {
		nid, err := ProjectNameToNID(projectName)
		require.NoError(t, err)
		assert.Equal(t, expectedNID, nid)
	}
}
