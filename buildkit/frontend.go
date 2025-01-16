package buildkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/containerd/platforms"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend/gateway/client"
	gw "github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/system"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/railwayapp/railpack-go/core/plan"
)

const (
	localNameDockerfile = "dockerfile"
	defaultRailpackPlan = "rpk"
)

func StartFrontend() {
	log.Info("Starting frontend")

	ctx := appcontext.Context()
	if err := gw.RunFromEnvironment(ctx, Build); err != nil {
		log.Error("error: %+v\n", err)
		os.Exit(1)
	}
}

func Build(ctx context.Context, c client.Client) (*client.Result, error) {
	platform := platforms.DefaultSpec()

	plan, err := readRailpackPlan(ctx, c)
	if err != nil {
		return nil, err
	}

	planLength := len(plan.Steps)

	state := llb.Image("ubuntu:noble").
		Dir("/app").
		Run(llb.Shlex(fmt.Sprintf("sh -c 'echo %d > /app/length.txt'", planLength))).
		Run(llb.Shlex("touch /app/test.txt"))

	def, err := state.Marshal(ctx)
	if err != nil {
		return nil, err
	}

	// Configure the image and how it will start
	imageConfig := Image{
		Image: specs.Image{
			Platform: platform,
		},
		Config: specs.ImageConfig{
			Entrypoint: []string{"/bin/bash", "-c"},
			Cmd:        []string{"npm run start"},
			Env: []string{
				"PATH=/mise/shims:" + system.DefaultPathEnvUnix,
				"MISE_DATA_DIR=/mise",
				"MISE_CONFIG_DIR=/mise",
				"MISE_INSTALL_PATH=/usr/local/bin/mise",
			},
			WorkingDir: "/app",
		},
	}
	imageConfigStr, _ := json.Marshal(imageConfig)

	res, err := c.Solve(ctx, client.SolveRequest{
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, err
	}

	res.AddMeta(exptypes.ExporterImageConfigKey, imageConfigStr)

	return res, nil
}

func readRailpackPlan(ctx context.Context, c client.Client) (*plan.BuildPlan, error) {
	opts := c.BuildOpts().Opts
	filename := opts["filename"]
	if filename == "" {
		filename = defaultRailpackPlan
	}

	fileContents, err := readFile(ctx, c, filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read railpack plan")
	}

	plan := plan.NewBuildPlan()
	err = json.Unmarshal([]byte(fileContents), plan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse railpack plan")
	}

	return plan, nil
}

func readFile(ctx context.Context, c client.Client, filename string) (string, error) {
	// Create a Local source for the dockerfile
	src := llb.Local("context",
		llb.FollowPaths([]string{filename}),
		llb.SessionID(c.BuildOpts().SessionID),
		llb.WithCustomName("load build definition from "+filename),
	)

	srcDef, err := src.Marshal(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal local source")
	}

	res, err := c.Solve(ctx, client.SolveRequest{
		Definition: srcDef.ToPB(),
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve dockerfile")
	}

	ref, err := res.SingleRef()
	if err != nil {
		return "", err
	}

	content, err := ref.ReadFile(ctx, client.ReadRequest{
		Filename: filename,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to read file")
	}

	fileContents := string(content)

	return fileContents, nil
}
