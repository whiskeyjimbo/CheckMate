package main

import (
	"fmt"
	"net"
	"os"
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

	// TODO: net dial only supports tcp/udp will need to break this out to allow for other protocols
	conn, err := net.Dial(protocol, address)
	if err != nil {
		fmt.Printf("Error: Port %s is not available on %s using %s protocol\n", port, host, protocol)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Success: Port %s is available on %s using %s protocol\n", port, host, protocol)
}
