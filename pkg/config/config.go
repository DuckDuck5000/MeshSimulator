package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type NodeConfig struct {
	ID        string   `yaml:"id"`
	Neighbors []string `yaml:"neighbors"`
}

type Topology struct {
	Nodes []NodeConfig `yaml:"nodes"`
}

// LoadConfig reads the topology YAML
func LoadConfig(path string) (*Topology, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Topology
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
