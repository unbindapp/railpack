package buildkit

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"github.com/railwayapp/railpack-go/core/plan"
)

type Node struct {
	Step       *plan.Step
	BaseState  *llb.State
	DiffState  *llb.State
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
	// Find root nodes
	roots := make([]*Node, 0)
	for _, node := range g.Nodes {
		if len(node.Parents) == 0 {
			roots = append(roots, node)
		}
	}

	for _, root := range roots {
		fmt.Printf("Root: %s\n", root.Step.Name)
	}

	for _, root := range roots {
		if err := g.processNode(root); err != nil {
			return nil, err
		}
	}

	// Verify all nodes were processed
	for _, node := range g.Nodes {
		if !node.Processed {
			return nil, fmt.Errorf("node %s was not processed", node.Step.Name)
		}
	}

	// Find all leaf nodes and get their diffs
	var diffs []*llb.State
	for _, node := range g.Nodes {
		if len(node.Children) == 0 && node.DiffState != nil {
			diffs = append(diffs, node.DiffState)
		}
	}

	// Merge the base state with all the diffs
	if len(diffs) == 0 {
		return g.BaseState, nil
	}

	statesToMerge := make([]llb.State, len(diffs))
	for i, diff := range diffs {
		statesToMerge[i] = *diff
	}
	result := llb.Merge(statesToMerge)
	return &result, nil
}

func (g *BuildGraph) processNode(node *Node) error {
	fmt.Printf("Processing node: %s\n", node.Step.Name)

	if node.InProgress {
		return fmt.Errorf("Circular dependency detected at step %s", node.Step.Name)
	}

	if node.Processed {
		return nil
	}

	node.InProgress = true

	// Process parents
	for _, parent := range node.Parents {
		if err := g.processNode(parent); err != nil {
			return err
		}
	}

	var fullState *llb.State
	if len(node.Parents) == 0 {
		fullState = g.BaseState
	} else {
		parentDifs := make([]*llb.State, len(node.Parents))
		for _, parent := range node.Parents {
			if parent.DiffState != nil {
				parentDifs = append(parentDifs, parent.DiffState)
			}
		}

		statesToMerge := make([]llb.State, 1, len(node.Parents)+1)
		statesToMerge[0] = *g.BaseState
		for _, diff := range parentDifs {
			if diff != nil {
				statesToMerge = append(statesToMerge, *diff)
			}
		}

		merged := llb.Merge(statesToMerge)
		fullState = &merged
	}

	// Convert this node to LLB
	stepState, err := convertStepToLLB2(node.Step, fullState)
	if err != nil {
		return err
	}

	diff := llb.Diff(*fullState, *stepState)
	node.BaseState = fullState
	node.DiffState = &diff
	node.Processed = true
	node.InProgress = false

	// Process all children nodes
	for _, child := range node.Children {
		if err := g.processNode(child); err != nil {
			return err
		}
	}

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
	return state, nil
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
