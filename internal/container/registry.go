package container

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types/registry"
	"github.com/pkg/errors"

	"code-intelligence.com/cifuzz/internal/api"
)

// UserRegistryConfig returns the registry config for the given registry.
// It uses the docker config file to get the credentials, so users need to
// have logged in to the registry using `docker login`.
func UserRegistryConfig(reg string) (*api.RegistryConfig, error) {
	// TODO check if this works on Windows
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cfg, err := config.Load(filepath.Join(homedir, ".docker"))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// strip the repo name from the registry
	reg = strings.Split(reg, "/")[0]
	auth, err := cfg.GetAuthConfig(reg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ac := registry.AuthConfig(auth)

	// convert types.AuthConfig to types.registry.AuthConfig
	return &api.RegistryConfig{
		URL:  reg,
		Auth: &ac,
	}, nil

}
