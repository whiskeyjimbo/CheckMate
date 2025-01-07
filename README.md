# CheckMate

![License](https://img.shields.io/badge/license-GPLv3-blue.svg)
![Go Version](https://img.shields.io/badge/language-go-blue.svg)

CheckMate is a service monitoring tool written in Go that provides real-time health checks and metrics for infrastructure. It supports multiple protocols, customizable rules, and Prometheus integration.

DISCLAIMER: This is a personal project and is not meant to be used in a production environment as it is not feature complete nor secure nor tested and under heavy development. 

## Features

### Core Features
- Multi-protocol support (TCP, HTTP, SMTP, DNS)
- Hierarchical configuration (Sites → Groups → Hosts → Checks)
- Group monitoring
- Configurable check intervals per service
- Prometheus metrics integration
- Rule-based monitoring with custom conditions
- Flexible notification system with rule-specific routing

### Metrics & Monitoring
- Service availability status
- Response time measurements
- Prometheus-compatible metrics endpoint
- Downtime tracking
- Latency histograms and gauges

### Other Features
- YAML-based configuration
- Modular architecture for easy extension using interfaces
- Site-based infrastructure organization
- Tag inheritance (site tags are inherited by groups and hosts)

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

CheckMate is configured using a YAML (default: ./config.yaml) file. 
Example configuration:

```yaml
sites:
  - name: "mars-prod"
    tags: ["region-mars", "prod"]
    groups:
      - name: "api-service.dev.com"
        tags: ["prod"]
        hosts:
          - host: "127.0.0.1"
            tags: ["primary"]
          - host: "localhost2"
            tags: ["secondary"]
        checks:
          - port: "443"
            protocol: HTTPS
            interval: 10s
            tags: ["api"]
          - port: "22"
            protocol: TCP
            interval: 10s
            tags: ["ssh"]

rules:
  - name: high_latency_warning
    condition: "responseTime > 5ms"
    tags: ["prod"]
    notifications: ["log"]

  - name: critical_downtime_prod
    condition: "downtime > 15s"
    tags: ["prod"]
    notifications: ["log"]

notifications:
  - type: "log"
```

### Site Configuration
- `name`: Unique identifier for the site
- `tags`: List of tags inherited by all groups in the site
- `groups`: List of service groups in this site

### Group Configuration
- `name`: The group identifier
- `tags`: Additional tags specific to this group (combined with site tags)
- `hosts`: List of hosts in this group
  - `host`: The hostname or IP to monitor
  - `tags`: Additional tags specific to this host
- `checks`: List of service checks applied to all hosts
  - `port`: Port number to check
  - `protocol`: One of: TCP, HTTP, SMTP, DNS
  - `interval`: Check frequency (e.g., "30s", "1m")
  - `tags`: Additional tags specific to this check
  - `ruleMode`: Override group's rule mode for this specific check
- `ruleMode`: How rules are evaluated for this group (optional)
  - `all`: Only fire rules when all hosts are down (default)
  - `any`: Fire rules when any host in the group is down

### Rule Configuration
- `name`: Unique rule identifier
- `condition`: Expression to evaluate (uses responseTime and downtime variables)
- `tags`: List of tags to match against groups
- `notifications`: List of notification types to use when rule triggers
  - If omitted, all configured notifiers will be used

### Notification Configuration
- `type`: Type of notification ("log", with more coming soon)

## High Availability Monitoring

Groups support high availability monitoring with configurable rule modes at both group and check levels:

### All Mode (Default)
- Group is considered "up" if any host is responding
- Rules only trigger when all hosts are down
- Ideal for redundant services where one available host is sufficient

### Any Mode
- Group monitoring tracks all hosts individually
- Rules trigger when any host goes down
- Suitable for services where each host's availability is critical
- Can be set at check level to override group settings

In both modes:
- Response times are averaged across all successful checks in the group (think i will change this later to use host level metrics..)
- Metrics are tracked at both host and group levels
- Prometheus histograms are used for latency tracking
- Notifications include specific failing hosts

Example Prometheus queries for HA monitoring:
```promql
# Count of available hosts in each group
count(checkmate_check_success{group="api-service"} == 1) by (site, group)

# Groups with all hosts down
count(checkmate_check_success{} == 0) by (site, group)

# Average response time across all hosts in a group
avg(checkmate_check_latency_milliseconds) by (site, group)
```

## Metrics

CheckMate exposes Prometheus metrics at `:9100/metrics` including:

### Core Metrics
- `checkmate_check_success`: Service availability (1 = up, 0 = down)
- `checkmate_check_latency_milliseconds`: Response time in milliseconds
- `checkmate_check_latency_milliseconds_histogram`: Response time distribution in milliseconds
- `checkmate_hosts_up`: Number of hosts up in a group (per port/protocol)
- `checkmate_hosts_total`: Total number of hosts in a group (per port/protocol)

### Graph Visualization Metrics (Beta)
> Note: These metrics are designed for Grafana's Node Graph visualization and are currently in  flux

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

# Graph Visualization (Beta)
# Node status
checkmate_node_info{type="host", port="443", protocol="HTTPS"}

# Edge latencies
avg(checkmate_edge_info{type="contains", metric="latency"}) by (source, target, port, protocol)
```

### Grafana Node Graph Setup (Beta)
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
  - Returns 200 OK when the service is ready to receive traffic
  - Returns 503 Service Unavailable during initialization

All health check endpoints are served on port 9100 alongside the metrics endpoint.

## Logging

CheckMate uses structured logging with the following fields:
- Basic check information:
  - `site`: Site name
  - `group`: Target hostname
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
- [x] High availability group monitoring