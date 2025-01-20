package buildkit

import (
	"fmt"
	"path/filepath"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/railwayapp/railpack-go/core/plan"
)

type ConvertPlanOptions struct {
	BuildPlatform BuildPlatform
}

func ConvertPlanToLLB(plan *plan.BuildPlan, opts ConvertPlanOptions) (*llb.State, *Image, error) {
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

func convertCommandToLLB(cmd plan.Command, state *llb.State, step *plan.Step) (*llb.State, error) {
	switch cmd := cmd.(type) {
	case plan.ExecCommand:
		opts := []llb.RunOption{llb.Shlex(cmd.Cmd)}
		if cmd.CustomName != "" {
			opts = append(opts, llb.WithCustomName(cmd.CustomName))
		}
		s := state.Run(opts...).Root()
		return &s, nil

	case plan.PathCommand:
		// TODO: Build up the path so we are not starting from scratch each time
		s := state.AddEnvf("PATH", "%s:%s", cmd.Path, system.DefaultPathEnvUnix)
		return &s, nil

	case plan.VariableCommand:
		s := state.AddEnv(cmd.Name, cmd.Value)
		return &s, nil

	case plan.CopyCommand:
		src := llb.Local("context")
		s := state.File(llb.Copy(src, cmd.Src, cmd.Dst, &llb.CopyInfo{
			CopyDirContentsOnly: true,
		}))
		return &s, nil

	case plan.FileCommand:
		asset, ok := step.Assets[cmd.Name]
		if !ok {
			return state, fmt.Errorf("asset %q not found", cmd.Name)
		}

		// Create parent directories for the file
		parentDir := filepath.Dir(cmd.Path)
		if parentDir != "/" {
			s := state.File(llb.Mkdir(parentDir, 0755, llb.WithParents(true)))
			state = &s
		}

		fileAction := llb.Mkfile(cmd.Path, 0644, []byte(asset))
		s := state.File(fileAction)
		if cmd.CustomName != "" {
			s = state.File(fileAction, llb.WithCustomName(cmd.CustomName))
		}

		return &s, nil
	}

	return state, nil
}
