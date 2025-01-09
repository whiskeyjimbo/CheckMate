# CheckMate

![License](https://img.shields.io/badge/license-GPLv3-blue.svg)
![Go Version](https://img.shields.io/badge/language-go-blue.svg)

CheckMate is a service monitoring tool written in Go that provides real-time health checks and metrics for infrastructure. It supports multiple protocols, customizable rules, and Prometheus integration.

DISCLAIMER: This is a personal project and is not meant to be used in a production environment as it is not feature complete nor secure nor tested and under heavy development.

## Features

### Core Features
- Multi-protocol support (TCP, HTTP, HTTPS with cert validation, SMTP, DNS)
- Hierarchical configuration (Sites → Groups → Hosts → Checks)
- High availability monitoring with configurable modes
- Configurable check intervals per service
- Prometheus metrics integration
- Simple Rule-based monitoring with custom conditions
- Flexible notification system
- Service tagging system
- TLS certificate expiration monitoring

### High Availability Monitoring

Groups support two monitoring modes that can be configured at different levels:

- **All Mode (Default)**
  - Group is considered "up" if any host is responding
  - Rules only trigger when all hosts are down
  - For redundant services where one available host is sufficient

- **Any Mode**
  - Group monitoring tracks all hosts individually
  - Rules trigger when any host goes down
  - Suitable for services where each host's availability is critical

Rule modes can be configured at three levels (in order of precedence):
1. Check level - Overrides group settings for specific checks
2. Group level - Default for all checks in the group
3. Default - Falls back to "all" mode if not specified

## Configuration

### Site Configuration
- `monitorSite`: Name of the monitoring instance
- `sites`: List of infrastructure sites to monitor
  - `name`: Site identifier
  - `tags`: Site-level tags
  - `groups`: List of service groups

### Group Configuration
- `name`: Group identifier
- `tags`: Group-level tags (combined with site tags)
- `hosts`: List of hosts to monitor
  - `host`: Hostname or IP
  - `tags`: Host-specific tags
- `checks`: Service checks applied to all hosts
  - `port`: Port number
  - `protocol`: TCP, HTTP, SMTP, or DNS
  - `interval`: Check frequency (e.g., "30s", "1m")
  - `tags`: Check-specific tags
  - `ruleMode`: Override group's rule mode
  - `verifyCert`: Enable certificate checking
- `ruleMode`: Group-level rule mode ("all" or "any")

### Rule Configuration
Rules define conditions for generating notifications. Each rule requires a `type` field:

```yaml
# Standard Rule Example
- name: "prod_service_degraded"
  type: "standard"
  condition: "responseTime > 1000 || downtime > 0"
  tags: ["prod", "critical"]
  notifications: ["log"]
# Certificate Rule Example
- name: "cert_expiring_soon"
  type: "cert"
  minDaysValidity: 30
  tags: ["https-api"]
  notifications: ["log"]
```

Common Fields:
- `name`: Rule identifier
- `type`: Either "standard" or "cert"
- `tags`: Tags to match against groups/checks
- `notifications`: Notification types to use

Type-specific Fields:
- Standard Rules:
  - `condition`: Expression using `downtime` and `responseTime` variables
- Certificate Rules:
  - `minDaysValidity`: Days before expiration to trigger alert

### Notification Configuration
- `type`: Notification type ("log", more coming soon)

## Metrics

CheckMate exposes Prometheus metrics at `:9100/metrics`

### Core Metrics
- `checkmate_host_check_status`: Service availability (1 = up, 0 = down)
- `checkmate_host_check_latency_milliseconds`: Response time in milliseconds
- `checkmate_check_latency_histogram_seconds`: Response time distribution
- `checkmate_hosts_up`: Number of hosts up in a group
- `checkmate_hosts_total`: Total number of hosts in a group
- `checkmate_cert_expiry_days`: Days until certificate expiration

### Graph Visualization Metrics (In Development)
> Note: These metrics are designed for Grafana's Node Graph visualization and are currently in flux

- `checkmate_node_info`: Node information for graph visualization
  - Labels: id, type (site/group/host), name, tags, port, protocol
  - Values: 1 for active nodes, 0 for inactive

- `checkmate_edge_info`: Edge information with latency
  - Labels: source, target, type, metric, port, protocol
  - Values: latency in milliseconds

Example Prometheus queries:
```promql
# Filter checks by site
checkmate_check_success{site="mars-lab"}

# Average response time for production APIs
avg(checkmate_check_latency_milliseconds{tags=~".*prod.*", tags=~".*api.*"})

# 95th percentile latency by site
histogram_quantile(0.95, sum(rate(checkmate_check_latency_milliseconds_histogram[5m])) by (le, site))

# Host availability ratio per group
sum(checkmate_hosts_up) by (id) / sum(checkmate_hosts_total) by (id)

# Graph Visualization (In Development)
checkmate_node_info{type="host", port="443", protocol="HTTPS"}
avg(checkmate_edge_info{type="contains", metric="latency"}) by (source, target, port, protocol)
```

### Grafana Node Graph Setup (In Development)
To visualize your infrastructure in Grafana's Node Graph:

1. Create a new Node Graph panel
2. Configure the Node Query:
   ```promql
   checkmate_node_info
   ```
3. Configure the Edge Query:
   ```promql
   checkmate_edge_info{metric="latency"}
   ```
4. Set transformations:
   - Nodes: Use 'id' for node ID, 'type' for node class
   - Edges: Use 'source' and 'target' for connections

> Note: Graph visualization features are in flux and the query/configuration interface may change

## Health Checks

CheckMate provides Kubernetes-compatible health check endpoints:

- `/health/live` - Liveness probe
  - Returns 200 OK when the service is running

- `/health/ready` - Readiness probe
  - Returns 200 OK when ready to receive traffic
  - Returns 503 Service Unavailable during initialization

All health check endpoints are served on port 9100 alongside metrics.

## Mini Roadmap

- [ ] Notification system expansion (Slack, Email)
- [ ] Configurable notification thresholds
- [ ] Database support for historical data
- [ ] Web UI for monitoring (MAYBE)

## Completed
- [x] Env Variables for config
- [x] Dockerfile for dev
- [x] Additional protocol support (HTTPS, TLS verification)
- [x] Kubernetes readiness/liveness probe support
- [x] Multiple host monitoring
- [x] Multi-protocol per host
- [x] Service tagging system
- [x] Site-based infrastructure organization
- [x] High availability group monitoring

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Development

### Prerequisites
- Go 1.21 or higher
- [air](https://github.com/air-verse/air) for live reloading (optional)

### Live Reloading
For development with automatic rebuilding on code changes:

1. Install Air:
```bash
go install github.com/air-verse/air@latest
```

2. Run with Air:
```bash
air
```