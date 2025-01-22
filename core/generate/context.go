package generate

import (
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
	Start     StartContext

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
		Start:     NewStartContext(),
		resolver:  resolver,
	}, nil
}

func (c *GenerateContext) ResolvePackages() (map[string]*resolver.ResolvedPackage, error) {
	return c.resolver.ResolvePackages()
}
