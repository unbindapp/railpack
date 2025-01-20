package cli

import (
	"context"
	"encoding/json"
	"os"

	"github.com/railwayapp/railpack-go/buildkit"
	"github.com/urfave/cli/v3"
)

var BuildCommand = &cli.Command{
	Name:                  "build",
	Aliases:               []string{"b"},
	Usage:                 "build an image with BuildKit",
	ArgsUsage:             "DIRECTORY",
	EnableShellCompletion: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "name of the image to build",
		},
		&cli.StringSliceFlag{
			Name:  "env",
			Usage: "environment variables to set",
		},
		&cli.BoolFlag{
			Name:  "llb",
			Usage: "output the LLB plan to stdout instead of building the image",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "output",
			Usage: "output the final filesystem to a local directory",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		buildResult, app, err := GenerateBuildResultForCommand(cmd)
		if err != nil {
			return cli.Exit(err, 1)
		}

		serializedPlan, err := json.MarshalIndent(buildResult, "", "  ")
		if err != nil {
			return cli.Exit(err, 1)
		}

		os.Stdout.Write(serializedPlan)
		os.Stdout.Write([]byte("\n"))

		err = buildkit.BuildWithBuildkitClient(app.Source, buildResult.Plan, buildkit.BuildWithBuildkitClientOptions{
			ImageName: cmd.String("name"),
			DumpLLB:   cmd.Bool("llb"),
			OutputDir: cmd.String("output"),
		})
		if err != nil {
			return cli.Exit(err, 1)
		}

		return nil
	},
}
