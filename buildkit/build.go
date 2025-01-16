package buildkit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/moby/buildkit/client"
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/nerdctlcontainer"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/appcontext"
	_ "github.com/moby/buildkit/util/grpcutil/encoding/proto"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/tonistiigi/fsutil"
)

type BuildWithBuildkitClientOptions struct {
	ImageName string
}

func BuildWithBuildkitClient(appDir string, plan *plan.BuildPlan, opts BuildWithBuildkitClientOptions) error {
	ctx := appcontext.Context()

	imageName := getImageName(appDir)

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

	// Get the buildkit info early so we can ensure we can connect to the buildkit host
	info, err := c.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to get buildkit info: %w", err)
	}

	buildPlatform := determineBuildPlatformFromHost()

	llbState, image, err := ConvertPlanToLLB(plan, ConvertPlanOptions{
		BuildPlatform: buildPlatform,
	})
	if err != nil {
		return fmt.Errorf("error converting plan to LLB: %w", err)
	}

	imageBytes, err := json.Marshal(image)
	if err != nil {
		return fmt.Errorf("error marshalling image: %w", err)
	}

	def, err := llbState.Marshal(ctx, llb.LinuxAmd64)
	if err != nil {
		return fmt.Errorf("error marshaling LLB state: %w", err)
	}

	// Create a pipe to connect buildkit output to docker load
	pipeR, pipeW := io.Pipe()
	defer pipeR.Close()

	ch := make(chan *client.SolveStatus)

	// Pipe the image into `docker load`
	// This is useful so that we don't have to connect the buildkit docker container to the local docker registry
	// We likely don't want to be using this in production, but it is useful for local development
	errCh := make(chan error, 1)
	go func() {
		cmd := exec.Command("docker", "load")
		cmd.Stdin = pipeR
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		errCh <- cmd.Run()
	}()

	progressDone := make(chan bool)
	go func() {
		displayCh := make(chan *client.SolveStatus)
		go func() {
			for s := range ch {
				displayCh <- s
			}
			close(displayCh)
		}()

		display, err := progressui.NewDisplay(os.Stdout, progressui.AutoMode)
		if err != nil {
			log.Error("failed to create progress display", "error", err)
		}

		display.UpdateFrom(ctx, displayCh)
		progressDone <- true
	}()

	appFS, err := fsutil.NewFS(appDir)
	if err != nil {
		return fmt.Errorf("error creating FS: %w", err)
	}

	log.Debugf("Building image for %s with BuildKit %s", buildPlatform.String(), info.BuildkitVersion.Version)

	res, err := c.Solve(ctx, def, client.SolveOpt{
		LocalMounts: map[string]fsutil.FS{
			"context": appFS,
		},
		Exports: []client.ExportEntry{
			{
				Type: client.ExporterDocker,
				Attrs: map[string]string{
					"name":                  imageName,
					"containerimage.config": string(imageBytes),
				},
				Output: func(_ map[string]string) (io.WriteCloser, error) {
					return pipeW, nil
				},
			},
		},
	}, ch)

	if err != nil {
		return fmt.Errorf("failed to solve: %w", err)
	}

	pipeW.Close()

	// Wait for progress monitoring to complete
	<-progressDone

	// Wait for docker load to complete
	if err := <-errCh; err != nil {
		return fmt.Errorf("docker load failed: %w", err)
	}

	fmt.Printf("Result: %+v\n", res)

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
