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
}

type BuildGraph struct {
	Nodes     map[string]*Node
	BaseState *llb.State
}

func NewBuildGraph(plan *plan.BuildPlan, baseState *llb.State) (*BuildGraph, error) {
	graph := &BuildGraph{
		Nodes:     make(map[string]*Node),
		BaseState: baseState,
	}

	// Create a node for each step
	for i := range plan.Steps {
		step := &plan.Steps[i]
		graph.Nodes[step.Name] = &Node{
			Step:      step,
			Parents:   make([]*Node, 0),
			Children:  make([]*Node, 0),
			Processed: false,
		}
	}

	// Add dependencies to each node
	for _, node := range graph.Nodes {
		for _, depName := range node.Step.DependsOn {
			if depNode, exists := graph.Nodes[depName]; exists {
				node.Parents = append(node.Parents, depNode)
				depNode.Children = append(depNode.Children, node)
			}
		}
	}

	graph.PrintGraph()

	return graph, nil
}

func (g *BuildGraph) GenerateLLB() (*llb.State, error) {
	// Get processing order using topological sort
	order, err := g.getProcessingOrder()
	if err != nil {
		return nil, err
	}

	// Process all nodes in order
	for _, node := range order {
		if err := g.processNode(node); err != nil {
			return nil, err
		}
	}

	// Find all leaf nodes and get their states
	var leafStates []llb.State
	var leafStepNames []string
	for _, node := range g.Nodes {
		if len(node.Children) == 0 && node.State != nil {
			leafStates = append(leafStates, *node.State)
			leafStepNames = append(leafStepNames, node.Step.Name)
		}
	}

	// If no leaf states, return base state
	if len(leafStates) == 0 {
		return g.BaseState, nil
	}

	// If only one leaf state, return it
	if len(leafStates) == 1 {
		return &leafStates[0], nil
	}

	// Merge all leaf states
	mergeName := fmt.Sprintf("merging steps: %s", strings.Join(leafStepNames, ", "))
	result := llb.Merge(leafStates, llb.WithCustomName(mergeName))
	return &result, nil
}

func (g *BuildGraph) processNode(node *Node) error {
	// If already processed, we're done
	if node.Processed {
		return nil
	}

	// Check if all parents are processed
	for _, parent := range node.Parents {
		if !parent.Processed {
			// If this node is marked in-progress, we have a dependency violation
			if node.InProgress {
				fmt.Printf("Dependencies for %s:\n", node.Step.Name)
				for _, dep := range node.Parents {
					fmt.Printf("  %s (processed: %v, in-progress: %v)\n",
						dep.Step.Name, dep.Processed, dep.InProgress)
				}
				return fmt.Errorf("Dependency violation: %s waiting for unprocessed parent %s",
					node.Step.Name, parent.Step.Name)
			}

			// Mark this node as in-progress and process the parent
			node.InProgress = true
			if err := g.processNode(parent); err != nil {
				node.InProgress = false
				return err
			}
			node.InProgress = false
		}
	}

	// Determine the state to build upon
	var currentState *llb.State

	if len(node.Parents) == 0 {
		currentState = g.BaseState
	} else if len(node.Parents) == 1 {
		// If only one parent, use its state directly
		currentState = node.Parents[0].State
	} else {
		// If multiple parents, merge their states
		parentStates := make([]llb.State, len(node.Parents))
		mergeStepNames := make([]string, len(node.Parents))

		for i, parent := range node.Parents {
			if parent.State == nil {
				return fmt.Errorf("Parent %s of %s has nil state",
					parent.Step.Name, node.Step.Name)
			}
			parentStates[i] = *parent.State
			mergeStepNames[i] = parent.Step.Name
		}

		mergeName := fmt.Sprintf("merging steps: %s", strings.Join(mergeStepNames, ", "))
		merged := llb.Merge(parentStates, llb.WithCustomName(mergeName))
		currentState = &merged
	}

	// Convert this node's step to LLB
	stepState, err := convertStepToLLB2(node.Step, currentState)
	if err != nil {
		return err
	}

	node.State = stepState
	node.Processed = true

	return nil
}

func convertStepToLLB2(step *plan.Step, baseState *llb.State) (*llb.State, error) {
	state := baseState
	for _, cmd := range step.Commands {
		var err error
		state, err = convertCommandToLLB(cmd, state, step)
		if err != nil {
			return nil, err
		}
	}

	if step.Outputs != nil && len(step.Outputs) > 0 {
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

		merged := llb.Merge([]llb.State{*state, result})
		state = &merged
	}

	return state, nil
}

func convertCommandToLLB(cmd plan.Command, state *llb.State, step *plan.Step) (*llb.State, error) {
	switch cmd := cmd.(type) {
	case plan.ExecCommand:
		opts := []llb.RunOption{llb.Shlex(cmd.Cmd)}
		if cmd.CustomName != "" {
			opts = append(opts, llb.WithCustomName(cmd.CustomName))
		}
		s := state.Run(opts...).Root()
		return &s, nil

	case plan.PathCommand:
		// TODO: Build up the path so we are not starting from scratch each time
		s := state.AddEnvf("PATH", "%s:%s", cmd.Path, system.DefaultPathEnvUnix)
		return &s, nil

	case plan.VariableCommand:
		s := state.AddEnv(cmd.Name, cmd.Value)
		return &s, nil

	case plan.CopyCommand:
		src := llb.Local("context")
		s := state.File(llb.Copy(src, cmd.Src, cmd.Dst, &llb.CopyInfo{
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

// getProcessingOrder returns nodes in topological order
func (g *BuildGraph) getProcessingOrder() ([]*Node, error) {
	order := make([]*Node, 0, len(g.Nodes))
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(node *Node) error
	visit = func(node *Node) error {
		if temp[node.Step.Name] {
			return fmt.Errorf("cycle detected: %s", node.Step.Name)
		}
		if visited[node.Step.Name] {
			return nil
		}
		temp[node.Step.Name] = true

		for _, parent := range node.Parents {
			if err := visit(parent); err != nil {
				return err
			}
		}

		delete(temp, node.Step.Name)
		visited[node.Step.Name] = true
		order = append(order, node)
		return nil
	}

	// Start with leaf nodes (nodes with no children)
	for _, node := range g.Nodes {
		if len(node.Children) == 0 {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	// Process any remaining nodes
	for _, node := range g.Nodes {
		if !visited[node.Step.Name] {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	// Reverse the order since we want parents before children
	for i := 0; i < len(order)/2; i++ {
		j := len(order) - 1 - i
		order[i], order[j] = order[j], order[i]
	}

	return order, nil
}

func (g *BuildGraph) PrintGraph() {
	fmt.Println("\nBuild Graph Structure:")
	fmt.Println("=====================")

	for name, node := range g.Nodes {
		fmt.Printf("\nNode: %s\n", name)
		fmt.Printf("  Status:\n")
		fmt.Printf("    Processed: %v\n", node.Processed)
		fmt.Printf("    InProgress: %v\n", node.InProgress)

		fmt.Printf("  Parents (%d):\n", len(node.Parents))
		for _, parent := range node.Parents {
			fmt.Printf("    - %s\n", parent.Step.Name)
		}

		fmt.Printf("  Children (%d):\n", len(node.Children))
		for _, child := range node.Children {
			fmt.Printf("    - %s\n", child.Step.Name)
		}
	}
	fmt.Println("\n=====================")
}
