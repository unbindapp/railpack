package plan

import (
	"encoding/json"
	"reflect"
	"testing"
)

func (p *BuildPlan) Equal(other *BuildPlan) bool {
	if !reflect.DeepEqual(p.Variables, other.Variables) {
		return false
	}

	if !reflect.DeepEqual(p.Steps, other.Steps) {
		return false
	}

	return true
}

func TestSerialization(t *testing.T) {
	plan := NewBuildPlan()
	plan.Variables["MISE_DATA_DIR"] = "/mise"

	plan.AddStep(Step{
		Name: "install",
		Commands: []Command{
			NewExecCommand("apt-get update"),
			NewExecCommand("apt-get install -y curl"),
		},
	})

	plan.AddStep(Step{
		Name:      "build",
		DependsOn: []string{"install"},
		Commands: []Command{
			NewCopyCommand(".", "."),
			NewExecCommand("bun i --no-save"),
			NewPathCommand("/mise/shims"),
			NewVariableCommand("MISE_DATA_DIR", "/mise"),
		},
	})

	serialized, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	plan2 := BuildPlan{}
	err = json.Unmarshal(serialized, &plan2)
	if err != nil {
		t.Fatal(err)
	}

	if !plan.Equal(&plan2) {
		t.Fatal("plans are not equal")
	}
}
