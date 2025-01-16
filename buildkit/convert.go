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

	// Install curl
	// state = state.Run(llb.Shlex("sh -c 'apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*'"), llb.WithCustomName("install base apt packages")).
	// 	Root()

	// if len(plan.Packages.Apt) > 0 {
	// 	aptPackages := strings.Join(plan.Packages.Apt, " ")
	// 	state = state.Run(llb.Shlex("sh -c 'apt-get update && apt-get install -y "+aptPackages+" && rm -rf /var/lib/apt/lists/*'"),
	// 		llb.WithCustomNamef("install apt packages: %s", aptPackages)).
	// 		Root()
	// }

	// // Install mise
	// state = state.AddEnv("GIT_SSL_CAINFO", "/etc/ssl/certs/ca-certificates.crt").
	// 	AddEnv("MISE_DATA_DIR", "/mise").
	// 	AddEnv("MISE_CONFIG_DIR", "/mise").
	// 	AddEnv("MISE_INSTALL_PATH", "/usr/local/bin/mise").
	// 	AddEnv("PATH", "/mise/shims:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin").
	// 	Run(llb.Shlex("sh -c 'curl -fsSL https://mise.run | sh'"), llb.WithCustomName("install mise")).
	// 	Root()

	// Set working directory
	state = state.Dir("/app")

	// Add all variables as environment variables
	for name, value := range plan.Variables {
		state = state.AddEnv(name, value)
	}

	for _, step := range plan.Steps {
		var err error
		stepState, err := convertStepToLLB(&step, &state)
		if err != nil {
			return nil, nil, err
		}

		state = *stepState
	}

	// Generate mise.toml
	// miseToml, err := mise.GenerateMiseToml(plan.Packages.Mise)
	// if err != nil {
	// 	return nil, nil, err
	// }

	// misePackages := make([]string, 0, len(plan.Packages.Mise))
	// for k := range plan.Packages.Mise {
	// 	misePackages = append(misePackages, k)
	// }

	// // Install mise packages
	// state = state.File(llb.Mkdir("/etc/mise", 0755, llb.WithParents(true)), llb.WithCustomName("create mise dir")).
	// 	File(llb.Mkfile("/etc/mise/config.toml", 0644, []byte(miseToml)), llb.WithCustomName("create mise config")).
	// 	Run(llb.Shlex("sh -c 'mise trust -a && mise install'"), llb.WithCustomNamef("install mise packages: %s", strings.Join(misePackages, ", "))).
	// 	Root()

	// // TODO: Be smarter about which steps build off each other based on step.DependsOn
	// // Parallelize steps that don't depend on each other
	// for _, step := range plan.Steps {
	// 	for _, cmd := range step.Commands {
	// 		state = convertCommandToLLB(cmd, &state)
	// 	}
	// }

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
