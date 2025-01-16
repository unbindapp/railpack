package buildkit

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/moby/buildkit/client"
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/nerdctlcontainer"
	"github.com/moby/buildkit/client/llb"
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

	llbState, image, err := ConvertPlanToLLB(plan)
	if err != nil {
		return fmt.Errorf("error converting plan to LLB: %w", err)
	}

	imageBytes, err := json.Marshal(image)
	if err != nil {
		return fmt.Errorf("error marshalling image: %w", err)
	}

	fmt.Println(string(imageBytes))

	def, err := llbState.Marshal(ctx, llb.LinuxAmd64)
	if err != nil {
		return fmt.Errorf("error marshaling LLB state: %w", err)
	}

	log.Debugf("Local mount: %s", appDir)

	appFS, err := fsutil.NewFS(appDir)
	if err != nil {
		return fmt.Errorf("error creating FS: %w", err)
	}

	res, err := c.Solve(ctx, def, client.SolveOpt{
		LocalMounts: map[string]fsutil.FS{
			"context": appFS,
		},
		Exports: []client.ExportEntry{
			{
				Type: "docker",
				Attrs: map[string]string{
					"name": opts.ImageName,
				},
			},
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to solve: %w", err)
	}

	fmt.Println(res)

	// res.AddMeta(exptypes.ExporterImageConfigKey, imageBytes)

	return nil
}
