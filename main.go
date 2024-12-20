package main

import (
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

func main() {
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	protocol := os.Getenv("PROTOCOL")
	if protocol == "" {
		protocol = "tcp"
	}

	if host == "" || port == "" || protocol == "" {
		fmt.Println("Error: HOST, PORT, and PROTOCOL environment variables must be set.")
		os.Exit(1)
	}

	address := fmt.Sprintf("%s:%s", host, port)

	switch strings.ToLower(protocol) {
	case "tcp":
		conn, err := net.Dial(protocol, address)
		if err != nil {
			fmt.Printf("Error: Port %s is not available on %s using %s protocol\n", port, host, protocol)
			os.Exit(1)
		}
		defer conn.Close()

		fmt.Printf("Success: Port %s is available on %s using %s protocol\n", port, host, protocol)
	case "http":
		resp, err := http.Get(fmt.Sprintf("http://%s", address))
		if err != nil {
			fmt.Printf("Error: HTTP request to %s failed: %v\n", address, err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("Success: HTTP request to %s succeeded with status code %d\n", address, resp.StatusCode)
		} else {
			fmt.Printf("Error: HTTP request to %s returned status code %d\n", address, resp.StatusCode)
			os.Exit(1)
		}
	case "smtp":
		c, err := smtp.Dial(address)
		if err != nil {
			fmt.Printf("Error: SMTP connection to %s failed: %v\n", address, err)
			os.Exit(1)
		}
		defer c.Close()
		fmt.Printf("Success: SMTP connection to %s succeeded\n", address)
	case "dns":
		_, err := net.LookupHost(host)
		if err != nil {
			fmt.Printf("Error: DNS resolution for %s failed: %v\n", host, err)
			os.Exit(1)
		}
		fmt.Printf("Success: DNS resolution for %s succeeded\n", host)
	default:
		fmt.Printf("Error: Unsupported protocol %s\n", protocol)
		os.Exit(1)
	}
}
