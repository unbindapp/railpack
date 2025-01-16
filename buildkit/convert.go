package buildkit

import (
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/railwayapp/railpack-go/core/plan"
)

type ConvertPlanOptions struct {
	BuildPlatform BuildPlatform
}

func ConvertPlanToLLB(plan *plan.BuildPlan, opts ConvertPlanOptions) (*llb.State, *Image, error) {
	platform := specs.Platform{
		OS:           opts.BuildPlatform.OS,
		Architecture: opts.BuildPlatform.Architecture,
		Variant:      opts.BuildPlatform.Variant,
	}

	state := llb.Image("ubuntu:noble",
		llb.Platform(platform),
	)

	// Install curl
	state = state.Run(llb.Shlex("sh -c 'apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*'"), llb.WithCustomName("install base apt packages")).
		Root()

	if len(plan.Packages.Apt) > 0 {
		aptPackages := strings.Join(plan.Packages.Apt, " ")
		state = state.Run(llb.Shlex("sh -c 'apt-get update && apt-get install -y "+aptPackages+" && rm -rf /var/lib/apt/lists/*'"),
			llb.WithCustomNamef("install apt packages: %s", aptPackages)).
			Root()
	}

	// Install mise
	state = state.AddEnv("GIT_SSL_CAINFO", "/etc/ssl/certs/ca-certificates.crt").
		AddEnv("MISE_DATA_DIR", "/mise").
		AddEnv("MISE_CONFIG_DIR", "/mise").
		AddEnv("MISE_INSTALL_PATH", "/usr/local/bin/mise").
		AddEnv("PATH", "/mise/shims:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin").
		Run(llb.Shlex("sh -c 'curl -fsSL https://mise.run | sh'"), llb.WithCustomName("install mise")).
		Root()

	// Set working directory
	state = state.Dir("/app")

	// Add all variables as environment variables
	for name, value := range plan.Variables {
		state = state.AddEnv(name, value)
	}

	// Generate mise.toml
	miseToml, err := plan.Packages.GenerateMiseToml()
	if err != nil {
		return nil, nil, err
	}

	misePackages := make([]string, 0, len(plan.Packages.Mise))
	for k := range plan.Packages.Mise {
		misePackages = append(misePackages, k)
	}

	// Install mise packages
	state = state.File(llb.Mkdir("/etc/mise", 0755, llb.WithParents(true)), llb.WithCustomName("create mise dir")).
		File(llb.Mkfile("/etc/mise/config.toml", 0644, []byte(miseToml)), llb.WithCustomName("create mise config")).
		Run(llb.Shlex("sh -c 'mise trust -a && mise install'"), llb.WithCustomNamef("install mise packages: %s", strings.Join(misePackages, ", "))).
		Root()

	// TODO: Be smarter about which steps build off each other based on step.DependsOn
	// Parallelize steps that don't depend on each other
	for _, step := range plan.Steps {
		for _, cmd := range step.Commands {
			state = convertCommandToLLB(cmd, &state)
		}
	}

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
		},
	}

	return &state, &image, nil
}

func convertCommandToLLB(cmd plan.Command, state *llb.State) llb.State {
	switch cmd := cmd.(type) {
	case plan.ExecCommand:
		return state.Run(llb.Shlex(cmd.Cmd)).Root()
	case plan.PathCommand:
		return state.Run(llb.Shlex(cmd.Path)).Root()
	case plan.VariableCommand:
		return state.AddEnv(cmd.Name, cmd.Value)
	case plan.CopyCommand:
		src := llb.Local("context")
		return state.File(llb.Copy(src, cmd.Src, cmd.Dst, &llb.CopyInfo{
			CopyDirContentsOnly: true,
		}))
	}

	return *state
}
