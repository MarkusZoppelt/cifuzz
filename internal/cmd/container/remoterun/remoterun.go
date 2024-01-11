package remoterun

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/api"
	"code-intelligence.com/cifuzz/internal/bundler"
	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/internal/cmdutils/auth"
	"code-intelligence.com/cifuzz/internal/cmdutils/logging"
	"code-intelligence.com/cifuzz/internal/cmdutils/resolve"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/internal/container"
	"code-intelligence.com/cifuzz/pkg/dialog"
	"code-intelligence.com/cifuzz/pkg/finding"
	"code-intelligence.com/cifuzz/pkg/log"
)

type containerRemoteRunOpts struct {
	bundler.Opts       `mapstructure:",squash"`
	Interactive        bool          `mapstructure:"interactive"`
	PrintJSON          bool          `mapstructure:"print-json"`
	Monitor            bool          `mapstructure:"monitor"`
	MonitorDuration    time.Duration `mapstructure:"monitor-duration"`
	MonitorInterval    time.Duration `mapstructure:"monitor-interval"`
	MinFindingSeverity string        `mapstructure:"min-finding-severity"`

	// CI Sense specific options
	Server  string `mapstructure:"server"`
	Project string `mapstructure:"project"`

	// can be set via cifuzz.yaml but is *NOT* added to the cifuzz.yaml.tmpl because we do not want to advertise this feature
	Registry string `mapstructure:"registry"`
}

type containerRemoteRunCmd struct {
	*cobra.Command
	opts      *containerRemoteRunOpts
	apiClient *api.APIClient
}

func New() *cobra.Command {
	return newWithOptions(&containerRemoteRunOpts{})
}

func newWithOptions(opts *containerRemoteRunOpts) *cobra.Command {
	var bindFlags func()

	cmd := &cobra.Command{
		Use:   "remote-run",
		Short: "Build and run a Fuzz Test container image on a CI server",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Bind viper keys to flags. We can't do this in the New
			// function, because that would re-bind viper keys which
			// were bound to the flags of other commands before.
			bindFlags()

			var argsToPass []string
			if cmd.ArgsLenAtDash() != -1 {
				argsToPass = args[cmd.ArgsLenAtDash():]
				args = args[:cmd.ArgsLenAtDash()]
			}

			err := config.FindAndParseProjectConfig(opts)
			if err != nil {
				return err
			}

			fuzzTests, err := resolve.FuzzTestArguments(opts.ResolveSourceFilePath, args, opts.BuildSystem, opts.ProjectDir)
			if err != nil {
				return err
			}
			opts.FuzzTests = fuzzTests
			opts.BuildSystemArgs = argsToPass

			return opts.Validate()
		},
		RunE: func(c *cobra.Command, args []string) error {
			var err error
			opts.Server, err = api.ValidateAndNormalizeServerURL(opts.Server)
			if err != nil {
				return err
			}

			cmd := &containerRemoteRunCmd{Command: c, opts: opts}
			cmd.apiClient = api.NewClient(opts.Server)
			return cmd.run()
		},
	}
	bindFlags = cmdutils.AddFlags(cmd,
		cmdutils.AddAdditionalFilesFlag,
		cmdutils.AddBranchFlag,
		cmdutils.AddBuildCommandFlag,
		cmdutils.AddCleanCommandFlag,
		cmdutils.AddBuildJobsFlag,
		cmdutils.AddCommitFlag,
		cmdutils.AddDictFlag,
		cmdutils.AddDockerImageFlagForContainerCommand,
		cmdutils.AddEngineArgFlag,
		cmdutils.AddEnvFlag,
		cmdutils.AddMinFindingSeverityFlag,
		cmdutils.AddInteractiveFlag,
		cmdutils.AddMonitorFlag,
		cmdutils.AddMonitorDurationFlag,
		cmdutils.AddMonitorIntervalFlag,
		cmdutils.AddPrintJSONFlag,
		cmdutils.AddProjectDirFlag,
		cmdutils.AddProjectFlag,
		cmdutils.AddRegistryFlag,
		cmdutils.AddSeedCorpusFlag,
		cmdutils.AddServerFlag,
		cmdutils.AddTimeoutFlag,
		cmdutils.AddResolveSourceFileFlag,
	)

	return cmd
}

func (c *containerRemoteRunCmd) run() error {
	var err error

	token, err := auth.EnsureValidToken(c.opts.Server)
	if err != nil {
		var connectionError *api.ConnectionError
		if errors.As(err, &connectionError) {
			return errors.New("No connection to CI Sense")
		}
		return err
	}

	if c.opts.Project == "" {
		projects, err := c.apiClient.ListProjects(token)
		if err != nil {
			log.Error(err)
			err = errors.New("Flag \"project\" must be set")
			return cmdutils.WrapIncorrectUsageError(err)
		}

		if c.opts.Interactive {
			c.opts.Project, err = dialog.ProjectPickerWithOptionNew(projects, "Select the project you want to start a fuzzing run for", c.apiClient, token)
			if err != nil {
				return err
			}

			if c.opts.Project == "<<cancel>>" {
				log.Info("Container remote run cancelled.")
				return nil
			}

			// this will ask users via a y/N prompt if they want to persist the
			// project choice
			err = dialog.AskToPersistProjectChoice(c.opts.Project)
			if err != nil {
				return err
			}
		} else {
			err = errors.New("Flag 'project' must be set.")
			return cmdutils.WrapIncorrectUsageError(err)
		}
	}

	buildPrinter := logging.NewBuildPrinter(c.ErrOrStderr(), log.ContainerBuildInProgressMsg)
	imageID, err := c.buildImage()
	if err != nil {
		buildPrinter.StopOnError(log.ContainerBuildInProgressErrorMsg)
		return err
	}

	buildPrinter.StopOnSuccess(log.ContainerBuildInProgressSuccessMsg, false)

	var registryCredentials *api.RegistryConfig

	// if the user has set the registry flag, we don't use the API to get the
	// registry but instead use the local docker config
	if c.opts.Registry != "" {
		log.Debugf("Getting registry credentials from local docker config for registry %s", c.opts.Registry)
		registryCredentials, err = container.UserRegistryConfig(c.opts.Registry)
		if err != nil {
			return err
		}
	} else {
		log.Debug("Getting registry credentials from CI Sense")
		registryCredentials, err = api.APIRequest[api.RegistryConfig](&api.RequestConfig{
			Client:       c.apiClient,
			Method:       "GET",
			Token:        token,
			PathSegments: []string{"v2", "docker_registry", "authentication"},
		})
		if err != nil {
			return err
		}
	}

	// we'll create a sha256 of the project name in the image name so that we
	// only get lowercase characters in the image name.
	projectHash := fmt.Sprintf("%x", sha256.Sum256([]byte(c.opts.Project)))
	// We'll use only a portion (12 characters) of the hash to keep the image
	// name reasonably short, similar to how Git uses only a portion (the last 7
	// chars) of the hash for commit IDs. SHA-1 is weaker than SHA-256, which is
	// why Git uses a longer portion for a similar level of security. However,
	// given the increased strength of SHA-256, using a longer portion, such as
	// 12 characters, might be a reasonable compromise.
	reducedHash := projectHash[len(projectHash)-12:]
	imageName := fmt.Sprintf("%s/cifuzz/%s", registryCredentials.URL, reducedHash)

	err = container.UploadImage(imageID, registryCredentials, imageName)
	if err != nil {
		return err
	}

	imageID = fmt.Sprintf("%s:%s", imageName, imageID)
	response, err := c.apiClient.PostContainerRemoteRun(imageID, c.opts.Project, c.opts.FuzzTests, token)
	if err != nil {
		return err
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	if c.opts.PrintJSON {
		_, _ = fmt.Fprintln(os.Stdout, string(responseJSON))
	}

	addr, err := cmdutils.BuildURLFromParts(c.opts.Server, "app", "projects", c.opts.Project, "runs")
	if err != nil {
		return err
	}

	log.Successf(`Successfully started fuzzing run. To view findings and coverage, open:
    %s`, addr)

	// if --monitor is set by user, we want to monitor the run
	if c.opts.Monitor {
		err = c.monitorCampaignRun(c.apiClient, response.Run.Nid, token)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *containerRemoteRunCmd) buildImage() (string, error) {
	b := bundler.New(&c.opts.Opts)
	bundlePath, err := b.Bundle()
	if err != nil {
		return "", err
	}

	return container.BuildImageFromBundle(bundlePath)
}

// monitorCampaignRun monitors the status of a campaign run on the CI Sense
// API. It returns when the run is finished, when it times out, or when a new
// finding is reported.
func (c *containerRemoteRunCmd) monitorCampaignRun(apiClient *api.APIClient, runNID string, token string) error {
	if c.opts.MonitorDuration > 0 {
		log.Infof("Max monitor duration is %.0f seconds.", c.opts.MonitorDuration.Seconds())
	}
	if c.opts.MinFindingSeverity != "" {
		log.Infof("Monitoring for findings of severity %s or higher.", c.opts.MinFindingSeverity)
	}
	log.Info("Monitoring will automatically stop when the run finishes, times out, or a finding is reported.")

	status, err := apiClient.GetContainerRemoteRunStatus(runNID, token)
	if err != nil {
		return err
	}

	if status.Run.Status == "finished" || status.Run.Status == "SUCCEEDED" {
		log.Successf("Run finished!")
		return nil
	}

	// if the monitor duration is set, we want to stop monitoring after the
	// duration has passed. If the duration is less than the pull interval, we
	// need to pull every second to make sure we don't miss the end of the run.
	var ticker *time.Ticker

	runFor := c.opts.MonitorDuration
	if runFor < c.opts.MonitorInterval {
		ticker = time.NewTicker(1 * time.Second)
	} else {
		ticker = time.NewTicker(c.opts.MonitorInterval)
	}
	defer ticker.Stop()
	stopChannel := make(chan struct{})

	// If the duration has passed, we want to stop monitoring.
	if runFor != 0 {
		time.AfterFunc(runFor, func() { close(stopChannel) })
	}

	for {
		select {
		case <-ticker.C:
			status, err := apiClient.GetContainerRemoteRunStatus(runNID, token)
			if err != nil {
				return err
			}

			findings, err := apiClient.RemoteFindingsForRun(runNID, token)
			if err != nil {
				return err
			}

			if c.opts.MinFindingSeverity != "" {
				var filteredFindings []api.Finding
				for idx := range findings.Findings {
					f := findings.Findings[idx]
					severity, err := finding.SeverityForErrorID(f.ErrorID)
					if err != nil {
						return err
					}

					var level string

					// we need to make sure that the mapping from error ID to severity
					// worked, otherwise we'll get a nil pointer dereference to severity
					if severity != nil {
						level = strings.ToUpper(string(severity.Level))
					} else {
						// if the mapping didn't work, we'll use MEDIUM as the default
						level = "MEDIUM"
						log.Warnf("Finding with unknown severity found (using MEDIUM): %s, NID: %s", f.DisplayName, f.Nid)
					}

					switch strings.ToUpper(c.opts.MinFindingSeverity) {
					case "LOW":
						if level == "LOW" || level == "MEDIUM" || level == "HIGH" || level == "CRITICAL" {
							filteredFindings = append(filteredFindings, f)
						}
					case "MEDIUM":
						if level == "MEDIUM" || level == "HIGH" || level == "CRITICAL" {
							filteredFindings = append(filteredFindings, f)
						}
					case "HIGH":
						if level == "HIGH" || level == "CRITICAL" {
							filteredFindings = append(filteredFindings, f)
						}
					case "CRITICAL":
						if level == "CRITICAL" {
							filteredFindings = append(filteredFindings, f)
						}
					default:
						filteredFindings = append(filteredFindings, f)
					}

				}
				findings.Findings = filteredFindings
			}

			if len(findings.Findings) > 0 {
				for idx := range findings.Findings {
					finding := findings.Findings[idx]
					log.Successf("Finding found: %s, NID: %s", finding.DisplayName, finding.Nid)
				}
				return nil
			}

			if status.Run.Status == "cancelled" {
				log.Warn("Run cancelled.")
				return nil
			}

			if status.Run.Status == "STOPPED" {
				// we can exit early if the campaign run has stopped before the
				// configuration duruation.
				close(stopChannel)
			}
		case <-stopChannel:
			log.Info("Run finished or timed out.")
			return nil
		}
	}
}
