package node

import "github.com/railwayapp/railpack-go/core/generate"

type NodeProvider struct{}

const DEFAULT_NODE_VERSION = "22"

func (p *NodeProvider) Plan(ctx *generate.GenerateContext) (bool, error) {
	ctx.Resolver.Default("node", DEFAULT_NODE_VERSION)

	return true, nil
}
