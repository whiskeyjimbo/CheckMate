# CheckMate

![License](https://img.shields.io/badge/license-GPLv3-blue.svg)
![Go Version](https://img.shields.io/badge/language-go-blue.svg)

CheckMate is a service monitoring tool written in Go that provides real-time health checks and metrics for infrastructure. It supports multiple protocols, customizable rules, and Prometheus integration.

DISCLAIMER: This is a personal project and is not meant to be used in a production environment as it is not feature complete nor secure nor tested and under heavy development. 

## Features

### Core Features
- Multi-protocol support (TCP, HTTP, SMTP, DNS(*))
- Configurable check intervals per service
- Prometheus metrics integration
- Rule-based monitoring with custom conditions (WIP)
- Structured logging with Zap
- Concurrent monitoring of multiple hosts and services

### Metrics & Monitoring
- Service availability status
- Response time measurements
- Custom rule evaluation (WIP)
- Prometheus-compatible metrics endpoint
- Downtime tracking (WIP)

### Technical Features
- YAML-based configuration
- Modular architecture for easy extension using interfaces

## Installation

### Prerequisites
- Go
- Git

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/whiskeyjimbo/CheckMate.git
cd CheckMate
```

2. Build the application:
```bash
go build
```

## Configuration

Create a `config.yaml` file with your service definitions and monitoring rules:

```yaml
hosts:
  - host: example.com
    checks:
      - port: "80"
        protocol: HTTP
        interval: 30s
      - port: "443"
        protocol: TCP
        interval: 1m
rules:
  - name: high_latency
    condition: "responseTime > 5s"
  - name: extended_downtime
    condition: "downtime > 5m"
```

### Configuration Options

#### Host Configuration
- `host`: Target hostname or IP address
- `checks`: List of service checks
  - `port`: Service port
  - `protocol`: Check protocol (HTTP, TCP, SMTP, DNS)
  - `interval`: Check frequency (e.g., "30s", "1m")

#### Rule Configuration
- `name`: Rule identifier
- `condition`: Expression using variables:
  - `responseTime`: Service response time
  - `downtime`: Accumulated downtime

## Metrics

CheckMate exposes Prometheus metrics at `:9100/metrics` including:
- `check_success`: Service availability (1 = up, 0 = down)
- `check_latency_milliseconds`: Response time gauge
- `check_latency_milliseconds_histogram`: Response time distribution

## Extending CheckMate

### Adding New Protocols

1. Create a new checker in `pkg/checkers/`:
```go
type NewProtocolChecker struct{}

func (c NewProtocolChecker) Check(address string) (success bool, responseTime int64, err error) {
    // Implement protocol check
}
```

2. Register in `pkg/checkers/checker.go`:
```go
func NewChecker(protocol string) (Checker, error) {
    switch protocol {
    case "NEWPROTOCOL":
        return &NewProtocolChecker{}, nil
    // ...
    }
}
```

### Adding Database Support

Implement the Database interface in `pkg/database/`:
```go
type Database interface {
    InsertCheck(host, port, protocol, status string, elapsed int64) error
    Close() error
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Roadmap

- [ ] Additional protocol support
- [ ] Notification system integration
- [ ] Configurable notification thresholds
- [ ] database support
- [x] Multiple host monitoring
- [x] Multi-protocol per host
- [ ] Service tagging system
- [ ] Docker container 
- [ ] Web UI for monitoring (MAYBE) 