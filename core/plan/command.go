package plan

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Command represents a command that can be executed
type Command interface {
	CommandType() string
	MarshalJSON() ([]byte, error)
}

// ExecCommand represents a command to be executed
type ExecCommand struct {
	Cmd        string `json:"cmd"`
	CustomName string `json:"custom_name,omitempty"`
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

// FileCommand represents a file creation operation
type FileCommand struct {
	Path string `json:"path"`

	// The name of the file in the build step assets
	Name string `json:"name"`

	CustomName string `json:"custom_name,omitempty"`
}

func (e ExecCommand) CommandType() string     { return "exec" }
func (g PathCommand) CommandType() string     { return "globalPath" }
func (v VariableCommand) CommandType() string { return "variable" }
func (c CopyCommand) CommandType() string     { return "copy" }
func (f FileCommand) CommandType() string     { return "file" }

func NewExecCommand(cmd string, customName ...string) Command {
	exec := ExecCommand{Cmd: cmd}
	if len(customName) > 0 {
		exec.CustomName = customName[0]
	}
	return exec
}

func NewPathCommand(path string, customName ...string) Command {
	pathCmd := PathCommand{Path: path}
	return pathCmd
}

func NewVariableCommand(name, value string, customName ...string) Command {
	variableCmd := VariableCommand{Name: name, Value: value}
	return variableCmd
}

func NewCopyCommand(src string, dst ...string) Command {
	dstPath := src
	if len(dst) > 0 {
		dstPath = dst[0]
	}

	copyCmd := CopyCommand{Src: src, Dst: dstPath}
	return copyCmd
}

func NewFileCommand(path, name string, customName ...string) Command {
	fileCmd := FileCommand{Path: path, Name: name}
	if len(customName) > 0 {
		fileCmd.CustomName = customName[0]
	}
	return fileCmd
}

func (e ExecCommand) MarshalJSON() ([]byte, error) {
	prefix := "RUN"
	if e.CustomName != "" {
		prefix += "#" + e.CustomName
	}
	return json.Marshal(prefix + ":" + e.Cmd)
}

func (g PathCommand) MarshalJSON() ([]byte, error) {
	prefix := "PATH"
	return json.Marshal(prefix + ":" + g.Path)
}

func (v VariableCommand) MarshalJSON() ([]byte, error) {
	prefix := "ENV"
	return json.Marshal(prefix + ":" + v.Name + "=" + v.Value)
}

func (c CopyCommand) MarshalJSON() ([]byte, error) {
	prefix := "COPY"
	return json.Marshal(prefix + ":" + c.Src + " " + c.Dst)
}

func (f FileCommand) MarshalJSON() ([]byte, error) {
	prefix := "FILE"
	if f.CustomName != "" {
		prefix += "#" + f.CustomName
	}
	return json.Marshal(prefix + ":" + f.Path + " " + f.Name)
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

	prefix := parts[0]
	payload := parts[1]

	// Split prefix into command type and custom name
	prefixParts := strings.SplitN(prefix, "#", 2)
	cmdType := prefixParts[0]
	customName := ""
	if len(prefixParts) > 1 {
		customName = prefixParts[1]
	}

	switch cmdType {
	case "RUN":
		return NewExecCommand(payload, customName), nil
	case "PATH":
		return NewPathCommand(payload, customName), nil
	case "ENV":
		envParts := strings.SplitN(payload, "=", 2)
		if len(envParts) != 2 {
			return nil, fmt.Errorf("invalid ENV format: %s", payload)
		}
		return NewVariableCommand(envParts[0], envParts[1], customName), nil
	case "COPY":
		copyParts := strings.Fields(payload)
		if len(copyParts) != 2 {
			return nil, fmt.Errorf("invalid COPY format: %s", payload)
		}
		return NewCopyCommand(copyParts[0], copyParts[1], customName), nil
	case "FILE":
		fileParts := strings.Fields(payload)
		if len(fileParts) != 2 {
			return nil, fmt.Errorf("invalid FILE format: %s", payload)
		}
		return NewFileCommand(fileParts[0], fileParts[1], customName), nil
	}
	return nil, fmt.Errorf("unknown command type: %s", cmdType)
}
