package main

import (
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

// Config represents the config file
type Config struct {
	Proto netlink.RouteProtocol
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
	b, err := os.ReadFile(path) // #nosec G304 -- path is provided by the operator via the -config-file flag, not attacker controlled
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
