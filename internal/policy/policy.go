package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

type Policy struct {
	Workload    string   `yaml:"workload"`
	Destination string   `yaml:"destination"`
	Methods     []string `yaml:"methods,omitempty"`
	Effect      Effect   `yaml:"effect"`
}

type Config struct {
	Policies []Policy `yaml:"policies"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing policy file %q: %w", path, err)
	}
	return &cfg, nil
}
