package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	findingPkg "code-intelligence.com/cifuzz/pkg/finding"
	"code-intelligence.com/cifuzz/pkg/parser/libfuzzer/stacktrace"
)

type Findings struct {
	Findings []Finding `json:"findings"`
	Links    []Link    `json:"links,omitempty"`
}

type Finding struct {
	Name                  string       `json:"name"`
	DisplayName           string       `json:"display_name"`
	FuzzTarget            string       `json:"fuzz_target"`
	FuzzingRun            string       `json:"fuzzing_run"`
	CampaignRun           string       `json:"campaign_run"`
	ErrorReport           *ErrorReport `json:"error_report"`
	Timestamp             string       `json:"timestamp"`
	FuzzTargetDisplayName string       `json:"fuzz_target_display_name,omitempty"`

	// new with v3
	JobNid           string       `json:"job_nid,omitempty"`
	Nid              string       `json:"nid,omitempty"`
	InputData        string       `json:"input_data,omitempty"`
	RunNid           string       `json:"run_nid,omitempty"`
	ErrorID          string       `json:"error_id,omitempty"`
	Logs             []string     `json:"logs,omitempty"`
	State            string       `json:"state,omitempty"`
	CreatedAt        string       `json:"created_at,omitempty"`
	FirstSeenFinding string       `json:"first_seen_finding,omitempty"`
	IssueTrackerLink string       `json:"issue_tracker_link,omitempty"`
	ProjectNid       string       `json:"project_nid,omitempty"`
	Stacktrace       []Stacktrace `json:"stacktrace,omitempty"`
}

type Stacktrace struct {
	File     string `json:"file"`
	Function string `json:"function"`
	Line     int64  `json:"line"`
	Column   int64  `json:"column"`
}

type ErrorReport struct {
	Logs      []string `json:"logs"`
	Details   string   `json:"details"`
	Type      string   `json:"type,omitempty"`
	InputData []byte   `json:"input_data,omitempty"`

	DebuggingInfo      *DebuggingInfo           `json:"debugging_info,omitempty"`
	HumanReadableInput string                   `json:"human_readable_input,omitempty"`
	MoreDetails        *findingPkg.ErrorDetails `json:"more_details,omitempty"`
	Tag                string                   `json:"tag,omitempty"`
	ShortDescription   string                   `json:"short_description,omitempty"`
}

type DebuggingInfo struct {
	ExecutablePath string         `json:"executable_path,omitempty"`
	RunArguments   []string       `json:"run_arguments,omitempty"`
	BreakPoints    []*BreakPoint  `json:"break_points,omitempty"`
	Environment    []*Environment `json:"environment,omitempty"`
}

type BreakPoint struct {
	SourceFilePath string           `json:"source_file_path,omitempty"`
	Location       *FindingLocation `json:"location,omitempty"`
	Function       string           `json:"function,omitempty"`
}

type FindingLocation struct {
	Line   uint32 `json:"line,omitempty"`
	Column uint32 `json:"column,omitempty"`
}

type Environment struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type Severity struct {
	Description string  `json:"description,omitempty"`
	Score       float32 `json:"score,omitempty"`
}

// DownloadRemoteFindings downloads all remote findings for a given project from CI Sense.
func (client *APIClient) DownloadRemoteFindings(project string, token string) (*Findings, error) {
	project = ConvertProjectNameForUseWithAPIV1V2(project)

	return APIRequest[Findings](&RequestConfig{
		Client:       client,
		Method:       "GET",
		Token:        token,
		PathSegments: []string{"v1", project, "findings"},
		// setting a timeout of 5 seconds for the request, since we don't want to
		// wait too long, especially when we need to await this request for command
		// completion
		Timeout: 5 * time.Second,
	})
}

// RemoteFindingsForRun uses the v3 API to download all findings for a given
// (container remote-)run.
func (client *APIClient) RemoteFindingsForRun(runNID string, token string) (*Findings, error) {
	return APIRequest[Findings](&RequestConfig{
		Client:       client,
		Method:       "GET",
		Token:        token,
		PathSegments: []string{"v3", "runs", runNID, "findings"},
	})
}

func (client *APIClient) UploadFinding(project string, fuzzTarget string, campaignRunName string, fuzzingRunName string, finding *findingPkg.Finding, token string) error {
	project = ConvertProjectNameForUseWithAPIV1V2(project)

	// loop through the stack trace and create a list of breakpoints
	breakPoints := []*BreakPoint{}
	for _, stackFrame := range finding.StackTrace {
		breakPoints = append(breakPoints, &BreakPoint{
			SourceFilePath: stackFrame.SourceFile,
			Location: &FindingLocation{
				Line:   stackFrame.Line,
				Column: stackFrame.Column,
			},
			Function: stackFrame.Function,
		})
	}

	findings := &Findings{
		Findings: []Finding{
			{
				Name:        project + finding.Name,
				DisplayName: finding.Name,
				FuzzTarget:  fuzzTarget,
				FuzzingRun:  fuzzingRunName,
				CampaignRun: campaignRunName,
				ErrorReport: &ErrorReport{
					Logs:      finding.Logs,
					Details:   finding.Details,
					Type:      string(finding.Type),
					InputData: finding.InputData,
					DebuggingInfo: &DebuggingInfo{
						BreakPoints: breakPoints,
					},
					MoreDetails:      finding.MoreDetails,
					Tag:              finding.Tag,
					ShortDescription: finding.ShortDescriptionColumns()[0],
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	body, err := json.MarshalIndent(findings, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = APIRequest[map[string]json.RawMessage](&RequestConfig{
		Client:       client,
		Method:       "POST",
		Body:         body,
		Token:        token,
		PathSegments: []string{"v1", project, "findings"},
	})
	if err != nil {
		return err
	}

	return nil
}

func (client *APIClient) GetRemoteFinding(findingName string, project string, token string) (*findingPkg.Finding, error) {
	findings, err := client.DownloadRemoteFindings(project, token)
	if err != nil {
		return nil, err
	}

	var remoteFinding *Finding
	for i := range findings.Findings {
		finding := &findings.Findings[i]
		if finding.DisplayName == findingName {
			// if a finding with the same name was already found,
			// we want to use the one with the latest timestamp
			if remoteFinding != nil {
				currentTimestamp, err := time.Parse(time.RFC3339, remoteFinding.Timestamp)
				if err != nil {
					continue
				}
				timestamp, err := time.Parse(time.RFC3339, finding.Timestamp)
				if err != nil {
					continue
				}
				if timestamp.After(currentTimestamp) {
					remoteFinding = finding
				}
			} else {
				remoteFinding = finding
			}
		}
	}

	if remoteFinding != nil {
		return RemoteToLocalFinding(remoteFinding)
	}

	return nil, errors.New(fmt.Sprintf("%s not found in CI Sense project: %s", findingName, project))
}

// RemoteToLocalFinding converts the api response finding to a local finding
func RemoteToLocalFinding(finding *Finding) (*findingPkg.Finding, error) {
	timeStamp, err := time.Parse(time.RFC3339, finding.Timestamp)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not parse timestamp %s", finding.Timestamp)
	}
	localFinding := &findingPkg.Finding{
		Origin:             "CI Sense",
		Name:               finding.DisplayName,
		Type:               findingPkg.ErrorType(finding.ErrorReport.Type),
		InputData:          finding.ErrorReport.InputData,
		Logs:               finding.ErrorReport.Logs,
		Details:            finding.ErrorReport.Details,
		HumanReadableInput: string(finding.ErrorReport.InputData),
		MoreDetails:        finding.ErrorReport.MoreDetails,
		Tag:                finding.ErrorReport.Tag,
		CreatedAt:          timeStamp,
		FuzzTest:           finding.FuzzTargetDisplayName,
	}

	for _, breakPoint := range finding.ErrorReport.DebuggingInfo.BreakPoints {
		localFinding.StackTrace = append(localFinding.StackTrace, &stacktrace.StackFrame{
			Function:   breakPoint.Function,
			SourceFile: breakPoint.SourceFilePath,
			Line:       breakPoint.Location.Line,
			Column:     breakPoint.Location.Column,
		})
	}

	return localFinding, nil
}
