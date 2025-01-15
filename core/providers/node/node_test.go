package node

import (
	"testing"

	"github.com/railwayapp/railpack-go/core/app"
	"github.com/railwayapp/railpack-go/core/generate"
	"github.com/stretchr/testify/require"
)

func createGenerateContext(t *testing.T, path string) *generate.GenerateContext {
	userApp, err := app.NewApp(path)
	if err != nil {
		t.Fatalf("error creating app: %v", err)
	}

	env := app.NewEnvironment(nil)

	ctx, err := generate.NewGenerateContext(userApp, env)
	if err != nil {
		t.Fatalf("error creating generate context: %v", err)
	}

	return ctx
}

func TestPackageManager(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		packageManager PackageManager
	}{
		{
			name:           "npm project",
			path:           "../../../examples/node-npm-latest",
			packageManager: PackageManagerNpm,
		},
		{
			name:           "bun project",
			path:           "../../../examples/node-bun",
			packageManager: PackageManagerBun,
		},
		{
			name:           "pnpm project",
			path:           "../../../examples/node-corepack",
			packageManager: PackageManagerPnpm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createGenerateContext(t, tt.path)
			provider := NodeProvider{}

			packageManager := provider.getPackageManager(ctx.App)
			require.Equal(t, tt.packageManager, packageManager)
		})
	}
}

func TestNodeCorepack(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantCorepack bool
	}{
		{
			name:         "corepack project",
			path:         "../../../examples/node-corepack",
			wantCorepack: true,
		},
		{
			name:         "bun project",
			path:         "../../../examples/node-bun",
			wantCorepack: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createGenerateContext(t, tt.path)
			provider := NodeProvider{}

			packageJson, err := provider.getPackageJson(ctx.App)
			require.NoError(t, err)

			usesCorepack := provider.usesCorepack(packageJson)
			require.Equal(t, tt.wantCorepack, usesCorepack)
		})
	}
}
