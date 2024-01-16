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
		return maven.GetSourceDirs(projectDir)
	}
	return []string{filepath.Join(projectDir, "src", "main")}, nil
}

func TestDirs(projectDir string, buildSystem string) ([]string, error) {
	if buildSystem == config.BuildSystemGradle {
		return gradle.GetTestSourceSets(projectDir)
	} else if buildSystem == config.BuildSystemMaven {
		return maven.GetTestDirs(projectDir)
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
		gradle.GetOverriddenJazzerVersion(projectDir)
	} else if buildSystem == config.BuildSystemMaven {
		maven.GetOverriddenJazzerVersion(projectDir)
	}
}
