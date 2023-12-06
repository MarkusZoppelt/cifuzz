package jazzerjs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/pkg/dependencies"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/options"
	fuzzer_runner "code-intelligence.com/cifuzz/pkg/runner"
	"code-intelligence.com/cifuzz/pkg/runner/libfuzzer"
	"code-intelligence.com/cifuzz/util/envutil"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/sliceutil"
)

type RunnerOptions struct {
	LibfuzzerOptions *libfuzzer.RunnerOptions
	TestPathPattern  string
	TestNamePattern  string
	PackageManager   string
}

func (options *RunnerOptions) ValidateOptions() error {
	err := options.LibfuzzerOptions.ValidateOptions()
	if err != nil {
		return err
	}

	if options.TestPathPattern == "" {
		return errors.New("Test name pattern must be specified.")
	}

	return nil
}

// TODO: define in a central place
type Runner struct {
	*RunnerOptions
	*libfuzzer.Runner
}

func NewRunner(options *RunnerOptions) *Runner {
	libfuzzerRunner := libfuzzer.NewRunner(options.LibfuzzerOptions)
	// TODO: handle different fuzzers properly
	libfuzzerRunner.SupportJazzer = false
	libfuzzerRunner.SupportJazzerJS = true
	return &Runner{options, libfuzzerRunner}
}

func (r *Runner) Run(ctx context.Context) error {
	err := r.ValidateOptions()
	if err != nil {
		return err
	}

	// Print version information for debugging purposes
	r.printDebugVersionInfos()

	args := []string{"npx", "jest"}

	// ---------------------------
	// --- fuzz target arguments -
	// ---------------------------
	args = append(args, options.JazzerJSTestPathPatternFlag(r.TestPathPattern))
	args = append(args, options.JazzerJSTestNamePatternFlag(r.TestNamePattern))
	args = append(args, options.JestTestFailureExitCodeFlag(fuzzer_runner.LibFuzzerErrorExitCode))

	env, err := r.FuzzerEnvironment()
	if err != nil {
		return err
	}

	return r.RunLibfuzzerAndReport(ctx, args, env)
}

func (r *Runner) FuzzerEnvironment() ([]string, error) {
	var env []string

	env, err := fuzzer_runner.AddEnvFlags(env, r.EnvVars)
	if err != nil {
		return nil, err
	}
	env, err = envutil.Setenv(env, "JAZZER_FUZZ", "1")
	if err != nil {
		return nil, err
	}

	if len(r.LibfuzzerOptions.EngineArgs) > 0 {
		env, err = r.setEngineArgsAsJazzerFlags(env)
		if err != nil {
			return nil, err
		}
	}

	return env, nil
}

func (r *Runner) Cleanup(ctx context.Context) {
	r.Runner.Cleanup(ctx)
}

func (r *Runner) printDebugVersionInfos() {
	jazzerJSVersion, err := dependencies.JazzerJSVersion()
	if err != nil {
		log.Warn(err)
	} else {
		log.Debugf("JazzerJS version: %s", jazzerJSVersion)
	}

	jestVersion, err := dependencies.JestVersion()
	if err != nil {
		log.Warn(err)
	} else {
		log.Debugf("Jest version: %s", jestVersion)
	}
}

type jazzerJSRC struct {
	Includes      []string `json:"includes"`
	Excludes      []string `json:"excludes"`
	CustomHooks   []string `json:"customHooks"`
	FuzzerOptions []string `json:"fuzzerOptions"`
	Sync          bool     `json:"sync"`
	Timeout       int32    `json:"timeout"`
}

// setEngineArgsAsJazzerFlags sets the engine args for libfuzzer with the
// environment variables JAZZER_FUZZER_OPTIONS and JAZZER_TIMEOUT.
// It checks if a .jazzerjsrc file exists in the project and prioritizes
// those values over the engine args. Setting the JAZZER_FUZZER_OPTIONS or
// JAZZER_TIMEOUT environment variable will make Jazzer.js ignore the values
// from the .jazzerjsrc so those values have to be added in the env too.
func (r *Runner) setEngineArgsAsJazzerFlags(env []string) ([]string, error) {
	// Check if .jazzerjsrc exists and store values
	var rc jazzerJSRC
	jazzerJSRCPath := filepath.Join(r.ProjectDir, ".jazzerjsrc")
	exist, err := fileutil.Exists(jazzerJSRCPath)
	if err != nil {
		return nil, err
	}
	if exist {
		f, err := os.ReadFile(jazzerJSRCPath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		err = json.Unmarshal(f, &rc)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	fuzzerOptions := rc.FuzzerOptions
	for _, arg := range r.LibfuzzerOptions.EngineArgs {
		flag, value, found := strings.Cut(arg, "=")
		if !found {
			continue
		}

		// Timeouts are stored in JAZZER_TIMEOUT environment variable
		// If a timeout has already been set in the .jazzerjsrc,
		// we ignore this flag
		if flag == "-timeout" {
			if rc.Timeout == 0 {
				env, err = envutil.Setenv(env, "JAZZER_TIMEOUT", value)
				if err != nil {
					return nil, err
				}
			}
			continue
		}

		// Check if any flag in the engine args has already been set in the
		// .jazzerjsrc. If not, we add it to the fuzzerOptions slice
		flagAlreadyInRC := false
		for _, opt := range rc.FuzzerOptions {
			rcFlag, _, found := strings.Cut(opt, "=")
			if !found {
				continue
			}

			if rcFlag == flag {
				flagAlreadyInRC = true
				break
			}
		}

		if !flagAlreadyInRC {
			fuzzerOptions = append(fuzzerOptions, arg)
		}
	}

	// If no new flag has been added, we don't need to set the environment variable
	// because Jazzer.js will take the values from the .jazzerjsrc automatically
	if sliceutil.Equal(fuzzerOptions, rc.FuzzerOptions) {
		return env, nil
	}

	// The values for JAZZER_FUZZER_OPTIONS have to be in form of a json array
	// e.g. ["-max_len=8192", "-seed=1"]
	fuzzerOptionAsJSONArray, err := json.Marshal(fuzzerOptions)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	env, err = envutil.Setenv(env, "JAZZER_FUZZER_OPTIONS", string(fuzzerOptionAsJSONArray))
	if err != nil {
		return nil, err
	}

	return env, nil
}
