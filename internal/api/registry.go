package api

import (
	"github.com/docker/docker/api/types/registry"
)

type RegistryConfig struct {
	URL  string               `json:"registry_url"`
	Auth *registry.AuthConfig `json:"x_registry_auth"`
}
