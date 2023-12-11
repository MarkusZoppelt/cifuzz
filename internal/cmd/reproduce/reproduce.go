package reproduce

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/api"
	"code-intelligence.com/cifuzz/internal/build"
	"code-intelligence.com/cifuzz/internal/build/other"
	"code-intelligence.com/cifuzz/internal/cmd/run/adapter"
	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/internal/cmdutils/auth"
	"code-intelligence.com/cifuzz/internal/cmdutils/logging"
	"code-intelligence.com/cifuzz/internal/completion"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/dialog"
	findingPkg "code-intelligence.com/cifuzz/pkg/finding"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runner"
	"code-intelligence.com/cifuzz/util/envutil"
)

type options struct {
	ProjectDir   string `mapstructure:"project-dir"`
	ConfigDir    string `mapstructure:"config-dir"`
	Interactive  bool   `mapstructure:"interactive"`
	Server       string `mapstructure:"server"`
	Project      string `mapstructure:"project"`
	BuildSystem  string `mapstructure:"build-system"`
	BuildCommand string `mapstructure:"build-command"`
	CleanCommand string `mapstructure:"clean-command"`

	FindingName string

	buildStdout io.Writer
	buildStderr io.Writer
}

func (opts *options) validate() error {
	var err error

	if opts.BuildSystem == "" {
		opts.BuildSystem, err = config.DetermineBuildSystem(opts.ProjectDir)
		if err != nil {
			return err
		}
	}

	err = config.ValidateBuildSystem(opts.BuildSystem)
	if err != nil {
		return err
	}

	// To build with other build systems, a build command must be provided
	if opts.BuildSystem == config.BuildSystemOther && opts.BuildCommand == "" {
		msg := "Flag \"build-command\" must be set when using build system type \"other\""
		return cmdutils.WrapIncorrectUsageError(errors.New(msg))
	}

	return nil
}

type reproduceCmd struct {
	*cobra.Command
	opts *options
}

func New() *cobra.Command {
	return newWithOptions(&options{})
}

func newWithOptions(opts *options) *cobra.Command {
	var bindFlags func()

	cmd := &cobra.Command{
		Use:   "reproduce <name>",
		Short: "Run a fuzz test with the reproducing input of a finding",
		Long: `This command reproduces local or remote findings from CI Sense.

<name> is the name of a finding.
Run 'cifuzz findings' to get a list of all available findings.

In case of remote findings, this command needs a token to access the API of the
remote fuzzing server. You can specify this token via the CIFUZZ_API_TOKEN
environment variable or by running 'cifuzz login' first.
Remote finding data is downloaded and stored in the local project.

Note that only other build systems are supported for now.
`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.ValidFindings,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Bind viper keys to flags. We can't do this in the New
			// function, because that would re-bind viper keys which
			// were bound to the flags of other commands before.
			bindFlags()
			err := config.FindAndParseProjectConfig(opts)
			if err != nil {
				return err
			}
			opts.FindingName = args[0]
			opts.buildStdout = cmd.OutOrStdout()
			opts.buildStderr = cmd.OutOrStderr()

			return opts.validate()
		},
		RunE: func(c *cobra.Command, args []string) error {
			var err error
			opts.Server, err = api.ValidateAndNormalizeServerURL(opts.Server)
			if err != nil {
				return err
			}
			cmd := reproduceCmd{Command: c, opts: opts}
			return cmd.run()
		},
	}

	// Note: If a flag should be configurable via viper as well (i.e.
	//       via cifuzz.yaml and CIFUZZ_* environment variables), bind
	//       it to viper in the PreRun function.
	bindFlags = cmdutils.AddFlags(cmd,
		cmdutils.AddProjectDirFlag,
		cmdutils.AddInteractiveFlag,
		cmdutils.AddServerFlag,
		cmdutils.AddProjectFlag,
		cmdutils.AddBuildCommandFlag,
		cmdutils.AddCleanCommandFlag,
		cmdutils.AddBuildJobsFlag,
	)

	return cmd
}

func (c *reproduceCmd) run() error {
	adapter, err := adapter.NewAdapter(c.opts.BuildSystem)
	if err != nil {
		return err
	}
	err = adapter.CheckDependencies(c.opts.ProjectDir)
	if err != nil {
		return err
	}

	finding, err := c.loadLocalOrRemoteFinding()
	if err != nil {
		return err
	}

	if c.opts.BuildSystem == config.BuildSystemOther {
		cBuildResult, err := c.wrapBuild(finding.FuzzTest, c.buildOther)
		if err != nil {
			return err
		}

		var env []string
		env, err = setSanitizerOptions(env)
		if err != nil {
			return err
		}

		// execute fuzz test binary with input file from finding
		cmd := exec.Command(cBuildResult.Executable, finding.InputFile)
		cmd.Dir = c.opts.ProjectDir
		cmd.Stdout = c.OutOrStdout()
		cmd.Stderr = c.OutOrStdout()
		cmd.Env, err = envutil.Copy(os.Environ(), env)
		if err != nil {
			return err
		}
		log.Printf("Command: %s", envutil.QuotedCommandWithEnv(cmd.Args, env))
		_ = cmd.Run()
	} else {
		return errors.New("Only other build systems are supported for now.")
	}

	return nil
}

func (c *reproduceCmd) wrapBuild(fuzzTest string, build func(string) (*build.CBuildResult, error)) (*build.CBuildResult, error) {
	var err error
	if logging.ShouldLogBuildToFile() {
		c.opts.buildStdout, err = logging.BuildOutputToFile(c.opts.ProjectDir, []string{fuzzTest})
		if err != nil {
			return nil, err
		}
		c.opts.buildStderr = c.opts.buildStdout
	}

	buildPrinter := logging.NewBuildPrinter(os.Stdout, log.BuildInProgressMsg)
	log.Infof("Building %s", pterm.Style{pterm.Reset, pterm.FgLightBlue}.Sprint(fuzzTest))
	cBuildResult, err := build(fuzzTest)
	if err != nil {
		buildPrinter.StopOnError(log.BuildInProgressErrorMsg)
		return nil, err
	}
	buildPrinter.StopOnSuccess(log.BuildInProgressSuccessMsg, true)

	return cBuildResult, nil
}

func (c *reproduceCmd) buildOther(fuzzTest string) (*build.CBuildResult, error) {
	builder, err := other.NewBuilder(&other.BuilderOptions{
		ProjectDir:   c.opts.ProjectDir,
		BuildCommand: c.opts.BuildCommand,
		CleanCommand: c.opts.CleanCommand,
		Sanitizers:   []string{"address", "undefined"},
		Stdout:       c.opts.buildStdout,
		Stderr:       c.opts.buildStderr,
	})
	if err != nil {
		return nil, err
	}

	err = builder.Clean()
	if err != nil {
		return nil, err
	}

	cBuildResult, err := builder.Build(fuzzTest)
	if err != nil {
		return nil, err
	}

	return cBuildResult, nil
}

// loadLocalOrRemoteFinding loads the finding if it exists locally, otherwise
// it tries to download it from CI Sense
func (c *reproduceCmd) loadLocalOrRemoteFinding() (*findingPkg.Finding, error) {
	finding, err := findingPkg.LoadFinding(c.opts.ProjectDir, c.opts.FindingName)
	if err != nil && !findingPkg.IsNotExistError(err) {
		return nil, err
	}

	if finding == nil {
		finding, err = c.downloadRemoteFinding()
		if err != nil {
			return nil, err
		}
		// save remote finding with input data
		err = finding.Save(c.opts.ProjectDir)
		if err != nil {
			return nil, err
		}
	}

	return finding, nil
}

func (c *reproduceCmd) downloadRemoteFinding() (*findingPkg.Finding, error) {
	token, err := auth.GetValidToken(c.opts.Server)
	if err != nil {
		return nil, errors.New("No valid token found.")
	}

	log.Success("You are authenticated.")

	apiClient := api.NewClient(c.opts.Server)
	// get remote findings if project is set and user is authenticated
	if c.opts.Project != "" {
		return apiClient.GetRemoteFinding(c.opts.FindingName, c.opts.Project, token)
	}

	if c.opts.Interactive { // let the user select a project
		remoteProjects, err := apiClient.ListProjects(token)
		if err != nil {
			return nil, err
		}
		c.opts.Project, err = dialog.ProjectPicker(remoteProjects, "Select a remote project:")
		if err != nil {
			return nil, err
		}
		if c.opts.Project != "<<cancel>>" {
			err = dialog.AskToPersistProjectChoice(c.opts.Project)
			if err != nil {
				return nil, err
			}

			return apiClient.GetRemoteFinding(c.opts.FindingName, c.opts.Project, token)
		}
	}

	return nil, errors.New(fmt.Sprintf("%s not found in CI Sense project: %s", c.opts.FindingName, c.opts.Project))
}

func setSanitizerOptions(env []string) ([]string, error) {
	var err error
	if os.Getenv("ASAN_OPTIONS") != "" {
		env, err = envutil.Setenv(env, "ASAN_OPTIONS", os.Getenv("ASAN_OPTIONS"))
		if err != nil {
			return nil, err
		}
	}
	if os.Getenv("UBSAN_OPTIONS") != "" {
		env, err = envutil.Setenv(env, "UBSAN_OPTIONS", os.Getenv("UBSAN_OPTIONS"))
		if err != nil {
			return nil, err
		}
	}

	env, err = runner.SetCommonUBSANOptions(env)
	if err != nil {
		return nil, err
	}

	env, err = runner.SetCommonASANOptions(env)
	if err != nil {
		return nil, err
	}

	return env, nil
}
