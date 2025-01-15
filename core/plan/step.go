package plan

type Step struct {
	Name string `json:"name"`

	DependsOn []string  `json:"depends_on,omitempty"`
	Commands  []Command `json:"commands,omitempty"`
}

func NewStep(name string) *Step {
	return &Step{
		Name:      name,
		DependsOn: make([]string, 0),
		Commands:  make([]Command, 0),
	}
}

func (s *Step) DependOn(name string) {
	s.DependsOn = append(s.DependsOn, name)
}

func (s *Step) AddCommands(commands []Command) {
	s.Commands = append(s.Commands, commands...)
}

func MergeSteps(steps ...*Step) *Step {
	if len(steps) == 0 {
		return nil
	}

	result := &Step{
		Name:      steps[0].Name,
		DependsOn: make([]string, 0),
		Commands:  make([]Command, 0),
	}

	for _, step := range steps {
		result.DependsOn = append(result.DependsOn, step.DependsOn...)
		result.Commands = append(result.Commands, step.Commands...)
	}

	return result
}
