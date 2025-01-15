package plan

import (
	"bytes"

	"github.com/BurntSushi/toml"
)

type PlanPackages struct {
	Apt  []string          `json:"apt,omitempty"`
	Mise map[string]string `json:"mise,omitempty"`
}

func NewPlanPackages() *PlanPackages {
	return &PlanPackages{
		Apt:  []string{},
		Mise: map[string]string{},
	}
}

func (p *PlanPackages) AddAptPackage(pkg string) {
	p.Apt = append(p.Apt, pkg)
}

func (p *PlanPackages) AddMisePackage(pkg string, version string) {
	p.Mise[pkg] = version
}

// MisePackage represents a single mise package configuration
type MisePackage struct {
	Version string `toml:"version"`
}

// MiseConfig represents the overall mise configuration
type MiseConfig struct {
	Tools map[string]MisePackage `toml:"tools"`
}

func (p *PlanPackages) GenerateMiseToml() (string, error) {
	config := MiseConfig{
		Tools: make(map[string]MisePackage),
	}

	for name, version := range p.Mise {
		config.Tools[name] = MisePackage{
			Version: version,
		}
	}

	buf := bytes.NewBuffer(nil)
	if err := toml.NewEncoder(buf).Encode(config); err != nil {
		return "", err
	}

	return buf.String(), nil
}
