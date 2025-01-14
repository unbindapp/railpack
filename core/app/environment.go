package app

import (
	"os"
	"regexp"
)

type Environment struct {
	Variables map[string]string
}

func NewEnvironment(variables map[string]string) *Environment {
	if variables == nil {
		variables = make(map[string]string)
	}

	return &Environment{Variables: variables}
}

// FromEnvs collects variables from the given environment variable names
func FromEnvs(envs []string) (*Environment, error) {
	env := NewEnvironment(nil)
	re := regexp.MustCompile(`([A-Za-z0-9_-]*)(?:=?)(.*)`)

	for _, e := range envs {
		matches := re.FindStringSubmatch(e)
		if len(matches) < 3 {
			continue
		}

		name := matches[1]
		value := matches[2]

		if value == "" {
			// No value, pull from current environment
			if v, ok := os.LookupEnv(name); ok {
				env.SetVariable(name, v)
			}
		} else {
			// Use provided name, value pair
			env.SetVariable(name, value)
		}
	}

	return env, nil
}

// GetVariable returns the value of the given variable name
func (e *Environment) GetVariable(name string) string {
	return e.Variables[name]
}

// SetVariable stores a variable in the Environment
func (e *Environment) SetVariable(name, value string) {
	e.Variables[name] = value
}
