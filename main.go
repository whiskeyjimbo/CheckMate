package main

import (
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Host     string        `yaml:"host"`
	Port     string        `yaml:"port"`
	Protocol string        `yaml:"protocol"`
	Interval time.Duration `yaml:"interval"`
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	configFile := getEnv("CONFIG_FILE", "config.yaml") // Default config file name
	config, err := loadConfig(configFile)
	if err != nil {
		sugar.Fatalf("Error loading config: %v", err)
	}

	address := fmt.Sprintf("%s:%s", config.Host, config.Port)

	for {
		switch protocol := strings.ToUpper(config.Protocol); protocol {
		case "TCP":
			conn, err := net.Dial(protocol, address)
			if err != nil {
				sugar.With("status", "failure").Errorf("Error: TCP connection to %s failed: %v", address, err)
			} else {
				defer conn.Close()
				sugar.With("status", "success").Infof("Success: TCP connection to %s succeeded", address)
			}
		case "HTTP":
			resp, err := http.Get(fmt.Sprintf("http://%s", address))
			if err != nil {
				sugar.With("status", "failure").Errorf("Error: HTTP request to %s failed: %v", address, err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					sugar.With("status", "success").Infof("Success: HTTP request to %s succeeded with status code %d", address, resp.StatusCode)
				} else {
					sugar.With("status", "failure").Errorf("Error: HTTP request to %s returned status code %d", address, resp.StatusCode)
				}
			}
		case "SMTP":
			c, err := smtp.Dial(address)
			if err != nil {
				sugar.With("status", "failure").Errorf("Error: SMTP connection to %s failed: %v", address, err)
			} else {
				defer c.Close()
				sugar.With("status", "success").Infof("Success: SMTP connection to %s succeeded", address)
			}
		case "DNS":
			_, err := net.LookupHost(config.Host)
			if err != nil {
				sugar.With("status", "failure").Errorf("Error: DNS resolution for %s failed: %v", config.Host, err)
			} else {
				sugar.With("status", "success").Infof("Success: DNS resolution for %s succeeded", config.Host)
			}
		default:
			sugar.With("status", "failure").Fatalf("Error: Unsupported protocol %s", protocol)
		}

		time.Sleep(config.Interval)
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func loadConfig(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %w", err)
	}

	return &config, nil
}
