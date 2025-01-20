package generate

import (
	"fmt"
	"strings"

	a "github.com/railwayapp/railpack-go/core/app"
	"github.com/railwayapp/railpack-go/core/mise"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/railwayapp/railpack-go/core/resolver"
)

type BuildStepOptions struct {
	ResolvedPackages map[string]*resolver.ResolvedPackage
}

type StepBuilder interface {
	Build(options *BuildStepOptions) (*plan.Step, error)
}

type GenerateContext struct {
	App       *a.App
	Env       *a.Environment
	Variables map[string]string
	Steps     []StepBuilder

	resolver *resolver.Resolver
}

func NewGenerateContext(app *a.App, env *a.Environment) (*GenerateContext, error) {
	resolver, err := resolver.NewResolver(mise.TestInstallDir)
	if err != nil {
		return nil, err
	}

	return &GenerateContext{
		App:       app,
		Env:       env,
		Variables: map[string]string{},
		Steps:     make([]StepBuilder, 0),
		resolver:  resolver,
	}, nil
}

type ProviderStepBuilder struct {
	DisplayName string
	DependsOn   []string
	Commands    []plan.Command
	Outputs     []string
}

func (c *GenerateContext) NewProviderStep(name string) *ProviderStepBuilder {
	step := &ProviderStepBuilder{
		DisplayName: name,
		DependsOn:   []string{PackagesStepName},
		Commands:    []plan.Command{},
		Outputs:     []string{},
	}

	c.Steps = append(c.Steps, step)

	return step
}

type PackageStepBuilder struct {
	DisplayName         string
	Resolver            *resolver.Resolver
	AptPackages         []string
	MisePackages        []*resolver.PackageRef
	SupportingMiseFiles []string
	Assets              map[string]string
	DependsOn           []string
}

func (c *GenerateContext) NewPackageStep(name string) *PackageStepBuilder {
	step := &PackageStepBuilder{
		DisplayName:         name,
		Resolver:            c.resolver,
		AptPackages:         []string{},
		MisePackages:        []*resolver.PackageRef{},
		SupportingMiseFiles: []string{},
		Assets:              map[string]string{},
		DependsOn:           []string{MiseInstallStepName},
	}

	c.Steps = append(c.Steps, step)

	return step
}

func (c *GenerateContext) ResolvePackages() (map[string]*resolver.ResolvedPackage, error) {
	return c.resolver.ResolvePackages()
}

func (b *ProviderStepBuilder) DependOn(name string) {
	b.DependsOn = append(b.DependsOn, name)
}

func (b *ProviderStepBuilder) AddCommands(commands []plan.Command) {
	b.Commands = append(b.Commands, commands...)
}

func (b *ProviderStepBuilder) Build(options *BuildStepOptions) (*plan.Step, error) {
	step := plan.NewStep(b.DisplayName)

	step.DependsOn = b.DependsOn
	step.Commands = b.Commands
	step.Outputs = b.Outputs

	return step, nil
}

func (b *PackageStepBuilder) AddAptPackage(name string) {
	b.AptPackages = append(b.AptPackages, name)
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

	// Setup apt commands
	if len(b.AptPackages) > 0 {
		pkgString := strings.Join(b.AptPackages, " ")
		step.AddCommands([]plan.Command{
			plan.NewExecCommand("apt-get update && apt-get install -y "+pkgString+" && rm -rf /var/lib/apt/lists/*", "install apt packages: "+pkgString),
		})
	}

	// Setup mise commands
	if len(b.MisePackages) > 0 {
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
	}

	return step, nil
}
