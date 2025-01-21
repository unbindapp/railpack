package node

import (
	"fmt"

	a "github.com/railwayapp/railpack-go/core/app"
)

const (
	PackageManagerNpm   PackageManager = "npm"
	PackageManagerPnpm  PackageManager = "pnpm"
	PackageManagerBun   PackageManager = "bun"
	PackageManagerYarn1 PackageManager = "yarn1"
	PackageManagerYarn2 PackageManager = "yarn2"
)

func (p PackageManager) Name() string {
	switch p {
	case PackageManagerNpm:
		return "npm"
	case PackageManagerPnpm:
		return "pnpm"
	case PackageManagerBun:
		return "bun"
	case PackageManagerYarn1, PackageManagerYarn2:
		return "yarn"
	default:
		return ""
	}
}

func (p PackageManager) InstallDeps(app *a.App) string {
	switch p {
	case PackageManagerNpm:
		hasLockfile := app.HasMatch("package-lock.json")
		if hasLockfile {
			return "npm ci"
		}
		return "npm install"
	case PackageManagerPnpm:
		return "pnpm install --frozen-lockfile"
	case PackageManagerBun:
		return "bun i --no-save"
	case PackageManagerYarn1:
		return "yarn install --frozen-lockfile"
	case PackageManagerYarn2:
		return "yarn install --check-cache"
	default:
		return ""
	}
}

func (p PackageManager) RunCmd(cmd string) string {
	return fmt.Sprintf("%s run %s", p.Name(), cmd)
}
