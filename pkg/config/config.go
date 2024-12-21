package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type HostConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Interval string `yaml:"interval"`
}

type Config struct {
	Hosts []HostConfig `yaml:"hosts"`
}

func LoadConfig(configFile string) (*Config, error) {
	fileContent, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(fileContent, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	for i := range config.Hosts {
		normalizeConfig(&config.Hosts[i])
	}

	return &config, nil
}

func normalizeConfig(c *HostConfig) {
	c.Protocol = strings.ToUpper(c.Protocol)
	if _, err := strconv.Atoi(c.Interval); err == nil {
		c.Interval = c.Interval + "s"
	}
}
