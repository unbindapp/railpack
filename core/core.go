package core

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/railwayapp/railpack-go/core/app"
	"github.com/railwayapp/railpack-go/core/generate"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/railwayapp/railpack-go/core/providers"
)

type GenerateBuildPlanOptions struct {
}

func GenerateBuildPlan(app *app.App, env *app.Environment, options *GenerateBuildPlanOptions) (*plan.BuildPlan, error) {
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
			log.Debugf("Provider %s matched", provider)
			break
		}
	}

	return nil, nil
}

func runProvider(provider providers.Provider, ctx *generate.GenerateContext) (bool, error) {
	return provider.Plan(ctx)
}
