package generate

import "github.com/railwayapp/railpack-go/core/plan"

type CommandStepBuilder struct {
	DisplayName string
	DependsOn   []string
	Commands    []plan.Command
	Outputs     []string
}

func (c *GenerateContext) NewCommandStep(name string) *CommandStepBuilder {
	step := &CommandStepBuilder{
		DisplayName: name,
		DependsOn:   []string{PackagesStepName},
		Commands:    []plan.Command{},
		Outputs:     []string{},
	}

	c.Steps = append(c.Steps, step)

	return step
}

func (b *CommandStepBuilder) DependOn(name string) {
	b.DependsOn = append(b.DependsOn, name)
}

func (b *CommandStepBuilder) AddCommand(command plan.Command) {
	b.Commands = append(b.Commands, command)
}

func (b *CommandStepBuilder) AddCommands(commands []plan.Command) {
	b.Commands = append(b.Commands, commands...)
}

func (b *CommandStepBuilder) Build(options *BuildStepOptions) (*plan.Step, error) {
	step := plan.NewStep(b.DisplayName)

	step.DependsOn = b.DependsOn
	step.Commands = b.Commands
	step.Outputs = b.Outputs

	return step, nil
}
