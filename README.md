# CheckMate

![License](https://img.shields.io/badge/license-GPLv3-blue.svg)
![Go Version](https://img.shields.io/badge/language-go-blue.svg)

CheckMate is a service monitoring tool written in Go that provides real-time health checks and metrics for infrastructure. It supports multiple protocols, customizable rules, and Prometheus integration.

DISCLAIMER: This is a personal project and is not meant to be used in a production environment as it is not feature complete nor secure nor tested and under heavy development. 

## Features

### Core Features
- Multi-protocol support (TCP, HTTP, SMTP, DNS)
- Hierarchical configuration (Sites → Hosts → Checks)
- Configurable check intervals per service
- Prometheus metrics integration
- Rule-based monitoring with custom conditions
- Flexible notification system with rule-specific routing

### Metrics & Monitoring
- Service availability status
- Response time measurements
- Rule-based alerting with customizable conditions
- Prometheus-compatible metrics endpoint
- Downtime tracking
- Latency histograms and gauges

### Technical Features
- YAML-based configuration
- Modular architecture for easy extension using interfaces
- Site-based infrastructure organization
- Tag inheritance (site tags are inherited by hosts)

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/whiskeyjimbo/CheckMate.git
cd CheckMate
```

2. Build using Make:
```bash
make build
```

3. Run using Make:
```bash
make run config.yaml ## see configuration below for more details
```

## Configuration

CheckMate is configured using a YAML (default: ./config.yaml) file. Here's a complete example:

```yaml
sites:
  - name: us-east-1
    tags: ["prod", "aws", "use1"]
    hosts:
      - host: api.example.com
        tags: ["public"]
        checks:
          - port: "443"
            protocol: HTTP
            interval: 30s
            tags: ["https", "api"]
          - port: "22"
            protocol: TCP
            interval: 1m
            tags: ["ssh"]

  - name: eu-west-1
    tags: ["prod", "aws", "euw1"]
    hosts:
      - host: eu.example.com
        tags: ["api", "public"]
        checks:
          - port: "443"
            protocol: HTTP
            interval: 30s

rules:
  - name: high_latency_warning
    condition: "responseTime > 2s"
    tags: ["prod"]
    notifications: ["log"]
  
  - name: critical_downtime
    condition: "downtime > 5m"
    tags: ["prod"]
    notifications: ["log", "slack"]

notifications:
  - type: "log"
```

### Site Configuration
- `name`: Unique identifier for the site
- `tags`: List of tags inherited by all hosts in the site
- `hosts`: List of hosts in this site

### Host Configuration
- `host`: The hostname or IP to monitor
- `tags`: Additional tags specific to this host (combined with site tags)
- `checks`: List of service checks
  - `port`: Port number to check
  - `protocol`: One of: TCP, HTTP, SMTP, DNS
  - `interval`: Check frequency (e.g., "30s", "1m")
  - `tags`: Additional tags specific to this check (combined with site and host tags)

### Rule Configuration
- `name`: Unique rule identifier
- `condition`: Expression to evaluate (uses responseTime and downtime variables)
- `tags`: List of tags to match against hosts
- `notifications`: List of notification types to use when rule triggers
  - If omitted, all configured notifiers will be used

### Notification Configuration
- `type`: Type of notification ("log", with more coming soon)
- Each notification type can have its own configuration options

## Metrics

CheckMate exposes Prometheus metrics at `:9100/metrics` including:
- `checkmate_check_success`: Service availability (1 = up, 0 = down)
- `checkmate_check_latency_milliseconds`: Response time gauge
- `checkmate_check_latency_milliseconds_histogram`: Response time distribution

Labels included with metrics:
- `site`: Site name
- `host`: Target hostname
- `port`: Service port
- `protocol`: Check protocol
- `tags`: Comma-separated list of combined site and host tags

Example Prometheus queries:
```promql
# Filter checks by site
checkmate_check_success{site="us-east-1"}

# Average response time for production APIs
avg(checkmate_check_latency_milliseconds{tags=~".*prod.*", tags=~".*api.*"})

# 95th percentile latency by site
histogram_quantile(0.95, sum(rate(checkmate_check_latency_milliseconds_histogram[5m])) by (le, site))
```

## Health Checks

CheckMate provides Kubernetes-compatible health check endpoints:

- `/health/live` - Liveness probe
  - Returns 200 OK when the service is running

- `/health/ready` - Readiness probe
  - Returns 200 OK when the service is ready to receive traffic
  - Returns 503 Service Unavailable during initialization

All health check endpoints are served on port 9100 alongside the metrics endpoint.

## Logging

CheckMate uses structured logging with the following fields:
- Basic check information:
  - `site`: Site name
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


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Roadmap

- [ ] Additional protocol support (HTTPS, TLS verification)
- [ ] Notification system integration (Slack, Email, etc.)
- [ ] Configurable notification thresholds
- [ ] Database support for historical data
- [ ] Docker container
- [ ] Web UI for monitoring (MAYBE)
- [x] Kubernetes readiness/liveness probe support
- [x] Multiple host monitoring
- [x] Multi-protocol per host
- [x] Service tagging system
- [x] Site-based infrastructure organization

## Technical Details

### Implementing a New Checker

To add support for a new protocol, implement the Checker interface:

```go
// 1. Create a new type for your checker
type MyNewChecker struct {
    protocol Protocol
}

// 2. Implement the Checker interface
func (c *MyNewChecker) Check(ctx context.Context, address string) CheckResult {
    // Perform your check logic here
    result := CheckResult{
        Success:      false,
        ResponseTime: 0,
        Error:        nil,
    }
    
    startTime := time.Now()
    
    // Your check implementation
    // For example:
    // - Open a connection
    // - Send/receive data
    // - Validate response
    
    result.ResponseTime = time.Since(startTime)
    result.Success = true // based on check success
    
    return result
}

func (c *MyNewChecker) Protocol() Protocol {
    return c.protocol
}

// 3. Register your checker in pkg/checkers/checker.go
func NewChecker(protocol string) (Checker, error) {
    switch Protocol(protocol) {
    // ... existing protocols ...
    case ProtocolMyNew:
        return &MyNewChecker{protocol: ProtocolMyNew}, nil
    default:
        return nil, fmt.Errorf("unsupported protocol: %s", protocol)
    }
}
```

### Implementing a New Notifier

To add a new notification system, implement the Notifier interface:

```go
// 1. Create a new notification type constant
const MyNewNotification NotificationType = "mynew"

// 2. Create your notifier type
type MyNewNotifier struct {
    // Add any required fields
    client    *myclient.Client
    apiKey    string
}

// 3. Implement the Notifier interface
func (n *MyNewNotifier) Initialize(ctx context.Context) error {
    // Setup your notification client/connection
    n.client = myclient.New(n.apiKey)
    return nil
}

func (n *MyNewNotifier) SendNotification(ctx context.Context, notification Notification) error {
    // Convert the notification to your system's format
    message := MyNotificationFormat{
        Text:     notification.Message,
        Severity: convertLevel(notification.Level),
        Tags:     notification.Tags,
        Metadata: map[string]string{
            "host":     notification.Host,
            "port":     notification.Port,
            "protocol": notification.Protocol,
        },
    }
    
    // Send the notification
    return n.client.Send(ctx, message)
}

func (n *MyNewNotifier) Type() NotificationType {
    return MyNewNotification
}

func (n *MyNewNotifier) Close() error {
    return n.client.Close()
}

// 4. Register your notifier in pkg/notifications/notifier.go
func NewNotifier(notifierType string, opts ...interface{}) (Notifier, error) {
    switch NotificationType(notifierType) {
    // ... existing notifiers ...
    case MyNewNotification:
        if len(opts) > 0 {
            if apiKey, ok := opts[0].(string); ok {
                return &MyNewNotifier{apiKey: apiKey}, nil
            }
        }
        return nil, fmt.Errorf("mynew notifier requires an API key")
    default:
        return nil, fmt.Errorf("unsupported notification type: %s", notifierType)
    }
}
```

### Available Make Commands
```bash
make dev          # Setup development environment
make lint         # Run linter
make test         # Run tests
make coverage     # Generate test coverage report
make docker-build # Build Docker image
make help         # Show all available commands
```