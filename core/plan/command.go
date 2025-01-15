package plan

// Command represents a command that can be executed
type Command interface {
	commandType() string
}

// ExecCommand represents a command to be executed
type ExecCommand struct {
	Cmd string `json:"cmd"`
}

// GlobalPathCommand represents a global path addition
type GlobalPathCommand struct {
	GlobalPath string `json:"globalPath"`
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

func (e ExecCommand) commandType() string       { return "exec" }
func (g GlobalPathCommand) commandType() string { return "globalPath" }
func (v VariableCommand) commandType() string   { return "variable" }
func (c CopyCommand) commandType() string       { return "copy" }

func NewExecCommand(cmd string) Command {
	return ExecCommand{Cmd: cmd}
}

func NewGlobalPathCommand(path string) Command {
	return GlobalPathCommand{GlobalPath: path}
}

func NewVariableCommand(name, value string) Command {
	return VariableCommand{Name: name, Value: value}
}

func NewCopyCommand(src, dst string) Command {
	return CopyCommand{Src: src, Dst: dst}
}
