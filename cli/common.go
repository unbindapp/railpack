package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/unbindapp/railpack/core"
	a "github.com/unbindapp/railpack/core/app"
	"github.com/unbindapp/railpack/internal/utils"
	"github.com/urfave/cli/v3"
)

var Version string // This will be set by main

func commonPlanFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "env",
			Usage: "environment variables to set",
		},
		&cli.StringSliceFlag{
			Name:  "previous",
			Usage: "versions of packages used for previous builds (e.g. 'package@version')",
		},
		&cli.StringFlag{
			Name:  "build-cmd",
			Usage: "build command to use",
		},
		&cli.StringFlag{
			Name:  "start-cmd",
			Usage: "start command to use",
		},
		&cli.StringFlag{
			Name:  "config-file",
			Usage: "path to config file to use",
		},
		&cli.BoolFlag{
			Name:  "error-missing-start",
			Usage: "error if no start command is found",
		},
	}
}

func GenerateBuildResultForCommand(cmd *cli.Command) (*core.BuildResult, *a.App, *a.Environment, error) {
	directory := cmd.Args().First()

	if directory == "" {
		return nil, nil, nil, cli.Exit("directory argument is required", 1)
	}

	app, err := a.NewApp(directory)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating app: %w", err)
	}

	log.Debugf("Building %s", app.Source)

	envsArgs := cmd.StringSlice("env")

	env, err := a.FromEnvs(envsArgs)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating env: %w", err)
	}

	previousVersions := utils.ParsePackageWithVersion(cmd.StringSlice("previous"))

	generateOptions := &core.GenerateBuildPlanOptions{
		RailpackVersion:          Version,
		BuildCommand:             cmd.String("build-cmd"),
		StartCommand:             cmd.String("start-cmd"),
		PreviousVersions:         previousVersions,
		ConfigFilePath:           cmd.String("config-file"),
		ErrorMissingStartCommand: cmd.Bool("error-missing-start"),
	}

	buildResult := core.GenerateBuildPlan(app, env, generateOptions)

	return buildResult, app, env, nil
}
