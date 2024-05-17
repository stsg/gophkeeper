// Package config config contains all configuration parameters for the application
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parameters contains all parameters for the application
type Parameters struct {
	Volumes  []Volume `yaml:"volumes"`
	filename string   `yaml:"filename"`
}

// Volume represents a volumes to check
type Volume struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// New creates new Parameters from the given filename
func New(filename string) (*Parameters, error) {
	p := &Parameters{
		filename: filename,
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("can't read config file %s: %w", filename, err)
	}
	if err = yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}
	return p, nil
}

// String returns a string representation of the Parameters struct, including the
// filename and the struct fields.
//
// No parameters.
// Returns a string.
func (p *Parameters) String() string {
	return fmt.Sprintf("config file: %q, %+v", p.filename, *p)
}

// MarshalVolumes returns the volumes as a list of strings with the format "name:path"
func (p *Parameters) MarshalVolumes() []string {
	res := make([]string, 0, len(p.Volumes))
	for _, v := range p.Volumes {
		res = append(res, fmt.Sprintf("%s:%s", v.Name, v.Path))
	}
	return res
}
