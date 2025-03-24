package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/unbindapp/railpack/core/app"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	v := m.Run()
	snaps.Clean(m, snaps.CleanOpts{Sort: true})
	os.Exit(v)
}

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

			buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{})

			if !buildResult.Success {
				t.Fatalf("failed to generate build plan for %s: %s", entry.Name(), buildResult.Logs)
			}

			plan := buildResult.Plan

			// Remove the mise.toml asset since the versions may change between runs
			for _, step := range plan.Steps {
				for name := range step.Assets {
					if name == "mise.toml" {
						step.Assets[name] = "[mise.toml]"
					}
				}
			}

			snaps.MatchStandaloneJSON(t, plan)
		})
	}
}
