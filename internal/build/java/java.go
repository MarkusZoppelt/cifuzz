package java

import (
	"path/filepath"

	"code-intelligence.com/cifuzz/internal/build/java/gradle"
	"code-intelligence.com/cifuzz/internal/build/java/maven"
	"code-intelligence.com/cifuzz/internal/config"
)

func SourceDirs(projectDir string, buildSystem string) ([]string, error) {
	if buildSystem == config.BuildSystemGradle {
		return gradle.GetMainSourceSets(projectDir)
	} else if buildSystem == config.BuildSystemMaven {
		sourceDir, err := maven.GetSourceDir(projectDir)
		if err != nil {
			return nil, err
		}
		if sourceDir != "" {
			return []string{sourceDir}, nil
		}
		return nil, nil
	}
	return []string{filepath.Join(projectDir, "src", "main")}, nil
}

func TestDirs(projectDir string, buildSystem string) ([]string, error) {
	if buildSystem == config.BuildSystemGradle {
		return gradle.GetTestSourceSets(projectDir)
	} else if buildSystem == config.BuildSystemMaven {
		testDir, err := maven.GetTestDir(projectDir)
		if err != nil {
			return nil, err
		}
		if testDir != "" {
			return []string{testDir}, nil
		}
		return nil, nil
	}
	return []string{filepath.Join(projectDir, "src", "test")}, nil
}

func RootDirectory(projectDir string, buildSystem string) (string, error) {
	if buildSystem == config.BuildSystemGradle {
		return gradle.GetRootDirectory(projectDir)
	} else if buildSystem == config.BuildSystemMaven {
		return maven.GetRootDirectory(projectDir)
	}

	return projectDir, nil
}

func CheckOverriddenJazzerVersion(projectDir string, buildSystem string) {
	if buildSystem == config.BuildSystemGradle {
		// not yet implemented
	} else if buildSystem == config.BuildSystemMaven {
		maven.GetOverriddenJazzerVersion(projectDir)
	}
}
