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

	resolvedPackages, err := ctx.Resolver.ResolvePackages()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve packages: %w", err)
	}

	buildPlan := plan.NewBuildPlan()

	buildPlan.Variables = ctx.Variables
	for _, step := range ctx.Steps {
		buildPlan.AddStep(*step)
	}

	for _, aptPackage := range ctx.AptPackages {
		buildPlan.Packages.AddAptPackage(aptPackage)
	}

	for _, resolvedPackage := range resolvedPackages {
		if resolvedPackage.ResolvedVersion != nil {
			buildPlan.Packages.AddMisePackage(resolvedPackage.Name, *resolvedPackage.ResolvedVersion)
		}
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
