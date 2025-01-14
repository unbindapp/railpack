package plan

type Step struct {
	Name string `json:"name"`

	DependsOn []string `json:"depends_on,omitempty"`
	Commands  []string `json:"commands,omitempty"`
}

func MergeSteps(steps ...*Step) *Step {
	if len(steps) == 0 {
		return nil
	}

	result := &Step{
		Name:      steps[0].Name,
		DependsOn: make([]string, 0),
		Commands:  make([]string, 0),
	}

	for _, step := range steps {
		result.DependsOn = append(result.DependsOn, step.DependsOn...)
		result.Commands = append(result.Commands, step.Commands...)
	}

	return result
}
