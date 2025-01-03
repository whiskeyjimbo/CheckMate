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
	RuleMode RuleMode `yaml:"ruleMode,omitempty"`
}

type HostConfig struct {
	Host     string        `yaml:"host"`
	Tags     []string      `yaml:"tags"`
	Checks   []CheckConfig `yaml:"checks"`
	RuleMode RuleMode      `yaml:"ruleMode,omitempty"`
}

type NotificationConfig struct {
	Type string `yaml:"type"`
}

type RuleMode string

const (
	RuleModeAll RuleMode = "all" // Fire rules only if all hosts are down (default)
	RuleModeAny RuleMode = "any" // Fire rules if any host is down
)

type GroupConfig struct {
	Name     string        `yaml:"name"`
	Tags     []string      `yaml:"tags"`
	Hosts    []HostConfig  `yaml:"hosts"`
	Checks   []CheckConfig `yaml:"checks"`
	RuleMode RuleMode      `yaml:"ruleMode,omitempty"`
}

func (g *GroupConfig) Validate() error {
	if g.RuleMode == "" {
		g.RuleMode = RuleModeAll
	}

	if g.RuleMode != RuleModeAll && g.RuleMode != RuleModeAny {
		return fmt.Errorf("invalid rule mode: %s", g.RuleMode)
	}
	return nil
}

type SiteConfig struct {
	Name   string        `yaml:"name"`
	Tags   []string      `yaml:"tags"`
	Groups []GroupConfig `yaml:"groups"`
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
		for _, host := range site.Groups {
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
