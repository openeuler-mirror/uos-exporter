# Systemd Exporter for Prometheus

Simple server that scrapes systemd metrics and exports them as Prometheus metrics.

## 概述

Systemd Exporter 用于监控 systemd 单元的状态和资源使用情况。它通过 systemd dbus 接口收集 systemd 单元的指标，如服务状态、重启次数、任务数量等，并将这些指标导出为 Prometheus 可读取的格式。

## 功能特点

- 监控所有 systemd 单元的状态（active、inactive、failed 等）
- 收集服务的启动时间、重启次数和任务数量
- 收集服务的 IP 流量统计（字节和包计数）
- 收集 socket 单元的连接统计
- 收集 timer 单元的触发时间
- 收集 watchdog 相关指标
- 可选择性地收集 systemd-resolved 相关指标

## 指标项

所有指标都有 `name` 标签，其中包含 systemd 单元名称。例如 `name="bluetooth.service"` 或 `name="systemd-coredump.socket"`。

主要的指标项包括：

| 指标名称                                  | 类型    | 描述                  |
| ----------------------------------------- | ------- | --------------------- |
| systemd_unit_state                        | Gauge   | systemd 单元的状态    |
| systemd_unit_start_time_seconds           | Gauge   | 单元启动时间          |
| systemd_unit_tasks_current                | Gauge   | 当前任务数            |
| systemd_unit_tasks_max                    | Gauge   | 最大任务数            |
| systemd_service_restart_total             | Counter | 服务重启次数          |
| systemd_service_ip_ingress_bytes          | Counter | 服务入站流量字节数    |
| systemd_service_ip_egress_bytes           | Counter | 服务出站流量字节数    |
| systemd_socket_accepted_connections_total | Counter | socket 接受的连接总数 |
| systemd_socket_current_connections        | Gauge   | socket 当前连接数     |
| systemd_timer_last_trigger_seconds        | Gauge   | 计时器最后触发时间    |

## 使用方法

运行 systemd_exporter 后，可以通过访问 `http://localhost:9558/metrics` 查看所有导出的指标。
