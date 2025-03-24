package generate

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	a "github.com/unbindapp/railpack/core/app"
	"github.com/unbindapp/railpack/core/mise"
	"github.com/unbindapp/railpack/core/plan"
	"github.com/unbindapp/railpack/core/resolver"
)

const (
	MisePackageStepName = "packages:mise"
)

type MiseStepBuilder struct {
	DisplayName           string
	Resolver              *resolver.Resolver
	SupportingAptPackages []string
	MisePackages          []*resolver.PackageRef
	SupportingMiseFiles   []string
	Assets                map[string]string
	Inputs                []plan.Input
	Variables             map[string]string
	app                   *a.App
	env                   *a.Environment
}

func (c *GenerateContext) NewMiseStepBuilder(displayName string) *MiseStepBuilder {
	supportingAptPackages := c.Config.BuildAptPackages

	step := &MiseStepBuilder{
		DisplayName:           displayName,
		Resolver:              c.Resolver,
		MisePackages:          []*resolver.PackageRef{},
		SupportingAptPackages: append(supportingAptPackages, c.Config.BuildAptPackages...),
		Assets:                map[string]string{},
		Inputs:                []plan.Input{},
		Variables:             map[string]string{},
		app:                   c.App,
		env:                   c.Env,
	}

	c.Steps = append(c.Steps, step)

	return step
}

func (c *GenerateContext) newMiseStepBuilder() *MiseStepBuilder {
	step := c.NewMiseStepBuilder(MisePackageStepName)

	return step
}

func (b *MiseStepBuilder) AddSupportingAptPackage(name string) {
	b.SupportingAptPackages = append(b.SupportingAptPackages, name)
}

func (b *MiseStepBuilder) AddInput(input plan.Input) {
	b.Inputs = append(b.Inputs, input)
}

func (b *MiseStepBuilder) Default(name string, defaultVersion string) resolver.PackageRef {
	for _, pkg := range b.MisePackages {
		if pkg.Name == name {
			return *pkg
		}
	}

	pkg := b.Resolver.Default(name, defaultVersion)
	b.MisePackages = append(b.MisePackages, &pkg)
	return pkg
}

func (b *MiseStepBuilder) Version(name resolver.PackageRef, version string, source string) {
	b.Resolver.Version(name, version, source)
}

func (b *MiseStepBuilder) Name() string {
	return b.DisplayName
}

func (b *MiseStepBuilder) GetOutputPaths() []string {
	supportingMiseConfigFiles := b.GetSupportingMiseConfigFiles(b.app.Source)
	files := []string{"/mise/shims", "/mise/installs", "/usr/local/bin/mise", "/etc/mise/config.toml", "/root/.local/state/mise"}
	files = append(files, supportingMiseConfigFiles...)
	return files
}

func (b *MiseStepBuilder) Build(options *BuildStepOptions) (*plan.Step, error) {
	step := plan.NewStep(b.DisplayName)

	step.Inputs = []plan.Input{
		plan.NewImageInput(plan.RAILPACK_BUILDER_IMAGE),
	}

	// Setup apt commands
	// TODO: This should be a separate step
	if len(b.SupportingAptPackages) > 0 {
		step.AddCommands([]plan.Command{
			options.NewAptInstallCommand(b.SupportingAptPackages),
		})
		step.Caches = options.Caches.GetAptCaches()
	}

	if len(b.MisePackages) == 0 {
		return step, nil
	}

	// Setup mise
	step.AddCommands([]plan.Command{
		plan.NewPathCommand("/mise/shims"),
	})
	maps.Copy(step.Variables, map[string]string{
		"MISE_DATA_DIR":     "/mise",
		"MISE_CONFIG_DIR":   "/mise",
		"MISE_CACHE_DIR":    "/mise/cache",
		"MISE_SHIMS_DIR":    "/mise/shims",
		"MISE_INSTALLS_DIR": "/mise/installs",
	})
	maps.Copy(step.Variables, b.Variables)

	if verbose := b.env.GetVariable("MISE_VERBOSE"); verbose != "" {
		step.Variables["MISE_VERBOSE"] = verbose
	}

	// Add user mise config files if they exist
	supportingMiseConfigFiles := b.GetSupportingMiseConfigFiles(b.app.Source)
	for _, file := range supportingMiseConfigFiles {
		step.AddCommands([]plan.Command{
			plan.NewCopyCommand(file),
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
	sort.Strings(pkgNames)

	step.AddCommands([]plan.Command{
		plan.NewFileCommand("/etc/mise/config.toml", "mise.toml", plan.FileOptions{
			CustomName: "create mise config",
		}),
		plan.NewExecCommand("sh -c 'mise trust -a && mise install'", plan.ExecOptions{
			CustomName: "install mise packages: " + strings.Join(pkgNames, ", "),
		}),
	})

	step.Assets = b.Assets
	step.Secrets = []string{}

	return step, nil
}

var miseConfigFiles = []string{
	"mise.toml",
	".tool-versions",
	".python-version",
	".nvmrc",
}

func (b *MiseStepBuilder) GetSupportingMiseConfigFiles(path string) []string {
	files := []string{}

	for _, file := range miseConfigFiles {
		if b.app.HasMatch(file) {
			files = append(files, file)
		}
	}

	return files
}
