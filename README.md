# CheckMate

A simple and extensible Go application for monitoring service availability and health.

## Features

* Checks TCP, HTTP, SMTP, and DNS protocols.
* Configurable check interval and target host/port/protocol.
* Customizable notifications (STUB).
* Stores check results in a database (STUB).
* Exposes metrics for Prometheus.
* modular design for easy extensibility ?.

## TODO
- [ ] Add more protocols
- [ ] Add notifications and support for notification rules
- [ ] Add database support
- [X] multiple host support
- [X] hosts with multiple protocols/ports
- [ ] tags for hosts and ports
- [ ] docker image

## Getting Started

### Prerequisites

* Go 

### Installation

1. Clone the repository:
   ```bash
   git clone [https://github.com/your-username/port-checker.git](https://github.com/your-username/port-checker.git)
   ```
2. Build the application:
    ``` Bash
    cd port-checker
    go build
    ```
### Configuration
1. Create a config.yaml file with the following structure:
    ```YAML
    hosts:
    - host: localhost
      checks:
      - port: 8080
        protocol: HTTP
        interval: 10s
      - port: 25
        protocol: SMTP
        interval: 30s 
    - host: 127.0.0.1
      checks:
      - port: 22
        protocol: TCP
        interval: 30s
    ```
2. Set the following environment variables:
    ```bash
    export CONFIG_FILE=/path/to/config.yaml # Path to the configuration file 
    ```
3. Running the Application
    ```Bash
    go build -o checkmate
    ./checkmate
    ```
## Extending the Application
### Adding New Checkers
1. Create a new file in the pkg/checkers directory.
2. Implement the Checker interface:
    ```Go
    type Checker interface {
        Check(address string) (success bool, responseTime int64, err error)
    }
    ```
3. Add the new checker to the checkersMap in main.go.

### Adding New Database Support (stubbed)
1. Create a new file in the pkg/database directory.
2. Implement the Database interface:
    ```Go
    type Database interface {
        InsertCheck(host, port, protocol, status string, elapsed int64) error
        Close() error
    }
    ```
3. tbd

### Adding New Notifications (stubbed)
1. Create a new file in the pkg/notifications directory.
2. Implement the notifier interface:
    ```Go
    type Notifier interface {
        SendNotification(message string) error
    }
    ```
3. tbd

## License
This project is licensed under the GPLv3 License - see the LICENSE file for details