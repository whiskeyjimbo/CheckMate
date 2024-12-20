package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	defaultHost     = "localhost"
	defaultPort     = "2525"
	defaultProtocol = "SMTP"
	defaultInterval = "10s"
)

type Config struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Interval string `yaml:"interval"`
}

func LoadConfig(configFile string) (*Config, error) {
	defaultConfig := Config{
		Host:     defaultHost,
		Port:     defaultPort,
		Protocol: defaultProtocol,
		Interval: defaultInterval,
	}

	fileContent, err := os.ReadFile(configFile)
	if err != nil {
		return &defaultConfig, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(fileContent, &defaultConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	normalizeConfig(&defaultConfig)

	return &defaultConfig, nil
}

func GetEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func normalizeConfig(c *Config) {
	fmt.Println("got here")
	c.Protocol = strings.ToUpper(c.Protocol)
	if _, err := strconv.Atoi(c.Interval); err == nil {
		fmt.Println("interval is a number")
		c.Interval = c.Interval + "s"
	}
}
