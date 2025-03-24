package buildkit

import (
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
)

type RegistryOptions struct {
	UseRegistryExport bool
	RegistryURL       string
	RegistryUser      string
	RegistryPassword  string
	RegistryPush      bool
	CompressionType   string
	CompressionLevel  string
}

func createAuthProvider(registryURL, username, password string) session.Attachable {
	// Create a new config file
	configFile := configfile.New("")

	// Add the auth entry for the registry
	configFile.AuthConfigs = map[string]types.AuthConfig{
		registryURL: {
			Username: username,
			Password: password,
			Auth:     "",
		},
	}

	// Create the auth provider configuration
	cfg := authprovider.DockerAuthProviderConfig{
		ConfigFile: configFile,
	}

	return authprovider.NewDockerAuthProvider(cfg)
}
