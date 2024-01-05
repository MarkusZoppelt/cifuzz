package container_test

import (
	"path"
	"testing"

	"code-intelligence.com/cifuzz/e2e"
)

var coverageTests = &[]e2e.TestCase{
	{
		Description:   "coverage generates coverage report",
		Command:       "coverage",
		Args:          []string{"--format=lcov --output=coverage com.example.FuzzTestCase::myFuzzTest"},
		SampleFolder:  []string{"maven-default", "gradle-default", "gradle-with-existing-junit"},
		ToolsRequired: []string{"java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().
				FileContains(path.Join("coverage", "report.lcov"), []string{
					"/com/example/ExploreMe",
					"LF:12", // Lines found
					"LH:5",  // Lines hit
					"BRF:8", // Branches found
					"BRH:1", // Branches hit

					// Maven coverage does not contain fuzz test
					//"/com/example/FuzzTestCase",
					//"LF:7",
					//"LH:7",
					//"BRF:0",
					//"BRH:0",
				})
		},
	},
}

func TestCoverage(t *testing.T) {
	e2e.RunTests(t, *coverageTests, nil)
}
