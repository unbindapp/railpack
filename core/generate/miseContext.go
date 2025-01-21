package generate

import (
	"fmt"
	"strings"

	a "github.com/railwayapp/railpack-go/core/app"
	"github.com/railwayapp/railpack-go/core/mise"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/railwayapp/railpack-go/core/resolver"
)

const (
	PackagesStepName = "packages"
)

type PackageStepBuilder struct {
	DisplayName           string
	Resolver              *resolver.Resolver
	SupportingAptPackages []string
	MisePackages          []*resolver.PackageRef
	SupportingMiseFiles   []string
	Assets                map[string]string
	DependsOn             []string

	app *a.App
	env *a.Environment
}

func (c *GenerateContext) NewPackageStep(name string) *PackageStepBuilder {
	step := &PackageStepBuilder{
		DisplayName:           name,
		Resolver:              c.resolver,
		MisePackages:          []*resolver.PackageRef{},
		SupportingAptPackages: []string{},
		Assets:                map[string]string{},
		DependsOn:             []string{},

		app: c.App,
		env: c.Env,
	}

	c.Steps = append(c.Steps, step)

	return step
}

func (b *PackageStepBuilder) AddAptPackage(name string) {
	b.SupportingAptPackages = append(b.SupportingAptPackages, name)
}

func (b *PackageStepBuilder) Default(name string, defaultVersion string) resolver.PackageRef {
	for _, pkg := range b.MisePackages {
		if pkg.Name == name {
			return *pkg
		}
	}

	pkg := b.Resolver.Default(name, defaultVersion)
	b.MisePackages = append(b.MisePackages, &pkg)
	return pkg
}

func (b *PackageStepBuilder) Version(name resolver.PackageRef, version string, source string) {
	b.Resolver.Version(name, version, source)
}

func (b *PackageStepBuilder) Build(options *BuildStepOptions) (*plan.Step, error) {
	step := plan.NewStep(b.DisplayName)

	step.DependsOn = b.DependsOn

	// Install mise
	step.AddCommands([]plan.Command{
		plan.NewVariableCommand("MISE_DATA_DIR", "/mise"),
		plan.NewVariableCommand("MISE_CONFIG_DIR", "/mise"),
		plan.NewVariableCommand("MISE_INSTALL_PATH", "/usr/local/bin/mise"),
		plan.NewPathCommand("/mise/shims"),
		plan.NewExecCommand("sh -c 'apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*'", "install curl"),
		plan.NewExecCommand("sh -c 'curl -fsSL https://mise.run | sh'", "install mise"),
	})

	// Add user mise config files if they exist
	supportingMiseConfigFiles := b.GetSupportingMiseConfigFiles(b.app.Source)
	for _, file := range supportingMiseConfigFiles {
		step.AddCommands([]plan.Command{
			plan.NewCopyCommand(file, "/app/"+file),
		})
	}

	// Setup apt commands
	if len(b.SupportingAptPackages) > 0 {
		pkgString := strings.Join(b.SupportingAptPackages, " ")
		step.AddCommands([]plan.Command{
			plan.NewExecCommand("sh -c 'apt-get update && apt-get install -y "+pkgString+" && rm -rf /var/lib/apt/lists/*'", "install apt packages: "+pkgString),
		})
	}

	// Setup mise commands
	packagesToInstall := make(map[string]string)
	for _, pkg := range b.MisePackages {
		resolved, ok := options.ResolvedPackages[pkg.Name]
		if ok && resolved.ResolvedVersion != nil {
			packagesToInstall[pkg.Name] = *resolved.ResolvedVersion
		}
	}

	miseToml, err := mise.GenerateMiseToml(packagesToInstall)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mise.toml: %w", err)
	}

	b.Assets["mise.toml"] = miseToml

	pkgNames := make([]string, 0, len(packagesToInstall))
	for k := range packagesToInstall {
		pkgNames = append(pkgNames, k)
	}

	step.AddCommands([]plan.Command{
		plan.NewFileCommand("/etc/mise/config.toml", "mise.toml", "create mise config"),
		plan.NewExecCommand("sh -c 'mise trust -a && mise install'", "install mise packages: "+strings.Join(pkgNames, ", ")),
	})

	step.Assets = b.Assets

	step.Outputs = []string{"/mise/shims", "/mise/installs", "/usr/local/bin/mise", "/etc/mise/config.toml", "/root/.local/state/mise"}

	return step, nil
}

var miseConfigFiles = []string{
	"mise.toml",
	".python-version",
	".nvmrc",
}

func (b *PackageStepBuilder) GetSupportingMiseConfigFiles(path string) []string {
	files := []string{}

	for _, file := range miseConfigFiles {
		if b.app.HasMatch(file) {
			files = append(files, file)
		}
	}

	return files
}
