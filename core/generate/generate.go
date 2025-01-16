package generate

import "github.com/railwayapp/railpack-go/core/plan"

const (
	PackagesStepName    = "packages"
	MiseInstallStepName = "mise"
)

func MiseStep(ctx *GenerateContext) *ProviderStepBuilder {
	step := ctx.NewProviderStep(MiseInstallStepName)
	step.DependsOn = []string{}

	step.AddCommands([]plan.Command{
		plan.NewExecCommand("sh -c 'apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*'", "install curl"),
		plan.NewVariableCommand("MISE_DATA_DIR", "/mise"),
		plan.NewVariableCommand("MISE_CONFIG_DIR", "/mise"),
		plan.NewVariableCommand("MISE_INSTALL_PATH", "/usr/local/bin/mise"),
		plan.NewPathCommand("/mise/shims"),
		plan.NewExecCommand("sh -c 'curl -fsSL https://mise.run | sh'", "install mise"),
	})

	return step
}
