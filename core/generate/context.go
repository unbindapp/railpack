package generate

import (
	"github.com/railwayapp/railpack-go/core/app"
	"github.com/railwayapp/railpack-go/core/mise"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/railwayapp/railpack-go/core/resolver"
)

type GenerateContext struct {
	app         *app.App
	env         *app.Environment
	Resolver    *resolver.Resolver
	Variables   map[string]string
	AptPackages []string
	Steps       map[string]*plan.Step
}

func NewGenerateContext(app *app.App, env *app.Environment) (*GenerateContext, error) {
	resolver, err := resolver.NewResolver(mise.TestInstallDir)
	if err != nil {
		return nil, err
	}

	return &GenerateContext{
		app:         app,
		env:         env,
		Resolver:    resolver,
		Variables:   map[string]string{},
		AptPackages: []string{},
		Steps:       make(map[string]*plan.Step),
	}, nil
}

func (c *GenerateContext) AddStep(step *plan.Step) {
	if existingStep, exists := c.Steps[step.Name]; exists {
		mergedStep := plan.MergeSteps(existingStep, step)
		c.Steps[step.Name] = mergedStep
	} else {
		c.Steps[step.Name] = step
	}
}

func (c *GenerateContext) AddAptPackage(name string) {
	c.AptPackages = append(c.AptPackages, name)
}
