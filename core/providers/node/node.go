package node

import (
	"fmt"
	"strings"

	"github.com/railwayapp/railpack-go/core/app"
	"github.com/railwayapp/railpack-go/core/generate"
	"github.com/railwayapp/railpack-go/core/plan"
)

type PackageJson struct {
	Scripts        map[string]string `json:"scripts"`
	PackageManager *string           `json:"packageManager"`
	Engines        map[string]string `json:"engines"`
}

type PackageManager string

const (
	DEFAULT_NODE_VERSION = "22"
	DEFAULT_BUN_VERSION  = "latest"
)

type NodeProvider struct{}

func (p *NodeProvider) Name() string {
	return "node"
}

func (p *NodeProvider) Plan(ctx *generate.GenerateContext) (bool, error) {
	packageJson, err := p.getPackageJson(ctx.App)
	if err != nil {
		return false, err
	}

	if packageJson == nil {
		return false, nil
	}

	if err := p.packages(ctx, packageJson); err != nil {
		return false, err
	}

	if err := p.install(ctx, packageJson); err != nil {
		return false, err
	}

	// if err := p.build(ctx, packageJson); err != nil {
	// 	return false, err
	// }

	return true, nil
}

func (p *NodeProvider) install(ctx *generate.GenerateContext, packageJson *PackageJson) error {
	corepack := p.usesCorepack(packageJson)
	if corepack {
		setup := plan.NewStep("setup")
		setup.AddCommands([]plan.Command{
			plan.NewCopyCommand("package.json", "."),
			plan.NewExecCommand("npm install -g corepack"),
			plan.NewExecCommand("corepack enable"),
			plan.NewExecCommand("corepack prepare --activate"),
		})

		ctx.AddStep(setup)
	}

	pkgManager := p.getPackageManager(ctx.App)

	install := plan.NewStep("install")
	install.AddCommands([]plan.Command{
		plan.NewCopyCommand(".", "."),
		plan.NewExecCommand(pkgManager.InstallDeps()),
	})

	if corepack {
		install.DependOn("setup")
	}

	ctx.AddStep(install)
	return nil
}

func (p *NodeProvider) packages(ctx *generate.GenerateContext, packageJson *PackageJson) error {
	packageManager := p.getPackageManager(ctx.App)

	// Node
	node := ctx.Resolver.Default("node", DEFAULT_NODE_VERSION)

	envVersion := ctx.Env.GetConfigVariable("NODE_VERSION")
	if envVersion != "" {
		ctx.Resolver.Version(node, envVersion, "RAILPACK_NODE_VERSION")
	}

	if packageJson.Engines != nil && packageJson.Engines["node"] != "" {
		ctx.Resolver.Version(node, packageJson.Engines["node"], "package.json > engines > node")
	}

	if packageManager == PackageManagerBun {
		bun := ctx.Resolver.Default("bun", DEFAULT_BUN_VERSION)

		envVersion := ctx.Env.GetConfigVariable("BUN_VERSION")
		if envVersion != "" {
			ctx.Resolver.Version(bun, envVersion, "RAILPACK_BUN_VERSION")
		}
	}

	return p.managerPackages(ctx, packageManager)
}

func (p *NodeProvider) managerPackages(ctx *generate.GenerateContext, packageManager PackageManager) error {
	// NPM
	if packageManager == PackageManagerNpm {
		npm := ctx.Resolver.Default("npm", "latest")

		lockfile, err := ctx.App.ReadFile("package-lock.json")
		if err != nil {
			return fmt.Errorf("error reading package-lock.json: %w", err)
		}

		if strings.Contains(lockfile, "\"lockfileVersion\": 1") {
			ctx.Resolver.Version(npm, "6", "package-lock.json")
		} else if strings.Contains(lockfile, "\"lockfileVersion\": 2") {
			ctx.Resolver.Version(npm, "8", "package-lock.json")
		}
	}

	// Pnpm
	if packageManager == PackageManagerPnpm {
		pnpm := ctx.Resolver.Default("pnpm", "latest")

		lockfile, err := ctx.App.ReadFile("pnpm-lock.yaml")
		if err == nil {
			if strings.HasPrefix(lockfile, "lockfileVersion: 5.3") {
				ctx.Resolver.Version(pnpm, "6", "pnpm-lock.yaml")
			} else if strings.HasPrefix(lockfile, "lockfileVersion: 5.4") {
				ctx.Resolver.Version(pnpm, "7", "pnpm-lock.yaml")
			} else if strings.HasPrefix(lockfile, "lockfileVersion: '6.0'") {
				ctx.Resolver.Version(pnpm, "8", "pnpm-lock.yaml")
			}
		}
	}

	// Yarn
	if packageManager == PackageManagerYarn1 {
		ctx.Resolver.Default("yarn", "1")
		ctx.AddAptPackage("tar")
		ctx.AddAptPackage("gpg")
	} else if packageManager == PackageManagerYarn2 {
		ctx.Resolver.Default("yarn", "2")
	}

	return nil
}

func (p *NodeProvider) usesCorepack(packageJson *PackageJson) bool {
	return packageJson.PackageManager != nil
}

func (p *NodeProvider) getPackageManager(app *app.App) PackageManager {
	packageManager := PackageManagerNpm

	if app.HasMatch("pnpm-lock.yaml") {
		packageManager = PackageManagerPnpm
	} else if app.HasMatch("bun.lockb") || app.HasMatch("bun.lock") {
		packageManager = PackageManagerBun
	} else if app.HasMatch(".yarnrc.yml") || app.HasMatch(".yarnrc.yaml") {
		packageManager = PackageManagerYarn2
	} else if app.HasMatch("yarn.lock") {
		packageManager = PackageManagerYarn1
	}

	return packageManager
}

func (p *NodeProvider) getPackageJson(app *app.App) (*PackageJson, error) {
	if !app.HasMatch("package.json") {
		return nil, nil
	}

	var packageJson PackageJson
	err := app.ReadJSON("package.json", &packageJson)
	if err != nil {
		return nil, fmt.Errorf("error reading package.json: %w", err)
	}

	return &packageJson, nil
}
