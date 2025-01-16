package plan

import "encoding/json"

type Step struct {
	Name string `json:"name"`

	DependsOn []string          `json:"depends_on,omitempty"`
	Commands  []Command         `json:"commands,omitempty"`
	Outputs   []string          `json:"outputs,omitempty"`
	Assets    map[string]string `json:"assets,omitempty"`
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

func (s *Step) UnmarshalJSON(data []byte) error {
	type Alias Step
	aux := &struct {
		Commands []json.RawMessage `json:"commands"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	s.Commands = make([]Command, len(aux.Commands))
	for i, rawCmd := range aux.Commands {
		cmd, err := UnmarshalCommand(rawCmd)
		if err != nil {
			return err
		}
		s.Commands[i] = cmd
	}

	return nil
}
