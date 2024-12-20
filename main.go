package main

import (
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // Flushes buffer, if any
	sugar := logger.Sugar()

	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	protocol := strings.ToUpper(os.Getenv("PROTOCOL"))
	if protocol == "" {
		protocol = "TCP"
	}

	if host == "" || port == "" || protocol == "" {
		sugar.Fatalf("Error: HOST, PORT, and PROTOCOL environment variables must be set.")
	}

	address := fmt.Sprintf("%s:%s", host, port)

	switch strings.ToUpper(protocol) {
	case "TCP":
		conn, err := net.Dial(protocol, address)
		if err != nil {
			sugar.Fatalf("Error: %s connection to %s failed: %v", protocol, address, err)
		}
		defer conn.Close()
		sugar.Infof("Success: %s connection to %s succeeded", protocol, address)
	case "HTTP":
		resp, err := http.Get(fmt.Sprintf("http://%s", address))
		if err != nil {
			sugar.Fatalf("Error: %s request to %s failed: %v", protocol, address, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			sugar.Infof("Success: %s request to %s succeeded with status code %d", protocol, address, resp.StatusCode)
		} else {
			sugar.Fatalf("Error: %s request to %s returned status code %d", protocol, address, resp.StatusCode)
		}
	case "SMTP":
		c, err := smtp.Dial(address)
		if err != nil {
			sugar.Fatalf("Error: %s connection to %s failed: %v", protocol, address, err)
		}
		defer c.Close()
		sugar.Infof("Success: %s connection to %s succeeded", protocol, address)
	case "DNS":
		_, err := net.LookupHost(host)
		if err != nil {
			sugar.Fatalf("Error: %s resolution for %s failed: %v", protocol, host, err)
		}
		sugar.Infof("Success: %s resolution for %s succeeded", protocol, host)
	default:
		sugar.Fatalf("Error: Unsupported protocol %s", protocol)
	}
}
