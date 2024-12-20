package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/whiskeyjimbo/CheckMate/pkg/checkers"
	"github.com/whiskeyjimbo/CheckMate/pkg/metrics"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	defaultHost     = "localhost"
	defaultPort     = "2525"
	defaultProtocol = "SMTP"
	defaultInterval = "10"
)

type Config struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Interval string `yaml:"interval"`
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	configFile := getEnv("CONFIG_FILE", "config.yaml")
	config, err := loadConfig(configFile)
	if err != nil {
		sugar.Infof("Using default values: %v", err)
	}
	if _, err := strconv.Atoi(config.Interval); err == nil {
		sugar.Infof("Invalid interval %s, assuming Seconds: %s", config.Interval, config.Interval+"s")
		config.Interval = config.Interval + "s"
	}

	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		sugar.Fatalf("Invalid interval %s: %v", config.Interval, err)
	}

	address := fmt.Sprintf("%s:%s", config.Host, config.Port)

	protocol := strings.ToUpper(config.Protocol)
	checker, err := checkers.NewChecker(protocol)
	if err != nil {
		sugar.Fatalf("Unsupported protocol %s", protocol)
	}

	promMetricsEndpoint := metrics.NewPrometheusMetrics(sugar)

	for {
		success, elapsed, err := checker.Check(address)
		if err != nil {
			sugar.With("status", "failure").With("responseTime_us", elapsed).Errorf("Check failed: %v", err)
		} else if success {
			sugar.With("status", "success").With("responseTime_us", elapsed).Info("Check succeeded")
		} else {
			sugar.With("status", "failure").With("responseTime_us", elapsed).Error("Check failed: Unknown")
		}
		promMetricsEndpoint.Update(config.Host, config.Port, config.Protocol, success, elapsed)

		time.Sleep(interval)
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func loadConfig(configFile string) (*Config, error) {
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

	return &defaultConfig, nil
}
