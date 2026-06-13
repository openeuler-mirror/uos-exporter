# Squid Exporter for Prometheus

Simple server that scrapes Squid metrics from cache manager and exports them as Prometheus metrics.

## Description

This exporter connects to a Squid proxy server and exports its metrics in Prometheus format. It collects information from the Squid cache_object manager such as client and server HTTP statistics, cache performance, service times, and various system resource usage metrics.

## Installation

### Binary installation

Download the latest binary from the releases page.

### From source

```
git clone https://github.com/your-org/squid_exporter.git
cd squid_exporter
go build
```

## Usage

### Command Line

Basic usage:

```
./squid_exporter
```

To specify Squid host and port:

```
./squid_exporter --squid.hostname localhost --squid.port 3128
```

### Configuration

The exporter can be configured using the following CLI parameters:

```
--squid.hostname       Hostname of the Squid server (default: "localhost")
--squid.port           Port of the Squid server (default: 3128)
--squid.login          Login for the Squid server (if authentication is required)
--squid.password       Password for the Squid server (if authentication is required)
--squid.extractTimes   Extract service time metrics (default: true)
```

Or configure the exporter using a YAML configuration file:

```
address: "127.0.0.1"  # Exporter listening address
port: 8090            # Exporter listening port
metricsPath: "/metrics"

# Squid configuration
squid:
  hostname: "localhost"
  port: 3128
  login: ""
  password: ""
  extractTimes: true
```

## Metrics

The exporter collects the following metrics from Squid:

### Client/Server HTTP Metrics

- Client HTTP requests total
- Client HTTP hits total
- Client HTTP errors total
- Server HTTP requests total
- Server HTTP errors total
- And more...

### Service Times

- HTTP request service times
- Cache hits service times
- Cache misses service times
- Near hits service times
- DNS lookups service times

### System Information

- Number of clients accessing cache
- CPU usage
- Memory usage
- File descriptor usage
- Storage metrics
- And more...

## Configure Prometheus to Scrape Squid Exporter

Add the following to your `prometheus.yaml`:

```yaml
scrape_configs:
  - job_name: "squid"
    static_configs:
      - targets: ["localhost:8090"]
```

## Squid Configuration

To allow the exporter to query metrics from Squid, add the following to your squid.conf:

```
# Allow cache manager from localhost
acl prometheus src 127.0.0.1
http_access allow manager prometheus
```

## License

[MIT License](https://opensource.org/licenses/MIT)
