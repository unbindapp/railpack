package buildkit

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/railwayapp/railpack-go/core/plan"
	p "github.com/railwayapp/railpack-go/core/plan"
)

type ConvertPlanOptions struct {
	BuildPlatform BuildPlatform
}

func ConvertPlanToLLB(plan *p.BuildPlan, opts ConvertPlanOptions) (*llb.State, *Image, error) {
	platform := opts.BuildPlatform.ToPlatform()

	state := llb.Image("ubuntu:noble",
		llb.Platform(platform),
	)

	// Set working directory
	state = state.Dir("/app")

	// Add all variables as environment variables
	for name, value := range plan.Variables {
		state = state.AddEnv(name, value)
	}

	graph, err := NewBuildGraph(plan, &state)
	if err != nil {
		return nil, nil, err
	}

	graphState, err := graph.GenerateLLB()
	if err != nil {
		return nil, nil, err
	}

	state = *graphState

	pathList := []string{}
	for _, step := range plan.Steps {
		for _, cmd := range step.Commands {
			if pathCmd, ok := cmd.(p.PathCommand); ok {
				pathList = append(pathList, pathCmd.Path)
			}
		}
	}

	fmt.Printf("pathList: %+v\n", pathList)

	image := Image{
		Image: specs.Image{
			Platform: specs.Platform{
				OS:           platform.OS,
				Architecture: platform.Architecture,
			},
		},
		Variant: platform.Variant,
		Config: specs.ImageConfig{
			Env: []string{
				"PATH=/mise/shims:" + system.DefaultPathEnvUnix,
			},
			WorkingDir: "/app",
		},
	}

	return &state, &image, nil
}

func convertStepToLLB(step *plan.Step, baseState *llb.State) (*llb.State, error) {
	state := baseState

	for _, cmd := range step.Commands {
		var err error
		state, err = convertCommandToLLB(cmd, state, step)
		if err != nil {
			return nil, err
		}
	}

	return state, nil
}
