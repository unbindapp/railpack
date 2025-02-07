package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/railwayapp/railpack/core/app"
	"github.com/stretchr/testify/require"
)

func TestGenerateBuildPlanForExamples(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Get all the examples
	examplesDir := filepath.Join(filepath.Dir(wd), "examples")
	entries, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// For each example, generate a build plan that we can snapshot test
		t.Run(entry.Name(), func(t *testing.T) {
			examplePath := filepath.Join(examplesDir, entry.Name())

			userApp, err := app.NewApp(examplePath)
			require.NoError(t, err)

			env := app.NewEnvironment(nil)

			buildResult, err := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{})
			require.NoError(t, err)

			plan := buildResult.Plan

			// Remove the mise.toml asset since the versions may change between runs
			for _, step := range plan.Steps {
				for name := range step.Assets {
					if name == "mise.toml" {
						step.Assets[name] = "[mise.toml]"
					}
				}
			}

			snaps.MatchJSON(t, plan)
		})
	}
}
