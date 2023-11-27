package gocmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"code-intelligence.com/cifuzz/internal/api"
	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/internal/config"
)

type options struct {
	Dir         string
	BuildSystem string
	Interactive bool   `mapstructure:"interactive"`
	Server      string `mapstructure:"server"`
	Project     string `mapstructure:"project"`
	testLang    string
}

func New() *cobra.Command {
	var bindFlags func()
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "go",
		Short: "Find good candidates and create fuzz tests for them.",
		Long:  "The cifuzz go command will find good candidates and creates executable fuzz tests for them.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Bind viper keys to flags. We can't do this in the New
			// function, because that would re-bind viper keys which
			// were bound to the flags of other commands before.
			bindFlags()
			opts.Interactive = viper.GetBool("interactive")

			var err error
			if opts.Dir == "" {
				opts.Dir, err = os.Getwd()
				if err != nil {
					return errors.WithStack(err)
				}
			}

			// TODO: maybe this is already handled by dlth? If yes, skip the
			// BuildSystem checks. Detect and validate buildSystem only when testLang
			// is not specified by the user.
			opts.BuildSystem, err = config.DetermineBuildSystem(opts.Dir)
			if err != nil {
				return err
			}

			err = config.ValidateBuildSystem(opts.BuildSystem)
			if err != nil {
				return err
			}

			if opts.Interactive {
				opts.Interactive = term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
			}

			if !opts.Interactive && opts.BuildSystem == config.BuildSystemNodeJS && opts.testLang == "" {
				err := errors.New("cifuzz init requires a test language for Node.js projects [js|ts]")
				return cmdutils.WrapIncorrectUsageError(err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Project = viper.GetString("project")
			opts.Server = viper.GetString("server")

			var err error
			opts.Server, err = api.ValidateAndNormalizeServerURL(opts.Server)
			if err != nil {
				return err
			}
			return run(opts)
		},
	}

	// Disable config check for this command, because `cifuzz go` should start
	// from a clean project.
	cmdutils.DisableConfigCheck(cmd)

	// Note: If a flag should be configurable via viper as well (i.e.
	//       via cifuzz.yaml and CIFUZZ_* environment variables), bind
	//       it to viper in the PreRun function.
	bindFlags = cmdutils.AddFlags(cmd,
		cmdutils.AddInteractiveFlag,
		cmdutils.AddProjectFlag,
		cmdutils.AddServerFlag,
	)
	return cmd
}

func run(opts *options) error {

	fmt.Println("TODO: implement me")

	fmt.Println("cifuzz go running for build system", opts.BuildSystem)

	// call init on project
	// ...

	// use dlth to generate candidates for project
	// ...

	// create fuzz test for selected candidate
	// ...

	// run fuzz test
	// ...

	return nil
}
