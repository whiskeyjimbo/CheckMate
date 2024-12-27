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
	Port     string   `yaml:"port"`
	Protocol string   `yaml:"protocol"`
	Interval string   `yaml:"interval"`
	Tags     []string `yaml:"tags"`
}

type HostConfig struct {
	Host   string        `yaml:"host"`
	Tags   []string      `yaml:"tags"`
	Checks []CheckConfig `yaml:"checks"`
}

type NotificationConfig struct {
	Type string `yaml:"type"`
}

type SiteConfig struct {
	Name  string       `yaml:"name"`
	Tags  []string     `yaml:"tags"`
	Hosts []HostConfig `yaml:"hosts"`
}

type Config struct {
	Sites         []SiteConfig         `yaml:"sites"`
	Rules         []rules.Rule         `yaml:"rules"`
	Notifications []NotificationConfig `yaml:"notifications"`
}

func loadConfig(configFile string) (*Config, error) {
	fileContent, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(fileContent, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	for _, site := range config.Sites {
		for _, host := range site.Hosts {
			for i := range host.Checks {
				normalizeConfig(&host.Checks[i])
			}
		}
	}

	return &config, nil
}

func LoadConfiguration(args []string) (*Config, error) {
	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	config, err := loadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return config, nil
}

func normalizeConfig(c *CheckConfig) {
	c.Protocol = strings.ToUpper(c.Protocol)
	if _, err := strconv.Atoi(c.Interval); err == nil {
		c.Interval = c.Interval + "s"
	}
}
