package python

import (
	"regexp"

	"github.com/railwayapp/railpack-go/core/generate"
	"github.com/railwayapp/railpack-go/core/plan"
)

const (
	DEFAULT_PYTHON_VERSION = "3.13"
)

type PythonProvider struct{}

func (p *PythonProvider) Name() string {
	return "python"
}

func (p *PythonProvider) Plan(ctx *generate.GenerateContext) (bool, error) {
	hasPython := ctx.App.HasMatch("main.py") ||
		p.hasRequirements(ctx) ||
		p.hasPyproject(ctx) ||
		p.hasPoetry(ctx) ||
		p.hasPdm(ctx)

	if !hasPython {
		return false, nil
	}

	if err := p.packages(ctx); err != nil {
		return false, err
	}

	if err := p.install(ctx); err != nil {
		return false, err
	}

	return false, nil
}

func (p *PythonProvider) install(ctx *generate.GenerateContext) error {
	install := ctx.NewProviderStep("install")

	hasRequirements := p.hasRequirements(ctx)
	hasPyproject := p.hasPyproject(ctx)
	hasPipfile := p.hasPipfile(ctx)
	hasPoetry := p.hasPoetry(ctx)
	hasPdm := p.hasPdm(ctx)
	hasUv := p.hasUv(ctx)

	if hasRequirements {
		install.AddCommands([]plan.Command{
			plan.NewCopyCommand("requirements.txt"),
			plan.NewExecCommand("pip install -r requirements.txt"),
		})
	} else if hasPyproject && hasPoetry {
		install.AddCommands([]plan.Command{
			plan.NewCopyCommand("pyproject.toml"),
			plan.NewCopyCommand("poetry.lock"),
			plan.NewExecCommand("poetry install --no-dev --no-interaction --no-ansi"),
		})
	} else if hasPyproject && hasPdm {
		install.AddCommands([]plan.Command{
			plan.NewCopyCommand("pyproject.toml"),
			plan.NewCopyCommand("pdm.lock"),
			plan.NewExecCommand("pdm install --prod"),
		})
	} else if hasPyproject && hasUv {
		install.AddCommands([]plan.Command{
			plan.NewCopyCommand("pyproject.toml"),
			plan.NewCopyCommand("uv.lock"),
			plan.NewExecCommand("uv sync --no-dev --frozen"),
		})
	} else if hasPipfile {
		install.AddCommands([]plan.Command{
			plan.NewCopyCommand("Pipfile"),
		})

		if ctx.App.HasMatch("Pipfile.lock") {
			install.AddCommands([]plan.Command{
				plan.NewCopyCommand("Pipfile.lock"),
				plan.NewExecCommand("pipenv install --deploy"),
			})
		} else {
			install.AddCommands([]plan.Command{
				plan.NewExecCommand("pipenv install --skip-lock"),
			})
		}

	}

	return nil
}

func (p *PythonProvider) packages(ctx *generate.GenerateContext) error {
	packages := ctx.NewPackageStep("packages")

	python := packages.Default("python", DEFAULT_PYTHON_VERSION)

	if envVersion := ctx.Env.GetConfigVariable("PYTHON_VERSION"); envVersion != "" {
		packages.Version(python, envVersion, "RAILPACK_PYTHON_VERSION")
	}

	if versionFile, err := ctx.App.ReadFile(".python-version"); err == nil {
		packages.Version(python, string(versionFile), ".python-version")
	}

	if runtimeFile, err := ctx.App.ReadFile("runtime.txt"); err == nil {
		packages.Version(python, string(runtimeFile), "runtime.txt")
	}

	if pipfileVersion := parseVersionFromPipfile(ctx); pipfileVersion != "" {
		packages.Version(python, pipfileVersion, "Pipfile")
	}

	if p.hasPoetry(ctx) {
		packages.Default("poetry", "latest")
	}

	if p.hasUv(ctx) {
		packages.Default("uv", "latest")
	}

	if p.hasPdm(ctx) {
		packages.Default("pdm", "latest")
	}

	return nil
}

var pipfileVersionRegex = regexp.MustCompile(`(python_version|python_full_version)\s*=\s*['"]([0-9.]*)"?`)

func parseVersionFromPipfile(ctx *generate.GenerateContext) string {
	pipfile, err := ctx.App.ReadFile("Pipfile")
	if err != nil {
		return ""
	}

	matches := pipfileVersionRegex.FindStringSubmatch(string(pipfile))

	if len(matches) > 2 {
		return matches[2]
	}
	return ""
}

func (p *PythonProvider) hasRequirements(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("requirements.txt")
}

func (p *PythonProvider) hasPyproject(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("pyproject.toml")
}

func (p *PythonProvider) hasPipfile(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("Pipfile")
}

func (p *PythonProvider) hasPoetry(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("poetry.lock")
}

func (p *PythonProvider) hasPdm(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("pdm.lock")
}

func (p *PythonProvider) hasUv(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("uv.lock")
}
