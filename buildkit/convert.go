package buildkit

import (
	"strings"

	"github.com/charmbracelet/log"
	"github.com/moby/buildkit/client/llb"
	"github.com/railwayapp/railpack-go/core/plan"
)

func ConvertPlanToLLB(plan *plan.BuildPlan) (*llb.State, error) {
	base := llb.Image("ubuntu:noble")
	state := base

	// Install curl
	state = base.Run(llb.Shlex("apt-get update")).
		Run(llb.Shlex("apt-get install -y curl")).
		Run(llb.Shlex("rm -rf /var/lib/apt/lists/*")).
		Root()

	if len(plan.Packages.Apt) > 0 {
		aptPackages := strings.Join(plan.Packages.Apt, " ")
		state = state.Run(llb.Shlex("apt-get update")).
			Run(llb.Shlex("apt-get install -y " + aptPackages)).
			Run(llb.Shlex("rm -rf /var/lib/apt/lists/*")).
			Root()
	}

	// Install mise
	state = state.AddEnv("GIT_SSL_CAINFO", "/etc/ssl/certs/ca-certificates.crt").
		AddEnv("MISE_DATA_DIR", "/mise").
		AddEnv("MISE_CONFIG_DIR", "/mise").
		AddEnv("MISE_INSTALL_PATH", "/usr/local/bin/mise").
		AddEnv("PATH", "/mise/shims:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin").
		Run(llb.Shlex("sh -c 'curl -fsSL https://mise.run | sh'")).Root()

	// Set working directory
	state = state.Dir("/app")

	// Add all variables as environment variables
	for name, value := range plan.Variables {
		state = state.AddEnv(name, value)
	}

	// Generate mise.toml
	miseToml, err := plan.Packages.GenerateMiseToml()
	if err != nil {
		return nil, err
	}

	log.Debugf("Mise TOML:\n%s", miseToml)

	// Install mise packages
	state = state.File(llb.Mkdir("/etc/mise", 0755, llb.WithParents(true))).
		File(llb.Mkfile("/etc/mise/config.toml", 0644, []byte(miseToml))).
		Run(llb.Shlex("mise trust -a")).
		Run(llb.Shlex("mise install")).
		Root()

	// TODO: Be smarter about which steps build off each other based on step.DependsOn
	// Parallelize steps that don't depend on each other
	for _, step := range plan.Steps {
		for _, cmd := range step.Commands {
			state = convertCommandToLLB(cmd, &state)
		}
	}

	return &state, nil
}

func convertCommandToLLB(cmd plan.Command, state *llb.State) llb.State {
	switch cmd := cmd.(type) {
	case plan.ExecCommand:
		return state.Run(llb.Shlex(cmd.Cmd)).Root()
	case plan.GlobalPathCommand:
		return state.Run(llb.Shlex(cmd.GlobalPath)).Root()
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
