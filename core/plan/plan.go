package plan

type BuildPlan struct {
	Variables map[string]string `json:"variables,omitempty"`
	Steps     []Step            `json:"steps,omitempty"`
	Packages  *PlanPackages     `json:"packages,omitempty"`
}

func NewBuildPlan() *BuildPlan {
	return &BuildPlan{
		Variables: map[string]string{},
		Steps:     []Step{},
		Packages:  NewPlanPackages(),
	}
}

func (p *BuildPlan) AddStep(step Step) {
	p.Steps = append(p.Steps, step)
}
