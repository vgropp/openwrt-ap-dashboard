package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type StationConfig struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Ifaces   []string `yaml:"ifaces"`
}

type Config struct {
	Stations     []StationConfig `yaml:"stations"`
	PollInterval int             `yaml:"poll_interval"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.PollInterval == 0 {
		c.PollInterval = 15
	}
	return &c, nil
}
