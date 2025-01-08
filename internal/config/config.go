package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/drone/envsubst"
	"github.com/whiskeyjimbo/CheckMate/internal/rules"
	"gopkg.in/yaml.v2"
)

type RuleMode string

const (
	RuleModeAll RuleMode = "all" // Fire rules only if all hosts are down (default)
	RuleModeAny RuleMode = "any" // Fire rules if any host is down
)

type Config struct {
	MonitorSite   string               `yaml:"monitorSite"`
	Sites         []SiteConfig         `yaml:"sites"`
	Rules         []rules.Rule         `yaml:"rules"`
	Notifications []NotificationConfig `yaml:"notifications"`
}

type SiteConfig struct {
	Name   string        `yaml:"name"`
	Tags   []string      `yaml:"tags"`
	Groups []GroupConfig `yaml:"groups"`
}

type GroupConfig struct {
	Name     string        `yaml:"name"`
	RuleMode RuleMode      `yaml:"ruleMode,omitempty"`
	Tags     []string      `yaml:"tags"`
	Hosts    []HostConfig  `yaml:"hosts"`
	Checks   []CheckConfig `yaml:"checks"`
}

type HostConfig struct {
	Host     string        `yaml:"host"`
	RuleMode RuleMode      `yaml:"ruleMode,omitempty"`
	Tags     []string      `yaml:"tags"`
	Checks   []CheckConfig `yaml:"checks"`
}

type CheckConfig struct {
	Port       string   `yaml:"port"`
	Protocol   string   `yaml:"protocol"`
	Interval   string   `yaml:"interval"`
	RuleMode   RuleMode `yaml:"ruleMode,omitempty"`
	Tags       []string `yaml:"tags"`
	VerifyCert bool     `yaml:"verifyCert,omitempty"`
}

type NotificationConfig struct {
	Type string `yaml:"type"`
}

func LoadConfiguration(args []string) (*Config, error) {
	configFile := "config.yaml"
	if len(args) > 1 {
		configFile = args[1]
	}

	config, err := loadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	normalizeConfig(config)
	return config, nil
}

func loadConfig(filename string) (*Config, error) {
	cleanPath := filepath.Clean(filename)
	if filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("absolute paths are not allowed: %s", filename)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	// Use envsubst to handle environment variable substitution
	expandedData, err := envsubst.EvalEnv(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to substitute environment variables: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func validateConfig(c *Config) error {
	if c.MonitorSite == "" {
		return fmt.Errorf("monitorSite must be specified")
	}

	for _, site := range c.Sites {
		if site.Name == "" {
			return fmt.Errorf("site name cannot be empty")
		}
		for _, group := range site.Groups {
			if err := group.Validate(); err != nil {
				return fmt.Errorf("invalid group '%s': %w", group.Name, err)
			}
		}
	}
	return nil
}

func normalizeConfig(c *Config) {
	for i := range c.Sites {
		for j := range c.Sites[i].Groups {
			for k := range c.Sites[i].Groups[j].Checks {
				normalizeCheck(&c.Sites[i].Groups[j].Checks[k])
			}
			for k := range c.Sites[i].Groups[j].Hosts {
				for l := range c.Sites[i].Groups[j].Hosts[k].Checks {
					normalizeCheck(&c.Sites[i].Groups[j].Hosts[k].Checks[l])
				}
			}
		}
	}
}

func normalizeCheck(c *CheckConfig) {
	c.Protocol = strings.ToUpper(c.Protocol)
	if _, err := strconv.Atoi(c.Interval); err == nil {
		c.Interval = c.Interval + "s"
	}
}

func (g *GroupConfig) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	if g.RuleMode == "" {
		g.RuleMode = RuleModeAll
	}
	return nil
}

type RuleModeResolver struct {
	Group GroupConfig
}

func NewRuleModeResolver(group GroupConfig) *RuleModeResolver {
	return &RuleModeResolver{
		Group: group,
	}
}

func (r *RuleModeResolver) GetEffectiveRuleMode(check CheckConfig) RuleMode {
	if check.RuleMode != "" {
		return check.RuleMode
	}
	if r.Group.RuleMode != "" {
		return r.Group.RuleMode
	}
	return RuleModeAll
}

func (r *RuleModeResolver) ShouldTrigger(anyDown bool, allDown bool, check CheckConfig) bool {
	switch r.GetEffectiveRuleMode(check) {
	case RuleModeAny:
		return anyDown
	default:
		return allDown
	}
}
