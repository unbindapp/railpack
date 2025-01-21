package buildkit

import (
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
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

	graphOutput, err := graph.GenerateLLB()
	if err != nil {
		return nil, nil, err
	}

	state = *graphOutput.State
	imageEnv := getImageEnv(graphOutput)

	image := Image{
		Image: specs.Image{
			Platform: specs.Platform{
				OS:           platform.OS,
				Architecture: platform.Architecture,
			},
		},
		Variant: platform.Variant,
		Config: specs.ImageConfig{
			Env:        imageEnv,
			WorkingDir: "/app",
		},
	}

	return &state, &image, nil
}

func getImageEnv(graphOutput *BuildGraphOutput) []string {
	pathString := strings.Join(graphOutput.PathList, ":")

	var pathEnv string
	if pathString == "" {
		pathEnv = "PATH=" + system.DefaultPathEnvUnix
	} else {
		pathEnv = "PATH=" + pathString + ":" + system.DefaultPathEnvUnix
	}

	imageEnv := []string{pathEnv}

	for k, v := range graphOutput.EnvVars {
		imageEnv = append(imageEnv, fmt.Sprintf("%s=%s", k, v))
	}

	return imageEnv
}
