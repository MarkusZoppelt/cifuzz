package adapter

import (
	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/cmd/run/reporthandler"
	"code-intelligence.com/cifuzz/internal/config"
)

type Adapter interface {
	CheckDependencies(string) error
	Run(*RunOptions) (*reporthandler.ReportHandler, error)
	Cleanup()
}

func NewAdapter(buildSystem string) (Adapter, error) {
	var adapter Adapter
	switch buildSystem {
	case config.BuildSystemCMake:
		adapter = &CMakeAdapter{}
	case config.BuildSystemMaven:
		adapter = &MavenAdapter{}
	case config.BuildSystemGradle:
		adapter = &GradleAdapter{}
	case config.BuildSystemNodeJS:
		adapter = &NodeJSAdapter{}
	case config.BuildSystemOther:
		adapter = &OtherAdapter{}
	case config.BuildSystemBazel:
		adapter = &BazelAdapter{}
	default:
		return nil, errors.Errorf("Unsupported build system \"%s\"", buildSystem)
	}
	return adapter, nil
}
