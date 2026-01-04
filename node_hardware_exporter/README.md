# Node Hardware Exporter for Prometheus

一个用于抓取硬件监控指标并以 Prometheus 格式导出的服务。

## 收集的指标

Node Hardware Exporter 收集以下硬件相关指标：

1. **Hardware Monitor (hwmon)** - 通过`/sys/class/hwmon`收集 CPU 温度、风扇转速、电压等硬件传感器数据
2. **Thermal Zone** - 收集系统散热区域温度和冷却设备状态
3. **Power Supply** - 收集电源供应信息，如电池容量、充电状态等
4. **Watchdog** - 收集系统看门狗设备状态
5. **DMI** - 收集系统硬件信息，如 BIOS 信息、主板信息、产品信息等
6. **NVMe** - 收集 NVMe 存储设备信息
7. **MDADM** - 收集 Linux 软件 RAID 信息
8. **DRM** - 收集显卡硬件信息和性能数据
9. **RAPL** - 收集 CPU 能耗信息
10. **EDAC** - 收集内存错误检测与纠正信息
11. **Perf** - 收集 CPU 性能计数器信息

## 运行

```
./node_hardware_exporter
```

默认情况下，node_hardware_exporter 在端口 9124 上提供指标。

## 配置

配置位于`config/export.yaml`文件，可以修改端口等设置。
