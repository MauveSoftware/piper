package main

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// Config represents the config file
type Config struct {
	Proto int
	Pipes []PipeConfig
}

// PipeConfig represent the config for a single pipe
type PipeConfig struct {
	Name   string
	Prefix string
	Source int
	Target int
}

func loadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not open config file")
	}

	cfg := &Config{}
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse config file")
	}

	return cfg, nil
}
