package providers

import (
	"github.com/unbindapp/railpack/core/generate"
	"github.com/unbindapp/railpack/core/providers/deno"
	"github.com/unbindapp/railpack/core/providers/golang"
	"github.com/unbindapp/railpack/core/providers/java"
	"github.com/unbindapp/railpack/core/providers/node"
	"github.com/unbindapp/railpack/core/providers/php"
	"github.com/unbindapp/railpack/core/providers/python"
	"github.com/unbindapp/railpack/core/providers/shell"
	"github.com/unbindapp/railpack/core/providers/staticfile"
)

type Provider interface {
	Name() string
	Detect(ctx *generate.GenerateContext) (bool, error)
	Initialize(ctx *generate.GenerateContext) error
	Plan(ctx *generate.GenerateContext) error
	StartCommandHelp() string
}

func GetLanguageProviders() []Provider {
	// Order is important here. The first provider that returns true from Detect() will be used.
	return []Provider{
		&php.PhpProvider{},
		&golang.GoProvider{},
		&java.JavaProvider{},
		&python.PythonProvider{},
		&deno.DenoProvider{},
		&node.NodeProvider{},
		&staticfile.StaticfileProvider{},
		&shell.ShellProvider{},
	}
}

func GetProvider(name string) Provider {
	for _, provider := range GetLanguageProviders() {
		if provider.Name() == name {
			return provider
		}
	}

	return nil
}
