package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/railwayapp/railpack-go/cli"
	urfave "github.com/urfave/cli/v3"
)

var verbose bool

func main() {

	cmd := &urfave.Command{
		Name:                  "railpack",
		Usage:                 "Automatically analyze and generate build plans for applications",
		EnableShellCompletion: true,
		Flags: []urfave.Flag{
			&urfave.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Usage:       "Enable verbose logging",
				Value:       false,
				Destination: &verbose,
			},
		},
		Before: func(ctx context.Context, cmd *urfave.Command) (context.Context, error) {
			configureLogging(verbose)

			return ctx, nil
		},
		Commands: []*urfave.Command{
			cli.BuildCommand,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func configureLogging(verbose bool) {
	log.SetTimeFormat("")

	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
