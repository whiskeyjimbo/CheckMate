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
- Rule-based monitoring with custom conditions
- Flexible Notification system with multiple providers

### Metrics & Monitoring
- Service availability status
- Response time measurements
- Rule-based alerting with customizable conditions
- Prometheus-compatible metrics endpoint
- downtime tracking
- Latency histograms and gauges

### Technical Features
- YAML-based configuration
- Modular architecture for easy extension using interfaces

### Health Checks
CheckMate provides Kubernetes-compatible health check endpoints:

- `/health/live` - Liveness probe
  - Returns 200 OK when the service is running
  - not sure how useful this is right now in its current state.

- `/health/ready` - Readiness probe
  - Returns 200 OK when the service is ready to receive traffic
  - Returns 503 Service Unavailable during initialization

All health check endpoints are served on port 9100 alongside the metrics endpoint.

Example Kubernetes probe configuration:
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 9100
  initialDelaySeconds: 5
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /health/ready
    port: 9100
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Installation

### Prerequisites
- Go
- Git
- Make (optional, for using Makefile commands)
- Ko (optional, for building a docker image)

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/whiskeyjimbo/CheckMate.git
cd CheckMate
```

2. Build using Make:
```bash
make build
```

Or build directly with Go:
```bash
go build
```

## Configuration

Create a `config.yaml` file with your service definitions and monitoring rules:

```yaml
hosts:
  - host: example.com
    tags: ["prod", "external"]    # Host tags
    checks:
      - port: "80"
        protocol: HTTP
        interval: 30s
      - port: "443"
        protocol: TCP
        interval: 1m
rules:
  - name: high_latency_prod
    condition: "responseTime > 5s"
    tags: ["prod"]               # Rule tags
  - name: extended_downtime
    condition: "downtime > 5m"
    tags: ["external"]           # Rule tags
notifications:
  - type: "log"    # Uses zap 
  # Future supported types will include:
  # - type: "slack"
  # - type: "email"
```

### Configuration Options

#### Host Configuration
- `host`: Target hostname or IP address
- `tags`: List of tags for grouping and rule targeting (optional)
- `checks`: List of service checks
  - `port`: Service port
  - `protocol`: Check protocol (HTTP, TCP, SMTP, DNS)
  - `interval`: Check frequency (e.g., "30s", "1m")

#### Rule Configuration
- `name`: Rule identifier
- `condition`: Expression using variables:
  - `responseTime`: Service response time in seconds
  - `downtime`: Accumulated downtime in seconds
- `tags`: List of tags to target specific hosts (optional)

#### Notifications Configuration
- `type`: Notification type (currently only `log` is supported)

### Tag System

CheckMate uses tags to create flexible associations between hosts and rules:

- **Host Tags**: Group hosts by environment, purpose, or any custom category
- **Rule Tags**: Target rules to specific groups of hosts
- **Tag Matching**:
  - Rules apply to hosts when they share at least one matching tag
  - Rules with no tags apply to all hosts
  - Hosts with no tags only match rules with no tags

Example:
```yaml
hosts:
  - host: "prod-web"
    tags: ["prod", "web"]
    checks:
      - port: "80"
        protocol: HTTP
        interval: 5s

  - host: "dev-web"
    tags: ["dev", "web"]
    checks:
      - port: "80"
        protocol: HTTP
        interval: 30s

rules:
  - name: "prod_latency"
    condition: "responseTime > 2s"
    tags: ["prod"]           # Only applies to prod hosts
  
  - name: "web_downtime"
    condition: "downtime > 30s"
    tags: ["web"]           # Applies to all web hosts
  
  - name: "dev_alert"
    condition: "downtime > 5m"
    tags: ["dev"]           # Only applies to dev hosts
```

## Metrics

CheckMate exposes Prometheus metrics at `:9100/metrics` including:
- `checkmate_check_success`: Service availability (1 = up, 0 = down)
- `checkmate_check_latency_milliseconds`: Response time gauge
- `checkmate_check_latency_milliseconds_histogram`: Response time distribution

Labels included with metrics:
- `host`: Target hostname
- `port`: Service port
- `protocol`: Check protocol
- `tags`: Comma-separated list of host tags

Example Prometheus queries:
```promql
# Filter checks by tag
checkmate_check_success{tags=~".*prod.*"}

# Average response time for production web servers
avg(checkmate_check_latency_milliseconds{tags=~".*prod.*", tags=~".*web.*"})

# 95th percentile latency for internal services
histogram_quantile(0.95, sum(rate(checkmate_check_latency_milliseconds_histogram{tags=~".*internal.*"}[5m])) by (le))
```

## Logging

CheckMate uses structured logging with the following fields:
- Basic check information:
  - `host`: Target hostname
  - `port`: Service port
  - `protocol`: Check protocol
  - `success`: Check result (true/false)
  - `responseTime_us`: Response time in microseconds
  - `tags`: Array of host tags
- Rule evaluation:
  - `rule`: Rule name
  - `ruleTags`: Tags assigned to the rule
  - `hostTags`: Tags assigned to the host
  - `condition`: Rule condition
  - `downtime`: Current downtime duration
  - `responseTime`: Last check response time

Example log output:
```json
{
  "level": "info",
  "ts": "2024-03-21T15:04:05.789Z",
  "caller": "checkmate/main.go:123",
  "msg": "Check succeeded",
  "host": "prod-web-01",
  "port": "80",
  "protocol": "HTTP",
  "responseTime_us": 150000,
  "success": true,
  "tags": ["prod", "web", "internal"]
}

{
  "level": "warn",
  "ts": "2024-03-21T15:04:05.789Z",
  "caller": "checkmate/main.go:234",
  "msg": "Rule condition met",
  "rule": "high_latency",
  "ruleTags": ["prod"],
  "hostTags": ["prod", "web"],
  "condition": "responseTime > 5s",
  "downtime": "0s",
  "responseTime": "6.2s"
}
```

## Development

### Available Make Commands
```bash
make dev          # Setup development environment
make lint         # Run linter
make test         # Run tests
make coverage     # Generate test coverage report
make docker-build # Build Docker image
make help         # Show all available commands
```

### Adding New Protocols

1. Create a new checker in `pkg/checkers/`:
```go
type NewProtocolChecker struct {
    protocol Protocol
}

func (c *NewProtocolChecker) Check(ctx context.Context, address string) CheckResult {
    // Implement protocol check
}
```

2. Register in `pkg/checkers/checker.go`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Roadmap

- [ ] Additional protocol support (HTTPS, TLS verification)
- [ ] Notification system integration (Slack, Email, etc.)
- [ ] Configurable notification thresholds
- [ ] database support
- [ ] Docker container
- [ ] Web UI for monitoring (MAYBE) 
- [X] Kubernetes readiness/liveness probe support
- [x] Multiple host monitoring
- [x] Multi-protocol per host
- [x] Service tagging system