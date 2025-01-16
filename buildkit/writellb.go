package buildkit

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/railwayapp/railpack-go/core/plan"
)

func WriteLLB(plan *plan.BuildPlan) error {
	ctx := appcontext.Context()

	llbState, image, err := ConvertPlanToLLB(plan)
	if err != nil {
		return fmt.Errorf("error converting plan to LLB: %w", err)
	}

	imageBytes, err := json.Marshal(image)
	if err != nil {
		return fmt.Errorf("error marshalling image: %w", err)
	}

	log.Debugf("Image config: %+v", image)

	st, err := llbState.WithImageConfig(imageBytes)
	if err != nil {
		return fmt.Errorf("error setting image config: %w", err)
	}

	dt, err := st.Marshal(ctx, llb.LinuxAmd64)
	if err != nil {
		return fmt.Errorf("error marshaling LLB state: %w", err)
	}

	err = llb.WriteTo(dt, os.Stdout)
	if err != nil {
		return fmt.Errorf("error writing LLB state to stdout: %w", err)
	}

	return nil
}
