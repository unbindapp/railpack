package plan

type BuildPlan struct {
	Variables map[string]string `json:"variables,omitempty"`
	Steps     []Step            `json:"steps,omitempty"`
	Start     Start             `json:"start,omitempty"`
}

type Start struct {
	BaseImage string   `json:"base_image,omitempty"`
	Command   string   `json:"cmd,omitempty"`
	Paths     []string `json:"paths,omitempty"`
}

func NewBuildPlan() *BuildPlan {
	return &BuildPlan{
		Variables: map[string]string{},
		Steps:     []Step{},
		Start:     Start{},
	}
}

func (p *BuildPlan) AddStep(step Step) {
	p.Steps = append(p.Steps, step)
}
