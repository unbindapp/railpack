package node

import (
	"testing"

	testingUtils "github.com/unbindapp/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestWorkspace(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		hasWorkspaces bool
		numPackages   int
	}{
		{
			name:          "npm workspaces",
			path:          "../../../examples/node-npm-workspaces",
			hasWorkspaces: true,
			numPackages:   2,
		},
		{
			name:          "pnpm workspaces",
			path:          "../../../examples/node-pnpm-workspaces",
			hasWorkspaces: true,
			numPackages:   2,
		},
		{
			name:          "no workspaces",
			path:          "../../../examples/node-npm",
			hasWorkspaces: false,
			numPackages:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			workspace, err := NewWorkspace(ctx.App)
			require.NoError(t, err)

			require.Equal(t, tt.hasWorkspaces, workspace.HasWorkspaces())
			require.Equal(t, tt.numPackages, len(workspace.Packages))

			if tt.hasWorkspaces {
				pkgA := workspace.GetPackage("packages/pkg-a")
				require.NotNil(t, pkgA)
				require.Equal(t, "pkg-a", pkgA.PackageJson.Name)

				pkgB := workspace.GetPackage("packages/pkg-b")
				require.NotNil(t, pkgB)
				require.Equal(t, "pkg-b", pkgB.PackageJson.Name)
			}
		})
	}
}

func TestConvertWorkspacePattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "single level pattern",
			pattern:  "packages/*",
			expected: "packages/*/package.json",
		},
		{
			name:     "recursive pattern",
			pattern:  "packages/**",
			expected: "packages/**/package.json",
		},
		{
			name:     "direct path",
			pattern:  "packages/foo",
			expected: "packages/foo/package.json",
		},
		{
			name:     "already has package.json",
			pattern:  "packages/foo/package.json",
			expected: "packages/foo/package.json/package.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertWorkspacePattern(tt.pattern)
			require.Equal(t, tt.expected, result)
		})
	}
}
