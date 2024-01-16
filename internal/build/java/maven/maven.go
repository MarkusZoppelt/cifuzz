package maven

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/util/fileutil"
)

var (
	classpathRegex         = regexp.MustCompile("(?m)^cifuzz.test.classpath=(?P<classpath>.*)$")
	rootDirRegex           = regexp.MustCompile("(?m)^cifuzz.rootDir=(?P<rootDir>.*)$")
	testSourceFoldersRegex = regexp.MustCompile("(?m)^cifuzz.test.source-folders=(?P<testSourceFolders>.*)$")
	mainSourceFoldersRegex = regexp.MustCompile("(?m)^cifuzz.main.source-folders=(?P<mainSourceFolders>.*)$")
	jazzerVersionRegex     = regexp.MustCompile("(?m)^cifuzz.deps.jazzer-version=(?P<jazzerVersion>.*)$")
)

type ParallelOptions struct {
	Enabled bool
	NumJobs uint
}

type BuilderOptions struct {
	ProjectDir string
	Parallel   ParallelOptions
	Stdout     io.Writer
	Stderr     io.Writer
}

func (opts *BuilderOptions) Validate() error {
	// Check that the project dir is set
	if opts.ProjectDir == "" {
		return errors.New("ProjectDir is not set")
	}
	// Check that the project dir exists and can be accessed
	_, err := os.Stat(opts.ProjectDir)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

type Builder struct {
	*BuilderOptions
}

func NewBuilder(opts *BuilderOptions) (*Builder, error) {
	err := opts.Validate()
	if err != nil {
		return nil, err
	}

	b := &Builder{BuilderOptions: opts}

	return b, err
}

func (b *Builder) Build() (*build.BuildResult, error) {
	deps, err := GetDependencies(b.ProjectDir, b.Parallel)
	if err != nil {
		return nil, err
	}

	result := &build.BuildResult{
		// BuildDir is not used by Jazzer
		BuildDir:    "",
		RuntimeDeps: deps,
	}

	return result, nil
}

func GetDependencies(projectDir string, parallel ParallelOptions) ([]string, error) {
	var flags []string
	if parallel.Enabled {
		flags = append(flags, "-T")
		if parallel.NumJobs != 0 {
			flags = append(flags, fmt.Sprint(parallel.NumJobs))
		} else {
			// Use one thread per cpu core
			flags = append(flags, "1C")
		}
	}

	args := append(flags, "test-compile", "-DcifuzzPrintTestClasspath")
	cmd := runMaven(projectDir, args)
	output, err := cmd.Output()
	if err != nil {
		log.Debugf("%s", string(output))
		return nil, cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}

	classpath := classpathRegex.FindStringSubmatch(string(output))
	deps := strings.Split(strings.TrimSpace(classpath[1]), string(os.PathListSeparator))

	// Add jacoco cli and java agent JAR paths
	cliJarPath, err := runfiles.Finder.JacocoCLIJarPath()
	if err != nil {
		return nil, err
	}
	agentJarPath, err := runfiles.Finder.JacocoAgentJarPath()
	if err != nil {
		return nil, err
	}
	deps = append(deps, cliJarPath, agentJarPath)
	return deps, nil
}

func runMaven(projectDir string, args []string) *exec.Cmd {
	// remove color and transfer progress from output
	args = append(args, "-B", "--no-transfer-progress")
	cmd := exec.Command("mvn", args...) // TODO find ./mvnw if available (unify with MavenRunner in coverage.go)
	cmd.Dir = projectDir

	log.Debugf("Working directory: %s", cmd.Dir)
	log.Debugf("Command: %s", cmd.String())

	return cmd
}

func GetRootDirectory(projectDir string) (string, error) {
	cmd := runMaven(projectDir, []string{"validate", "-q", "-DcifuzzPrintRootDir"})
	output, err := cmd.Output()
	if err != nil {
		log.Debugf("%s\n", string(output))
		return "", cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}

	result := rootDirRegex.FindStringSubmatch(string(output))
	if result == nil {
		return "", errors.New("Unable to parse maven root directory")
	}
	rootDir := strings.TrimSpace(result[1])

	return rootDir, nil
}

func GetTestDirs(projectDir string) ([]string, error) {
	cmd := runMaven(projectDir, []string{"validate", "-q", "-DcifuzzPrintTestSourceFolders"})
	output, err := cmd.Output()
	if err != nil {
		log.Debugf("%s\n", string(output))
		return nil, cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}

	result := testSourceFoldersRegex.FindStringSubmatch(string(output))
	if result == nil {
		return nil, errors.New("Unable to parse maven test sources.")
	}
	paths := strings.Split(strings.TrimSpace(result[1]), string(os.PathListSeparator))

	// only return valid paths
	var testDirs []string
	for _, path := range paths {
		exists, err := fileutil.Exists(path)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error checking if Maven test directory %s exists", path)
		}
		if exists {
			testDirs = append(testDirs, path)
		}
	}

	log.Debugf("Found maven test sources at: %s", testDirs)
	return testDirs, nil
}

func GetSourceDirs(projectDir string) ([]string, error) {
	cmd := runMaven(projectDir, []string{"validate", "-q", "-DcifuzzPrintMainSourceFolders"})
	output, err := cmd.Output()
	if err != nil {
		log.Debugf("%s\n", string(output))
		return nil, errors.WithMessagef(err, "Failed to get source directory of project")
	}

	result := mainSourceFoldersRegex.FindStringSubmatch(string(output))
	if result == nil {
		return nil, errors.New("Unable to parse maven main sources.")
	}
	paths := strings.Split(strings.TrimSpace(result[1]), string(os.PathListSeparator))

	// only return valid paths
	var sourceDirs []string
	for _, path := range paths {
		exists, err := fileutil.Exists(path)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error checking if Maven source directory %s exists", path)
		}
		if exists {
			sourceDirs = append(sourceDirs, path)
		}
	}

	log.Debugf("Found maven main sources at: %s", sourceDirs)
	return sourceDirs, nil
}

func GetOverriddenJazzerVersion(projectDir string) string {
	cmd := runMaven(projectDir, []string{"validate", "-q", "-DcifuzzPrintJazzerVersion"})
	output, err := cmd.Output()
	if err != nil {
		log.Debugf("%s\n", string(output))
		return ""
	}

	result := jazzerVersionRegex.FindStringSubmatch(string(output))
	if result == nil {
		return ""
	}
	jazzerVersion := strings.TrimSpace(result[1])
	if jazzerVersion != "" {
		log.Warnf("Overriding default Jazzer version with version %s.", jazzerVersion)
	}

	return jazzerVersion
}

func GetPluginVersion(projectDir string) (string, error) {
	// not using runMaven() here to avoid multiple log prints of command
	cmd := exec.Command("mvn", "-B", "--no-transfer-progress", "validate", "-q", "-DcifuzzPrintExtensionVersion")
	cmd.Dir = projectDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		log.Debugf("Command: %s", cmd.String())
		_, writeErr := stderr.Write(stderr.Bytes())
		if writeErr != nil {
			log.Errorf(errors.WithStack(writeErr), "Failed to write command output to stderr: %v", writeErr.Error())
		}
		return "", errors.WithStack(err)
	}

	if len(output) == 0 {
		return "", nil
	}
	return strings.TrimSpace(strings.TrimPrefix(string(output), "cifuzz.plugin.version=")), nil
}
