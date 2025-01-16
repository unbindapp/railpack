package plan

import (
	"encoding/json"
	"fmt"
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

	if (p.Packages == nil) != (other.Packages == nil) {
		return false
	}

	if p.Packages != nil {
		return reflect.DeepEqual(*p.Packages, *other.Packages)
	}
	return true
}

func TestSerialization(t *testing.T) {
	plan := NewBuildPlan()
	plan.Variables["MISE_DATA_DIR"] = "/mise"
	plan.Packages.AddAptPackage("curl")
	plan.Packages.AddMisePackage("bun", "1.0.0")
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

	fmt.Printf("%+v\n", plan)
	fmt.Printf("%+v\n", plan2)

	serialized1, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	serialized2, err := json.MarshalIndent(plan2, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(serialized1))
	fmt.Println(string(serialized2))

	fmt.Println(plan.Equal(&plan2))

	if !plan.Equal(&plan2) {
		t.Fatal("plans are not equal")
	}
}
