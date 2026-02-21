// Package pipeline provides a YAML-based workflow execution engine.
package pipeline

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Pipeline represents a complete workflow definition.
type Pipeline struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
	Steps   []Step `yaml:"steps" json:"steps"`
}

// Step represents a single action in a pipeline.
type Step struct {
	ID        string            `yaml:"id" json:"id"`
	Action    string            `yaml:"action" json:"action"`
	Input     string            `yaml:"input,omitempty" json:"input,omitempty"`
	Template  string            `yaml:"template,omitempty" json:"template,omitempty"`
	Data      string            `yaml:"data,omitempty" json:"data,omitempty"`
	Options   map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
	To        string            `yaml:"to,omitempty" json:"to,omitempty"`
	Subject   string            `yaml:"subject,omitempty" json:"subject,omitempty"`
	Attach    string            `yaml:"attach,omitempty" json:"attach,omitempty"`
	OnFailure string            `yaml:"on_failure,omitempty" json:"onFailure,omitempty"`
}

// StepResult holds the output of a completed pipeline step.
type StepResult struct {
	StepID string `json:"stepId"`
	Output string `json:"output"`
	Error  error  `json:"error,omitempty"`
}

// LoadPipeline reads and parses a pipeline YAML file.
func LoadPipeline(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("pipeline file not found: %s — check that the path is correct", path)
		}
		return nil, fmt.Errorf("could not read pipeline file %s: %w", path, err)
	}

	return ParsePipeline(data)
}

// ParsePipeline parses a pipeline from YAML bytes.
func ParsePipeline(data []byte) (*Pipeline, error) {
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid pipeline YAML: %w", err)
	}

	if err := validatePipeline(&p); err != nil {
		return nil, err
	}

	return &p, nil
}

func validatePipeline(p *Pipeline) error {
	if p.Name == "" {
		return fmt.Errorf("pipeline is missing a 'name' field")
	}

	if len(p.Steps) == 0 {
		return fmt.Errorf("pipeline %q has no steps defined", p.Name)
	}

	seen := make(map[string]bool)
	for i, step := range p.Steps {
		if step.ID == "" {
			return fmt.Errorf("step %d is missing an 'id' field", i+1)
		}
		if seen[step.ID] {
			return fmt.Errorf("duplicate step ID %q — each step must have a unique ID", step.ID)
		}
		seen[step.ID] = true

		if step.Action == "" {
			return fmt.Errorf("step %q is missing an 'action' field", step.ID)
		}
	}

	return nil
}
