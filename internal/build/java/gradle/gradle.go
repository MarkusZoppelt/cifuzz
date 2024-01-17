package gradle

import (
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/regexutil"
)

var (
	classpathRegex         = regexp.MustCompile("(?m)^cifuzz.test.classpath=(?P<classpath>.*)$")
	rootDirRegex           = regexp.MustCompile("(?m)^cifuzz.rootDir=(?P<rootDir>.*)$")
	testSourceFoldersRegex = regexp.MustCompile("(?m)^cifuzz.test.source-folders=(?P<testSourceFolders>.*)$")
	mainSourceFoldersRegex = regexp.MustCompile("(?m)^cifuzz.main.source-folders=(?P<mainSourceFolders>.*)$")
	jazzerVersionRegex     = regexp.MustCompile("(?m)^cifuzz.deps.jazzer-version=(?P<jazzerVersion>.*)$")
	pluginVersionRegex     = regexp.MustCompile(`(?m)^cifuzz.plugin.version=(?P<version>\d+.\d+[.\d]*)`)
)

func FindGradleWrapper(projectDir string) (string, error) {
	wrapper := "gradlew"
	if runtime.GOOS == "windows" {
		wrapper = "gradlew.bat"
	}

	return fileutil.SearchFileBackwards(projectDir, wrapper)
}

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
	deps, err := GetDependencies(b.ProjectDir)
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

func GetDependencies(projectDir string) ([]string, error) {
	cmd, err := buildGradleCommand(projectDir, []string{"cifuzzPrintTestClasspath", "-q"})
	if err != nil {
		return nil, err
	}
	log.Debugf("Command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
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

// GetGradleCommand returns the name of the gradle command.
// The gradle wrapper is preferred to use and gradle
// acts as a fallback command.
func GetGradleCommand(projectDir string) (string, error) {
	wrapper, err := FindGradleWrapper(projectDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if wrapper != "" {
		return wrapper, nil
	}

	gradleCmd, err := runfiles.Finder.GradlePath()
	if err != nil {
		return "", err
	}
	return gradleCmd, nil
}

func buildGradleCommand(projectDir string, args []string) (*exec.Cmd, error) {
	gradleCmd, err := GetGradleCommand(projectDir)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(gradleCmd, args...)
	cmd.Dir = projectDir

	return cmd, nil
}

func GetRootDirectory(projectDir string) (string, error) {
	cmd, err := buildGradleCommand(projectDir, []string{"cifuzzPrintRootDir", "-q"})
	if err != nil {
		return "", nil
	}

	log.Debugf("Command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		return "", cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}
	result := rootDirRegex.FindStringSubmatch(string(output))
	if result == nil {
		return "", errors.New("Unable to parse gradle root directory from init script.")
	}
	rootDir := strings.TrimSpace(result[1])

	return rootDir, nil
}

func GetTestSourceSets(projectDir string) ([]string, error) {
	cmd, err := buildGradleCommand(projectDir, []string{"cifuzzPrintTestSourceFolders", "-q"})
	if err != nil {
		return nil, err
	}

	log.Debugf("Command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		return nil, cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}
	result := testSourceFoldersRegex.FindStringSubmatch(string(output))
	if result == nil {
		return nil, errors.New("Unable to parse gradle test sources.")
	}
	paths := strings.Split(strings.TrimSpace(result[1]), string(os.PathListSeparator))

	// only return valid paths
	var sourceSets []string
	for _, path := range paths {
		exists, err := fileutil.Exists(path)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error checking if Gradle test source path %s exists", path)
		}
		if exists {
			sourceSets = append(sourceSets, path)
		}
	}

	log.Debugf("Found gradle test sources at: %s", sourceSets)
	return sourceSets, nil
}

func GetMainSourceSets(projectDir string) ([]string, error) {
	cmd, err := buildGradleCommand(projectDir, []string{"cifuzzPrintMainSourceFolders", "-q"})
	if err != nil {
		return nil, err
	}

	log.Debugf("Command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		return nil, cmdutils.WrapExecError(errors.WithStack(err), cmd)
	}
	result := mainSourceFoldersRegex.FindStringSubmatch(string(output))
	if result == nil {
		return nil, errors.New("Unable to parse gradle main sources.")
	}
	paths := strings.Split(strings.TrimSpace(result[1]), string(os.PathListSeparator))

	// only return valid paths
	var sourceSets []string
	for _, path := range paths {
		exists, err := fileutil.Exists(path)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error checking if Gradle main source path %s exists", path)
		}
		if exists {
			sourceSets = append(sourceSets, path)
		}
	}

	log.Debugf("Found gradle main sources at: %s", sourceSets)
	return sourceSets, nil
}

func GetOverriddenJazzerVersion(projectDir string) string {
	cmd, err := buildGradleCommand(projectDir, []string{"cifuzzPrintJazzerVersion", "-q"})
	if err != nil {
		return ""
	}
	log.Debugf("Command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
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
	cmd, err := buildGradleCommand(projectDir, []string{"cifuzzPrintPluginVersion", "-q"})
	if err != nil {
		return "", err
	}

	outputBs, err := cmd.CombinedOutput()
	output := string(outputBs)
	if err != nil {
		if strings.Contains(output, "Task 'cifuzzPrintPluginVersion' not found") {
			return "", nil
		}

		log.Debugf("Command: %s", cmd.String())
		log.Print(output)
		return "", errors.WithStack(err)
	}

	match, found := regexutil.FindNamedGroupsMatch(pluginVersionRegex, output)
	if !found {
		log.Debugf("Command: %s", cmd.String())
		log.Print(output)
		return "", errors.New("Failed to extract version from task 'cifuzzPrintPluginVersion'")
	}

	return match["version"], nil
}
