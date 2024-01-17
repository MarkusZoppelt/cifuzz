package dependencies

import (
	"fmt"

	"github.com/Masterminds/semver"

	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
)

type Key string

const (
	Bazel          Key = "bazel"
	Clang          Key = "clang"
	CMake          Key = "cmake"
	LLVMCov        Key = "llvm-cov"
	LLVMSymbolizer Key = "llvm-symbolizer"
	LLVMProfData   Key = "llvm-profdata"

	GenHTML Key = "genhtml"
	Perl    Key = "perl"

	Java           Key = "java"
	Maven          Key = "mvn"
	MavenExtension Key = "CI Fuzz Maven extension"
	Gradle         Key = "gradle"
	GradlePlugin   Key = "CI Fuzz Gradle plugin"

	Node Key = "node"

	VisualStudio Key = "Visual Studio"

	MessageVersion             = "CI Fuzz requires %s version >=%s but found %s"
	MessageMissing             = "CI Fuzz requires %s, but it is not installed or can't be accessed"
	MessageInstallInstructions = `For install instructions see:

  https://docs.code-intelligence.com/ci-fuzz/how-to/ci-fuzz-installation`
)

// Dependency represents a single dependency
type Dependency struct {
	finder runfiles.RunfilesFinder

	Key        Key
	MinVersion semver.Version
	// these fields are used to implement custom logic to
	// retrieve version or installation information for the
	// specific dependency
	GetVersion func(*Dependency, string) (*semver.Version, error)
	Installed  func(*Dependency, string) bool
}

// helper to easily check against functions from the runfiles.RunfilesFinder interface
func (dep *Dependency) checkFinder(finderFunc func() (string, error)) bool {
	if _, err := finderFunc(); err != nil {
		log.Debug(err)
		return false
	}
	return true
}

// Check iterates of a list of dependencies and checks if they are fulfilled
func Check(keys []Key, projectDir string) error {
	err := check(keys, deps, runfiles.Finder, projectDir)
	if err != nil {
		return err
	}

	return nil
}

func Version(key Key, projectDir string) (*semver.Version, error) {
	dep, found := deps[key]
	if !found {
		panic(fmt.Sprintf("Undefined dependency %s", key))
	}

	dep.finder = runfiles.Finder
	return dep.GetVersion(dep, projectDir)
}

func check(keys []Key, deps Dependencies, finder runfiles.RunfilesFinder, projectDir string) error {
	for _, key := range keys {
		dep, found := deps[key]
		if !found {
			panic(fmt.Sprintf("Undefined dependency %s", key))
		}

		dep.finder = finder

		if !dep.Installed(dep, projectDir) {
			return fmt.Errorf("%s\n%s", fmt.Sprintf(MessageMissing, dep.Key), MessageInstallInstructions)
		}

		if dep.MinVersion.Equal(semver.MustParse("0.0.0")) {
			log.Debugf("Checking dependency: %s ", dep.Key)
		} else {
			log.Debugf("Checking dependency: %s version >= %s", dep.Key, dep.MinVersion.String())
		}

		currentVersion, err := dep.GetVersion(dep, projectDir)
		if err != nil {
			log.Warnf("Unable to get current version for %s: %v", dep.Key, err)
			// we want to be lenient if we were not able to extract the version
			continue
		}

		if currentVersion.Compare(&dep.MinVersion) == -1 {
			return fmt.Errorf(MessageVersion, dep.Key, dep.MinVersion.String(), currentVersion.String())
		}
	}

	return nil
}
