package buildkit

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/unbindapp/railpack/buildkit/build_llb"
	p "github.com/unbindapp/railpack/core/plan"
)

type ConvertPlanOptions struct {
	BuildPlatform BuildPlatform
	SecretsHash   string
	CacheKey      string
	SessionID     string
}

const (
	WorkingDir = "/app"
)

func ConvertPlanToLLB(plan *p.BuildPlan, opts ConvertPlanOptions) (*llb.State, *Image, error) {
	platform := opts.BuildPlatform.ToPlatform()

	localState := llb.Local("context",
		llb.SharedKeyHint("local"),
		llb.SessionID(opts.SessionID),
		llb.WithCustomName("loading ."),
		llb.FollowPaths([]string{"."}),
	)

	cacheStore := build_llb.NewBuildKitCacheStore(opts.CacheKey)
	graph, err := build_llb.NewBuildGraph(plan, &localState, cacheStore, opts.SecretsHash, &platform)
	if err != nil {
		return nil, nil, err
	}

	graphOutput, err := graph.GenerateLLB()
	if err != nil {
		return nil, nil, err
	}

	state := getStartState(*graphOutput.State)
	imageEnv := getImageEnv(graphOutput, plan)

	startCommand := plan.Deploy.StartCmd
	if startCommand == "" {
		startCommand = "/bin/bash"
	}

	image := Image{
		Image: specs.Image{
			Platform: specs.Platform{
				OS:           platform.OS,
				Architecture: platform.Architecture,
			},
			RootFS: specs.RootFS{
				Type: "layers",
			},
		},
		Variant: platform.Variant,
		Config: specs.ImageConfig{
			Env:        imageEnv,
			WorkingDir: WorkingDir,
			Entrypoint: []string{"/bin/sh", "-c"},
			Cmd:        []string{startCommand},
		},
	}

	return &state, &image, nil
}

func getStartState(buildState llb.State) llb.State {
	startState := buildState.Dir(WorkingDir)
	return startState
}

func getImageEnv(graphOutput *build_llb.BuildGraphOutput, plan *p.BuildPlan) []string {
	paths := []string{}
	paths = append(paths, plan.Deploy.Paths...)
	paths = append(paths, graphOutput.GraphEnv.PathList...)
	paths = append(paths, system.DefaultPathEnvUnix)
	slices.Sort(paths)
	pathString := strings.Join(paths, ":")

	envMap := make(map[string]string, len(graphOutput.GraphEnv.EnvVars)+len(plan.Deploy.Variables)+1)
	maps.Copy(envMap, graphOutput.GraphEnv.EnvVars)
	maps.Copy(envMap, plan.Deploy.Variables)

	envMap["PATH"] = pathString

	envVars := make([]string, 0, len(envMap))
	for _, k := range slices.Sorted(maps.Keys(envMap)) {
		v := envMap[k]
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	return envVars
}
