package monitor

import (
	"context"
	"fmt"
	"time"
	"wireflow/monitor/collector"
	exporter "wireflow/monitor/wireflow-exporter"
)

// MetricWorker 定义采集管理结构
type MetricWorker struct {
	interval     time.Duration
	stopChan     chan struct{}
	cpuCollector collector.MetricCollector
}

func NewMetricWorker() *MetricWorker {
	return &MetricWorker{
		stopChan:     make(chan struct{}),
		cpuCollector: collector.NewCPUCollector(),
	}
}

// Start 背景运行：延迟与链路探测
func (mw *MetricWorker) StartLinkProbing(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// 1. 获取最新的 Peer 列表 (从你的 core 模块获取)
				// targets := core.GetActivePeers()
				// 2. 执行并发探测
				// RunCycle(targets)
				fmt.Println("start link probing")
			case <-mw.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Start 系统指标采集（CPU/MEM）
func (mw *MetricWorker) StartSystemMetrics(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				metrics, err := mw.cpuCollector.Collect()
				if err != nil {
					// 记录日志，不要让程序崩掉
					continue
				}

				for _, m := range metrics {
					// 获取原始值，并断言为 float64
					// 注意：gopsutil 返回的百分比通常已经是 float64 了
					val, ok := m.Value().(float64)
					if !ok {
						// 如果断言失败，记录日志或跳过，防止程序崩溃
						// log.Printf("metric %s has invalid value type", m.Name())
						continue
					}

					switch m.Name() {
					case "cpu_usage_total":
						exporter.NodeCpuUsage.Set(val)

					case "cpu_usage_core":
						coreID := m.Labels()["core"]
						exporter.NodeCoreUsage.WithLabelValues(coreID).Set(val)
					}
				}
			case <-mw.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}
