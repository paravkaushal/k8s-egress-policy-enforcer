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

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating policy file %q: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	for i, p := range c.Policies {
		if p.Workload == "" {
			return fmt.Errorf("policy[%d]: Workload is required", i)
		}
		if p.Destination == "" {
			return fmt.Errorf("policy[%d]: Destination is required", i)
		}
		if p.Effect != EffectAllow && p.Effect != EffectDeny {
			return fmt.Errorf("policy[%d]: effect must be %q or %q, got %q", i, EffectAllow, EffectDeny, p.Effect)
		}
	}
	return nil
}
