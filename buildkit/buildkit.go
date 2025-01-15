package buildkit

import (
	"fmt"
	"os"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/appcontext"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/railwayapp/railpack-go/core/plan"
)

type Image struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Config       Config `json:"config"`
}

type Config struct {
	specs.ImageConfig
}

func WriteLLB(plan *plan.BuildPlan) error {
	ctx := appcontext.Context()

	llbState, err := ConvertPlanToLLB(plan)
	if err != nil {
		return fmt.Errorf("error converting plan to LLB: %w", err)
	}

	dt, err := llbState.Marshal(ctx, llb.LinuxAmd64)
	if err != nil {
		return fmt.Errorf("error marshaling LLB state: %w", err)
	}
	llb.WriteTo(dt, os.Stdout)

	return nil
}
