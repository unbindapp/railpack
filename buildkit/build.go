package buildkit

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/moby/buildkit/client"
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/nerdctlcontainer"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/filesync"
	"github.com/moby/buildkit/util/appcontext"
	_ "github.com/moby/buildkit/util/grpcutil/encoding/proto"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/tonistiigi/fsutil"
)

type BuildWithBuildkitClientOptions struct {
	ImageName string
}

func BuildWithBuildkitClient(appDir string, plan *plan.BuildPlan, opts BuildWithBuildkitClientOptions) error {
	ctx := appcontext.Context()

	imageName := getImageName(appDir)

	// Connect to buildkit daemon
	// If running in Docker, you'll need the address of your buildkit container

	buildkitHost := os.Getenv("BUILDKIT_HOST")
	if buildkitHost == "" {
		log.Error("BUILDKIT_HOST environment variable is not set")
		return fmt.Errorf("BUILDKIT_HOST environment variable is not set")
	}

	log.Debugf("Connecting to buildkit host: %s", buildkitHost)

	c, err := client.New(ctx, buildkitHost)
	if err != nil {
		return fmt.Errorf("failed to connect to buildkit: %w", err)
	}
	defer c.Close()

	info, err := c.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to get buildkit info: %w", err)
	}

	log.Debugf("Buildkit version: %s", info.BuildkitVersion.Version)

	// llbState, image, err := ConvertPlanToLLB(plan)
	// if err != nil {
	// 	return fmt.Errorf("error converting plan to LLB: %w", err)
	// }

	// imageBytes, err := json.Marshal(image)
	// if err != nil {
	// 	return fmt.Errorf("error marshalling image: %w", err)
	// }

	// fmt.Println(string(imageBytes))

	llbState := llb.Image("ubuntu:noble")

	def, err := llbState.Marshal(ctx, llb.LinuxAmd64)
	if err != nil {
		return fmt.Errorf("error marshaling LLB state: %w", err)
	}

	outputFS, err := fsutil.NewFS(appDir)
	if err != nil {
		return fmt.Errorf("error creating FS: %w", err)
	}

	res, err := c.Solve(ctx, def, client.SolveOpt{
		Exports: []client.ExportEntry{
			// {
			// 	Type:      "local",
			// 	OutputDir: "output",
			// },
			// {
			// 	Type: "docker",
			// 	Attrs: map[string]string{
			// 		"name": opts.ImageName,
			// 	},
			// },
			{
				Type: client.ExporterDocker,
				Attrs: map[string]string{
					"name": imageName,
					// "containerimage.config": string(imgJSON),
				},
				Output: func(_ map[string]string) (io.WriteCloser, error) {
					return os.Stdout, nil
				},
			},
		},
		Session: []session.Attachable{
			filesync.NewFSSyncProvider(filesync.StaticDirSource{
				"output": outputFS,
			}),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to solve: %w", err)
	}

	fmt.Println(res)

	// res.AddMeta(exptypes.ExporterImageConfigKey, imageBytes)

	return nil
}

func getImageName(appDir string) string {
	parts := strings.Split(appDir, string(os.PathSeparator))
	name := parts[len(parts)-1]
	if name == "" {
		name = "railpack-app" // Fallback if path ends in separator
	}
	return name
}
