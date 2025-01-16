package plan

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Command represents a command that can be executed
type Command interface {
	commandType() string
	MarshalJSON() ([]byte, error)
}

// ExecCommand represents a command to be executed
type ExecCommand struct {
	Cmd string `json:"cmd"`
}

// PathCommand represents a global path addition
type PathCommand struct {
	Path string `json:"path"`
}

// VariableCommand represents a shell variable setting
type VariableCommand struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CopyCommand represents a file copy operation
type CopyCommand struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

func (e ExecCommand) commandType() string     { return "exec" }
func (g PathCommand) commandType() string     { return "globalPath" }
func (v VariableCommand) commandType() string { return "variable" }
func (c CopyCommand) commandType() string     { return "copy" }

func NewExecCommand(cmd string) Command {
	return ExecCommand{Cmd: cmd}
}

func NewPathCommand(path string) Command {
	return PathCommand{Path: path}
}

func NewVariableCommand(name, value string) Command {
	return VariableCommand{Name: name, Value: value}
}

func NewCopyCommand(src, dst string) Command {
	return CopyCommand{Src: src, Dst: dst}
}

func (e ExecCommand) MarshalJSON() ([]byte, error) {
	return json.Marshal("RUN:" + e.Cmd)
}

func (g PathCommand) MarshalJSON() ([]byte, error) {
	return json.Marshal("PATH:" + g.Path)
}

func (v VariableCommand) MarshalJSON() ([]byte, error) {
	return json.Marshal("ENV:" + v.Name + "=" + v.Value)
}

func (c CopyCommand) MarshalJSON() ([]byte, error) {
	return json.Marshal("COPY:" + c.Src + " " + c.Dst)
}

func UnmarshalCommand(data []byte) (Command, error) {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return nil, err
	}

	parts := strings.SplitN(str, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid command format: %s", str)
	}

	cmdType := parts[0]
	payload := parts[1]

	switch cmdType {
	case "RUN":
		return NewExecCommand(payload), nil
	case "PATH":
		return NewPathCommand(payload), nil
	case "ENV":
		envParts := strings.SplitN(payload, "=", 2)
		if len(envParts) != 2 {
			return nil, fmt.Errorf("invalid ENV format: %s", payload)
		}
		return NewVariableCommand(envParts[0], envParts[1]), nil
	case "COPY":
		copyParts := strings.Fields(payload)
		if len(copyParts) != 2 {
			return nil, fmt.Errorf("invalid COPY format: %s", payload)
		}
		return NewCopyCommand(copyParts[0], copyParts[1]), nil
	}
	return nil, fmt.Errorf("unknown command type: %s", cmdType)
}
