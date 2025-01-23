package php

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/railwayapp/railpack-go/core/generate"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/stretchr/objx"
)

const (
	DEFAULT_PHP_VERSION = "8.4"
)

type PhpProvider struct{}

func (p *PhpProvider) Name() string {
	return "php"
}

func (p *PhpProvider) Plan(ctx *generate.GenerateContext) (bool, error) {
	hasPhp := ctx.App.HasMatch("index.php") ||
		ctx.App.HasMatch("composer.json")

	if !hasPhp {
		return false, nil
	}

	if err := p.packages(ctx); err != nil {
		return false, err
	}

	if err := p.install(ctx); err != nil {
		return false, err
	}

	ctx.Start.Paths = append(ctx.Start.Paths, ".")

	if err := p.start(ctx); err != nil {
		return false, err
	}

	return false, nil
}

func (p *PhpProvider) start(ctx *generate.GenerateContext) error {
	ctx.Start.Paths = []string{"."}
	ctx.Start.Command = "php-fpm & nginx -c /etc/nginx/nginx.conf"

	return nil
}

func (p *PhpProvider) install(ctx *generate.GenerateContext) error {
	nginxPackages := ctx.NewAptStep("packages:nginx")
	nginxPackages.Packages = []string{"nginx", "git", "zip", "unzip"}

	if _, err := p.readComposerJson(ctx); err == nil {
		install := ctx.NewCommandStep("install")
		install.AddCommands([]plan.Command{
			plan.CopyCommand{Image: "composer:latest", Src: "/usr/bin/composer", Dst: "/usr/bin/composer"},
			plan.NewCopyCommand("."),
			plan.NewExecCommand("composer install --ignore-platform-reqs"),
		})

		install.DependsOn = []string{"packages"}
	}

	nginxSetup := ctx.NewCommandStep("nginx:setup")
	nginxSetup.DependsOn = []string{"packages:nginx"}

	nginxSetup.AddCommands([]plan.Command{
		plan.NewFileCommand("/etc/nginx/nginx.conf", "nginx.conf", "create nginx config"),
	})

	tmpl, err := template.New("nginx.conf").Parse(nginxConf)
	if err != nil {
		return fmt.Errorf("failed to parse nginx.conf template: %w", err)
	}

	data := map[string]interface{}{
		"PORT":                  "80", // Default port
		"NIXPACKS_PHP_ROOT_DIR": "/app",
		"IS_LARAVEL":            false,
		"nginx": map[string]string{
			"conf": "/etc/nginx",
		},
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute nginx template: %w", err)
	}

	// Add the rendered config as an asset
	nginxSetup.Assets["nginx.conf"] = buf.String()

	fmt.Printf("Compiled nginx.conf: %s\n", buf.String())

	// Set the start command to run both php-fpm and nginx
	// ctx.Start.Command = "php-fpm & nginx -g 'daemon off;'"

	return nil
}

func (p *PhpProvider) packages(ctx *generate.GenerateContext) error {
	imageStep := ctx.NewImageStep("packages", func(options *generate.BuildStepOptions) string {
		if phpVersion, ok := options.ResolvedPackages["php"]; ok && phpVersion.ResolvedVersion != nil {
			return getPhpImage(*phpVersion.ResolvedVersion)
		}

		// Return the default if we were not able to resolve the version
		return getPhpImage(DEFAULT_PHP_VERSION)
	})

	php := imageStep.Default("php", DEFAULT_PHP_VERSION)

	// Read composer.json to get the PHP version
	if composerJson, err := p.readComposerJson(ctx); err == nil {
		phpVersion := objx.New(composerJson).Get("require.php")
		if phpVersion.IsStr() {
			imageStep.Version(php, phpVersion.Str(), "composer.json > require > php")
		}
	}

	return nil
}

func getPhpImage(phpVersion string) string {
	return fmt.Sprintf("php:%s-fpm-bookworm", phpVersion)
}

func (p *PhpProvider) readComposerJson(ctx *generate.GenerateContext) (map[string]interface{}, error) {
	var composerJson map[string]interface{}
	err := ctx.App.ReadJSON("composer.json", &composerJson)
	if err != nil {
		return nil, err
	}

	return composerJson, nil
}
