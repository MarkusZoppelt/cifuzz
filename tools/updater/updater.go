package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"code-intelligence.com/cifuzz/pkg/log"
)

func main() {
	flags := pflag.NewFlagSet("updater", pflag.ExitOnError)
	dep := flags.String("dependency", "", "which dependency to update eg. gradle-plugin, jazzer, jazzerjs")
	versionFlag := flags.String("version", "", "target version to update to, for example 1.2.3")
	handleErr(flags.Parse(os.Args))

	if *dep == "" && *versionFlag != "" {
		handleErr(errors.New("version flag can only be used in combination with dependency flag"))
	}

	var deps []string
	if *dep == "" {
		deps = []string{"gradle-plugin", "maven-extension", "jazzer", "jazzerjs"}
	} else {
		deps = []string{*dep}
	}

	for _, dep := range deps {
		updateDependencyForDep(dep, *versionFlag)
	}
}

func updateDependencyForDep(dep string, version string) {
	var targetVersion string
	if version == "" {
		log.Infof("Search for latest version of %s", dep)
		targetVersion = findLatestVersion(dep)
	} else {
		_, err := semver.NewVersion(version)
		handleErr(err)
		targetVersion = version
	}

	log.Infof("Updating %s to: %s", dep, targetVersion)

	switch dep {
	case "gradle-plugin":
		re := regexp.MustCompile(`("com.code-intelligence.cifuzz"\)? version ")(?P<version>\d+.\d+.\d+.*|dev)(")`)
		paths := []string{
			"e2e/tests/samples/gradle-default/build.gradle",
			"e2e/tests/samples/gradle-with-existing-junit/build.gradle",
			"examples/gradle-kotlin/build.gradle.kts",
			"examples/gradle-multi/testsuite/build.gradle.kts",
			"examples/gradle/build.gradle",
			// integration-test projects are updated dynamically in the tests
			"internal/bundler/testdata/jazzer/gradle/multi-custom/testsuite/build.gradle.kts",
			"internal/cmdutils/resolve/testdata/gradle/build.gradle",
			"pkg/messaging/instructions/gradle",
			"pkg/messaging/instructions/gradlekotlin",
			"test/projects/gradle/app/build.gradle.kts",
			"test/projects/gradle/testsuite/build.gradle.kts",
		}
		for _, path := range paths {
			updateFile(path, targetVersion, re)
		}

		re = regexp.MustCompile(`(com.code-intelligence.cifuzz:com.code-intelligence.cifuzz.gradle.plugin:)(?P<version>\d+.\d+.\d+.*|dev)(")`)
		updateFile("tools/dependency-bundler/bundle-dependencies.sh", targetVersion, re)

		re = regexp.MustCompile(`(GradlePlugin,\n\s*MinVersion:\s*\*semver\.MustParse\(")(?P<version>\d+.\d+.\d+|dev)(")`)
		updateFile("pkg/dependencies/definitions.go", targetVersion, re)

	case "maven-extension":
		re := regexp.MustCompile(`(<artifactId>cifuzz-maven-extension<\/artifactId>\s*<version>)(?P<version>\d+.\d+.\d+.*|dev)(<\/version>)`)
		paths := []string{
			"e2e/tests/samples/maven-default/pom.xml",
			"examples/maven-multi/pom.xml",
			"examples/maven/pom.xml",
			"integration-tests/errors/java/testdata-sql-ldap/pom.xml",
			"integration-tests/errors/java/testdata/pom.xml",
			// "integration-tests/java-maven/testdata/pom.xml" not required, as it's dynamically updated in the tests
			"integration-tests/java-maven-spring/testdata/pom.xml",
			"internal/build/java/maven/testdata/pom.xml",
			"internal/bundler/testdata/jazzer/maven/pom.xml",
			"internal/cmdutils/resolve/testdata/maven/pom.xml",
			"pkg/messaging/instructions/maven",
			"test/projects/maven/pom.xml",
		}
		for _, path := range paths {
			updateFile(path, targetVersion, re)
		}

		re = regexp.MustCompile(`(com.code-intelligence:cifuzz-maven-extension:)(?P<version>\d+.\d+.\d+.*|dev)(")`)
		updateFile("tools/dependency-bundler/bundle-dependencies.sh", targetVersion, re)

		re = regexp.MustCompile(`(MavenExtension,\n\s*MinVersion:\s*\*semver\.MustParse\(")(?P<version>\d+.\d+.\d+|dev)(")`)
		updateFile("pkg/dependencies/definitions.go", targetVersion, re)

	case "jazzer":
		re := regexp.MustCompile(`(<artifactId>jazzer-junit<\/artifactId>\s*<version>)(?P<version>\d+.\d+.\d+.*|dev)(<\/version>)`)
		updateFile("tools/list-fuzz-tests/pom.xml", targetVersion, re)

		re = regexp.MustCompile(`(com.code-intelligence:jazzer-junit:)(?P<version>\d+.\d+.\d+.*|dev)(")`)
		updateFile("tools/dependency-bundler/bundle-dependencies.sh", targetVersion, re)

	case "jazzerjs":
		updateJazzerNpm("examples/nodejs", targetVersion)
		updateJazzerNpm("examples/nodejs-typescript", targetVersion)

		re := regexp.MustCompile(`(@jazzer\.js\/jest-runner@)(?P<version>\d+.\d+.\d+|dev)`)
		updateFile("pkg/messaging/instructions/nodejs", targetVersion, re)
		updateFile("pkg/messaging/instructions/nodets", targetVersion, re)

		re = regexp.MustCompile(`("@jazzer\.js\/jest-runner": "\^)(?P<version>\d+.\d+.\d+|dev)(")`)
		updateFile("integration-tests/errors/nodejs/testdata/package.json", targetVersion, re)
	default:
		log.Error(errors.New("unsupported dependency selected"))
		os.Exit(1)
	}
}

func updateJazzerNpm(path string, version string) {
	cmd := exec.Command("npm", "install", "--save-dev", fmt.Sprintf("@jazzer.js/jest-runner@%s", version))
	cmd.Dir = path
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	handleErr(err)
}

func updateFile(path string, version string, re *regexp.Regexp) {
	content, err := os.ReadFile(path)
	handleErr(err)
	buildFile := string(content)

	s := re.ReplaceAllString(buildFile, fmt.Sprintf(`${1}%s${3}`, version))

	err = os.WriteFile(path, []byte(s), 0x644)
	handleErr(err)

	fmt.Printf("updated %s to %s\n", path, version)
}

func handleErr(err error) {
	if err != nil {
		log.Error(errors.WithStack(err))
		os.Exit(1)
	}
}
