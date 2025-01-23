package php

import (
	"github.com/railwayapp/railpack-go/core/generate"
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

	ctx.Start.Paths = append(ctx.Start.Paths, ".")

	return false, nil
}

var runtimeAptPackages = []string{
	"libgd-dev",
	"libedit-dev",
	"libicu-dev",
	"libjpeg-dev",
	"libmysqlclient-dev",
	"libonig-dev",
	"libpng-dev",
	"libpq-dev",
	"libreadline-dev",
	"libsqlite3-dev",
	"libssl-dev",
	"libxml2-dev",
	"libzip-dev",
	"openssl",
	"libcurl4-openssl-dev",
}

// These packages (+ runtime) are only needed for building php
// They are not included at runtime
var buildAptPackages = []string{
	"gettext",
	"curl",
	"git",
	"autoconf",
	"build-essential",
	"bison",
	"pkg-config",
	"zlib1g-dev",
	"re2c",
}

func (p *PhpProvider) packages(ctx *generate.GenerateContext) error {
	packages := ctx.NewPackageStep("packages")
	packages.Default("php", DEFAULT_PHP_VERSION)
	packages.SupportingAptPackages = runtimeAptPackages
	packages.SupportingAptPackages = append(packages.SupportingAptPackages, buildAptPackages...)

	runtimePackages := ctx.NewPackageStep("packages:runtime")
	runtimePackages.SupportingAptPackages = runtimeAptPackages
	runtimePackages.Outputs = []string{"/"}

	return nil
}
