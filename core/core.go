package core

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/railwayapp/railpack-go/core/app"
	"github.com/railwayapp/railpack-go/core/generate"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/railwayapp/railpack-go/core/providers"
	"github.com/railwayapp/railpack-go/core/resolver"
)

type GenerateBuildPlanOptions struct{}

type BuildResult struct {
	Plan             *plan.BuildPlan                      `json:"plan"`
	ResolvedPackages map[string]*resolver.ResolvedPackage `json:"resolved_packages"`
}

func GenerateBuildPlan(app *app.App, env *app.Environment, options *GenerateBuildPlanOptions) (*BuildResult, error) {
	ctx, err := generate.NewGenerateContext(app, env)
	if err != nil {
		return nil, err
	}

	generate.MiseStep(ctx)

	for _, provider := range providers.GetLanguageProviders() {
		matched, err := runProvider(provider, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to run provider: %w", err)
		}

		if matched {
			log.Debugf("Provider `%s` matched", provider.Name())
			break
		}
	}

	resolvedPackages, err := ctx.ResolvePackages()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve packages: %w", err)
	}

	buildPlan := plan.NewBuildPlan()

	buildStepOptions := &generate.BuildStepOptions{
		ResolvedPackages: resolvedPackages,
	}

	buildPlan.Variables = ctx.Variables
	for _, stepBuilder := range ctx.Steps {
		step, err := stepBuilder.Build(buildStepOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to build step: %w", err)
		}

		buildPlan.AddStep(*step)
	}

	buildResult := &BuildResult{
		Plan:             buildPlan,
		ResolvedPackages: resolvedPackages,
	}

	return buildResult, nil
}

func runProvider(provider providers.Provider, ctx *generate.GenerateContext) (bool, error) {
	return provider.Plan(ctx)
}
