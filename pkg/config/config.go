package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/whiskeyjimbo/CheckMate/pkg/rules"
	"gopkg.in/yaml.v2"
)

type CheckConfig struct {
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Interval string `yaml:"interval"`
}

type HostConfig struct {
	Host   string        `yaml:"host"`
	Checks []CheckConfig `yaml:"checks"`
}

type Config struct {
	Hosts    []HostConfig `yaml:"hosts"`
	RawRules []rules.Rule `yaml:"rules"`
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

	for _, host := range config.Hosts {
		for i := range host.Checks {
			normalizeConfig(&host.Checks[i])
		}
	}

	return &config, nil
}

func normalizeConfig(c *CheckConfig) {
	c.Protocol = strings.ToUpper(c.Protocol)
	if _, err := strconv.Atoi(c.Interval); err == nil {
		c.Interval = c.Interval + "s"
	}
}
