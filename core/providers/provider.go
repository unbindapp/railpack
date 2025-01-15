package providers

import (
	"github.com/railwayapp/railpack-go/core/generate"
	"github.com/railwayapp/railpack-go/core/providers/node"
)

type Provider interface {
	Name() string
	Plan(ctx *generate.GenerateContext) (bool, error)
}

func GetLanguageProviders() []Provider {
	return []Provider{
		&node.NodeProvider{},
	}
}
