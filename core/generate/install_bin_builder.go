package generate

import (
	"fmt"

	"github.com/unbindapp/railpack/core/plan"
	"github.com/unbindapp/railpack/core/resolver"
)

const (
	BinDir = "/railpack"
)

type InstallBinStepBuilder struct {
	DisplayName           string
	Resolver              *resolver.Resolver
	SupportingAptPackages []string
	Package               resolver.PackageRef
}

func (c *GenerateContext) NewInstallBinStepBuilder(name string) *InstallBinStepBuilder {
	step := &InstallBinStepBuilder{
		DisplayName: c.GetStepName(name),
		Resolver:    c.Resolver,
		Package:     resolver.PackageRef{},
	}

	c.Steps = append(c.Steps, step)

	return step
}

func (b *InstallBinStepBuilder) Name() string {
	return b.DisplayName
}

func (b *InstallBinStepBuilder) Default(name string, defaultVersion string) resolver.PackageRef {
	b.Package = b.Resolver.Default(name, defaultVersion)
	return b.Package
}

func (b *InstallBinStepBuilder) GetOutputPaths() []string {
	return []string{b.getBinPath()}
}

func (b *InstallBinStepBuilder) Version(name resolver.PackageRef, version string, source string) {
	b.Resolver.Version(name, version, source)
}

func (b *InstallBinStepBuilder) Build(options *BuildStepOptions) (*plan.Step, error) {
	packageVersion := options.ResolvedPackages[b.Package.Name].ResolvedVersion
	if packageVersion == nil {
		return nil, fmt.Errorf("package %s not found", b.Package.Name)
	}

	step := plan.NewStep(b.DisplayName)

	step.Inputs = []plan.Input{
		plan.NewImageInput(plan.RAILPACK_BUILDER_IMAGE),
	}

	binPath := b.getBinPath()

	step.AddCommands([]plan.Command{
		plan.NewExecCommand(fmt.Sprintf("mise install-into %s@%s %s", b.Package.Name, *packageVersion, binPath)),
		plan.NewPathCommand(binPath),
		plan.NewPathCommand(fmt.Sprintf("%s/bin", binPath)),
	})

	step.Secrets = []string{}

	return step, nil
}

func (b *InstallBinStepBuilder) getBinPath() string {
	return fmt.Sprintf("%s/%s", BinDir, b.Package.Name)
}
