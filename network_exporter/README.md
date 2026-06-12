# Network Exporter for Prometheus

Simple server that exports network metrics for Prometheus, including ping, mtr, tcp and http checks.

## Metrics

The exporter provides the following metrics:

### Ping Metrics

- ping_status: Ping Status
- ping_rtt_seconds: Round Trip Time in seconds
- ping_rtt_snt_count: Packet sent count
- ping_rtt_snt_fail_count: Packet sent fail count
- ping_rtt_snt_seconds: Packet sent time total
- ping_loss_percent: Packet loss in percent
- ping_targets: Number of active targets
- ping_up: Exporter state

### MTR Metrics

- mtr_rtt_seconds: Round Trip Time in seconds
- mtr_rtt_snt_count: Round Trip Send Package Total
- mtr_rtt_snt_fail_count: Round Trip Send Package Fail Total
- mtr_rtt_snt_seconds: Round Trip Send Package Time Total
- mtr_hops: Number of route hops
- mtr_targets: Number of active targets
- mtr_up: Exporter state

### TCP Metrics

- tcp_connection_seconds: Connection time in seconds
- tcp_connection_status: Connection Status
- tcp_targets: Number of active targets
- tcp_up: Exporter state

### HTTP Metrics

- http_get_seconds: HTTP Get Drill Down time in seconds
- http_get_content_bytes: HTTP Get Content Size in bytes
- http_get_status: HTTP Get Status
- http_get_targets: Number of active targets
- http_get_up: Exporter state

## Usage

```bash
./network-exporter --config=/path/to/config/network-exporter.yaml
```
