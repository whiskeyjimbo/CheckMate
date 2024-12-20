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
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	host := getEnv("HOST", "localhost")
	port := getEnv("PORT", "80")
	protocol := getEnv("PROTOCOL", "http")
	intervalStr := getEnv("INTERVAL", "10s")

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		sugar.Fatalf("Error: Invalid INTERVAL value: %v", err)
	}

	address := fmt.Sprintf("%s:%s", host, port)

	for {
		switch strings.ToUpper(protocol) {
		case "TCP":
			conn, err := net.Dial(protocol, address)
			if err != nil {
				sugar.Errorf("Error: TCP connection to %s failed: %v", address, err)
			} else {
				defer conn.Close()
				sugar.Infof("Success: TCP connection to %s succeeded", address)
			}
		case "HTTP":
			resp, err := http.Get(fmt.Sprintf("http://%s", address))
			if err != nil {
				sugar.Errorf("Error: HTTP request to %s failed: %v", address, err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					sugar.Infof("Success: HTTP request to %s succeeded with status code %d", address, resp.StatusCode)
				} else {
					sugar.Errorf("Error: HTTP request to %s returned status code %d", address, resp.StatusCode)
				}
			}
		case "SMTP":
			c, err := smtp.Dial(address)
			if err != nil {
				sugar.Errorf("Error: SMTP connection to %s failed: %v", address, err)
			} else {
				defer c.Close()
				sugar.Infof("Success: SMTP connection to %s succeeded", address)
			}
		case "DNS":
			_, err := net.LookupHost(host)
			if err != nil {
				sugar.Errorf("Error: DNS resolution for %s failed: %v", host, err)
			} else {
				sugar.Infof("Success: DNS resolution for %s succeeded", host)
			}
		default:
			sugar.Fatalf("Error: Unsupported protocol %s", protocol)
		}

		time.Sleep(interval)
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
