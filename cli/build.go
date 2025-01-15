package cli

import (
	"context"
	"encoding/json"

	"github.com/charmbracelet/log"
	"github.com/railwayapp/railpack-go/buildkit"
	"github.com/railwayapp/railpack-go/core"
	a "github.com/railwayapp/railpack-go/core/app"
	"github.com/urfave/cli/v3"
)

var BuildCommand = &cli.Command{
	Name:                  "build",
	Aliases:               []string{"b"},
	Usage:                 "generate a build plan for a directory",
	ArgsUsage:             "DIRECTORY",
	EnableShellCompletion: true,
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "env",
			Usage: "environment variables to set",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		directory := cmd.Args().First()

		if directory == "" {
			return cli.Exit("directory argument is required", 1)
		}

		app, err := a.NewApp(directory)
		if err != nil {
			return cli.Exit(err, 1)
		}

		log.Debugf("Building %s", app.Source)

		envsArgs := cmd.StringSlice("env")
		env, err := a.FromEnvs(envsArgs)
		if err != nil {
			return cli.Exit(err, 1)
		}

		buildResult, err := core.GenerateBuildPlan(app, env, &core.GenerateBuildPlanOptions{})
		if err != nil {
			return cli.Exit(err, 1)
		}

		serializedPlan, err := json.MarshalIndent(buildResult, "", "  ")
		if err != nil {
			return cli.Exit(err, 1)
		}

		log.Infof("Plan:\n %s", string(serializedPlan))

		err = buildkit.WriteLLB(buildResult.Plan)
		if err != nil {
			return cli.Exit(err, 1)
		}

		return nil
	},
}
