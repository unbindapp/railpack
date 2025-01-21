package buildkit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	"github.com/railwayapp/railpack-go/core/plan"
)

type Node struct {
	Step       *plan.Step
	State      *llb.State
	Parents    []*Node
	Children   []*Node
	Processed  bool
	InProgress bool

	InputPathList []string
	InputEnvVars  map[string]string

	OutputPathList []string
	OutputEnvVars  map[string]string
}

func (node *Node) convertStepToLLB(baseState *llb.State) (*llb.State, error) {
	step := node.Step
	state := baseState

	// Add commands for input variables and path
	for k, v := range node.InputEnvVars {
		newState, err := node.convertCommandToLLB(plan.VariableCommand{Name: k, Value: v}, state, step)
		if err != nil {
			return nil, err
		}
		state = newState
	}

	for _, path := range node.InputPathList {
		newState, err := node.convertCommandToLLB(plan.PathCommand{Path: path}, state, step)
		if err != nil {
			return nil, err
		}
		state = newState
	}

	// Process the step commands
	for _, cmd := range step.Commands {
		var err error
		state, err = node.convertCommandToLLB(cmd, state, step)
		if err != nil {
			return nil, err
		}
	}

	if len(step.Outputs) > 0 {
		result := llb.Scratch()

		for _, output := range step.Outputs {
			result = result.File(llb.Copy(*state, output, output, &llb.CopyInfo{
				CreateDestPath:      true,
				AllowWildcard:       true,
				AllowEmptyWildcard:  true,
				CopyDirContentsOnly: false,
				FollowSymlinks:      true,
			}))
		}

		merged := llb.Merge([]llb.State{*baseState, result})
		state = &merged
	}

	return state, nil
}

func (node *Node) convertCommandToLLB(cmd plan.Command, state *llb.State, step *plan.Step) (*llb.State, error) {
	switch cmd := cmd.(type) {
	case plan.ExecCommand:
		opts := []llb.RunOption{llb.Shlex(cmd.Cmd)}
		if cmd.CustomName != "" {
			opts = append(opts, llb.WithCustomName(cmd.CustomName))
		}
		s := state.Run(opts...).Root()
		return &s, nil

	case plan.PathCommand:
		node.appendPath(cmd.Path)
		pathList := node.getPathList()
		pathString := strings.Join(pathList, ":")

		s := state.AddEnvf("PATH", "%s:%s", pathString, system.DefaultPathEnvUnix)

		return &s, nil

	case plan.VariableCommand:
		s := state.AddEnv(cmd.Name, cmd.Value)
		node.OutputEnvVars[cmd.Name] = cmd.Value

		return &s, nil

	case plan.CopyCommand:
		src := llb.Local("context")
		s := state.File(llb.Copy(src, cmd.Src, cmd.Dst, &llb.CopyInfo{
			CreateDestPath:      true,
			FollowSymlinks:      true,
			AllowWildcard:       true,
			AllowEmptyWildcard:  true,
			CopyDirContentsOnly: true,
		}))
		return &s, nil

	case plan.FileCommand:
		asset, ok := step.Assets[cmd.Name]
		if !ok {
			return state, fmt.Errorf("asset %q not found", cmd.Name)
		}

		// Create parent directories for the file
		parentDir := filepath.Dir(cmd.Path)
		if parentDir != "/" {
			s := state.File(llb.Mkdir(parentDir, 0755, llb.WithParents(true)))
			state = &s
		}

		fileAction := llb.Mkfile(cmd.Path, 0644, []byte(asset))
		s := state.File(fileAction)
		if cmd.CustomName != "" {
			s = state.File(fileAction, llb.WithCustomName(cmd.CustomName))
		}

		return &s, nil
	}

	return state, nil
}

func (node *Node) getPathList() []string {
	pathList := make([]string, 0)
	pathList = append(pathList, node.InputPathList...)
	pathList = append(pathList, node.OutputPathList...)
	return pathList
}

func (node *Node) appendPath(path string) {
	// Check if path already exists in input or output paths
	for _, existingPath := range node.OutputPathList {
		if existingPath == path {
			return
		}
	}

	node.OutputPathList = append(node.OutputPathList, path)
}
