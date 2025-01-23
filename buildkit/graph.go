package buildkit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
	"github.com/railwayapp/railpack-go/core/plan"
)

type BuildGraph struct {
	Nodes      map[string]*Node
	BaseState  *llb.State
	CacheStore *BuildKitCacheStore
	Plan       *plan.BuildPlan
}

type BuildGraphOutput struct {
	State    *llb.State
	PathList []string
	EnvVars  map[string]string
}

func NewBuildGraph(plan *plan.BuildPlan, baseState *llb.State, cacheStore *BuildKitCacheStore) (*BuildGraph, error) {
	graph := &BuildGraph{
		Nodes:      make(map[string]*Node),
		BaseState:  baseState,
		CacheStore: cacheStore,
		Plan:       plan,
	}

	// Create a node for each step
	for i := range plan.Steps {
		step := &plan.Steps[i]
		graph.Nodes[step.Name] = &Node{
			Step:           step,
			Parents:        make([]*Node, 0),
			Children:       make([]*Node, 0),
			Processed:      false,
			OutputEnvVars:  make(map[string]string),
			OutputPathList: make([]string, 0),
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

	return graph, nil
}

func (g *BuildGraph) GenerateLLB() (*BuildGraphOutput, error) {
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

	outputPathList := make([]string, 0)
	outputEnvVars := make(map[string]string)

	for _, node := range g.Nodes {
		if len(node.Children) == 0 && node.State != nil {
			leafStates = append(leafStates, *node.State)
			leafStepNames = append(leafStepNames, node.Step.Name)

			// Add output path and env vars
			outputPathList = append(outputPathList, node.OutputPathList...)
			for k, v := range node.OutputEnvVars {
				outputEnvVars[k] = v
			}
		}

	}

	// If no leaf states, return base state
	if len(leafStates) == 0 {
		return &BuildGraphOutput{
			State:    g.BaseState,
			PathList: outputPathList,
			EnvVars:  outputEnvVars,
		}, nil
	}

	// If only one leaf state, return it
	if len(leafStates) == 1 {
		return &BuildGraphOutput{
			State:    &leafStates[0],
			PathList: outputPathList,
			EnvVars:  outputEnvVars,
		}, nil
	}

	// Merge all leaf states
	mergeName := fmt.Sprintf("merging steps: %s", strings.Join(leafStepNames, ", "))
	result := llb.Merge(leafStates, llb.WithCustomName(mergeName))

	return &BuildGraphOutput{
		State:    &result,
		PathList: outputPathList,
		EnvVars:  outputEnvVars,
	}, nil
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
	currentEnvVars := make(map[string]string)
	currentPathList := make([]string, 0)

	if len(node.Parents) == 0 {
		currentState = g.BaseState
	} else if len(node.Parents) == 1 {
		// If only one parent, use its state directly
		currentState = node.Parents[0].State
		currentEnvVars = node.Parents[0].OutputEnvVars
		currentPathList = node.Parents[0].OutputPathList
	} else {
		// If multiple parents, merge their states
		parentStates := make([]llb.State, len(node.Parents))
		mergeStepNames := make([]string, len(node.Parents))

		for i, parent := range node.Parents {
			if parent.State == nil {
				return fmt.Errorf("Parent %s of %s has nil state",
					parent.Step.Name, node.Step.Name)
			}

			// Build up the current path and env vars
			currentPathList = append(currentPathList, parent.OutputPathList...)
			for k, v := range parent.OutputEnvVars {
				currentEnvVars[k] = v
			}

			parentStates[i] = *parent.State
			mergeStepNames[i] = parent.Step.Name
		}

		mergeName := fmt.Sprintf("merging steps: %s", strings.Join(mergeStepNames, ", "))
		merged := llb.Merge(parentStates, llb.WithCustomName(mergeName))
		currentState = &merged
	}

	node.InputPathList = currentPathList
	node.InputEnvVars = currentEnvVars

	// Convert this node's step to LLB
	stepState, err := g.convertStepToLLB(node, currentState)
	if err != nil {
		return err
	}

	node.State = stepState
	node.Processed = true

	return nil
}

func (g *BuildGraph) convertStepToLLB(node *Node, baseState *llb.State) (*llb.State, error) {
	step := node.Step
	state := *baseState
	state = state.Dir("/app")

	// Add commands for input variables and path
	for k, v := range node.InputEnvVars {
		newState, err := g.convertCommandToLLB(node, plan.VariableCommand{Name: k, Value: v}, state, step)
		if err != nil {
			return nil, err
		}
		state = newState
	}

	for _, path := range node.InputPathList {
		newState, err := g.convertCommandToLLB(node, plan.PathCommand{Path: path}, state, step)
		if err != nil {
			return nil, err
		}
		state = newState
	}

	// Process the step commands
	for _, cmd := range step.Commands {
		var err error
		state, err = g.convertCommandToLLB(node, cmd, state, step)
		if err != nil {
			return nil, err
		}
	}

	if len(step.Outputs) > 0 {
		result := llb.Scratch()

		for _, output := range step.Outputs {
			result = result.File(llb.Copy(state, output, output, &llb.CopyInfo{
				CreateDestPath:      true,
				AllowWildcard:       true,
				AllowEmptyWildcard:  true,
				CopyDirContentsOnly: false,
				FollowSymlinks:      true,
			}))
		}

		merged := llb.Merge([]llb.State{*baseState, result})
		state = merged
	}

	return &state, nil
}

func (g *BuildGraph) convertCommandToLLB(node *Node, cmd plan.Command, state llb.State, step *plan.Step) (llb.State, error) {
	switch cmd := cmd.(type) {
	case plan.ExecCommand:
		opts := []llb.RunOption{llb.Shlex(cmd.Cmd)}
		if cmd.CustomName != "" {
			opts = append(opts, llb.WithCustomName(cmd.CustomName))
		}

		if cmd.CacheKey != "" {
			if planCache, ok := g.Plan.Caches[cmd.CacheKey]; ok {
				cache := g.CacheStore.GetCache(cmd.CacheKey, &planCache)
				opts = append(opts,
					llb.AddMount(planCache.Directory, *cache.cacheState, llb.AsPersistentCacheDir(cache.cacheKey, llb.CacheMountShared)),
				)
			} else {
				return state, fmt.Errorf("cache with key %q not found", cmd.CacheKey)
			}
		}

		s := state.Run(opts...).Root()
		return s, nil

	case plan.PathCommand:
		node.appendPath(cmd.Path)
		pathList := node.getPathList()
		pathString := strings.Join(pathList, ":")

		s := state.AddEnvf("PATH", "%s:%s", pathString, system.DefaultPathEnvUnix)

		return s, nil

	case plan.VariableCommand:
		s := state.AddEnv(cmd.Name, cmd.Value)
		node.OutputEnvVars[cmd.Name] = cmd.Value

		return s, nil

	case plan.CopyCommand:
		src := llb.Local("context")
		s := state.File(llb.Copy(src, cmd.Src, cmd.Dst, &llb.CopyInfo{
			CreateDestPath:      true,
			FollowSymlinks:      true,
			CopyDirContentsOnly: false,
		}))
		return s, nil

	case plan.FileCommand:
		asset, ok := step.Assets[cmd.Name]
		if !ok {
			return state, fmt.Errorf("asset %q not found", cmd.Name)
		}

		// Create parent directories for the file
		parentDir := filepath.Dir(cmd.Path)
		if parentDir != "/" {
			s := state.File(llb.Mkdir(parentDir, 0755, llb.WithParents(true)))
			state = s
		}

		fileAction := llb.Mkfile(cmd.Path, 0644, []byte(asset))
		s := state.File(fileAction)
		if cmd.CustomName != "" {
			s = state.File(fileAction, llb.WithCustomName(cmd.CustomName))
		}

		return s, nil
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
