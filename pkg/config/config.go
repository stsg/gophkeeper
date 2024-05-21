// Package config config contains all configuration parameters for the application
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parameters contains all parameters for the application
type Parameters struct {
	DataSet  []Data `yaml:"data"`
	filename string `yaml:"filename"`
}

// Data represents a volumes to check
type Data struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
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
func (p *Parameters) MarshalDataSet() []string {
	res := make([]string, 0, len(p.DataSet))
	for _, v := range p.DataSet {
		res = append(res, fmt.Sprintf("%s:%s", v.Name, v.Value))
	}
	return res
}
