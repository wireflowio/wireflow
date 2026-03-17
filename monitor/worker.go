package monitor

import (
	"context"
	"log"
	"os"
	"time"
	"wireflow/internal"
	"wireflow/monitor/collector"
)

const DiskCollectorPath = "/"

// MetricWorker 定义采集管理结构
type MetricWorker struct {
	stopChan            chan struct{}
	cpuCollector        collector.MetricCollector
	memCollector        collector.MetricCollector
	diskCollector       collector.MetricCollector
	peerStatusCollector collector.MetricCollector
	trafficCollector    collector.MetricCollector
}

func NewMetricWorker() *MetricWorker {
	return &MetricWorker{
		stopChan:            make(chan struct{}),
		cpuCollector:        collector.NewCPUCollector(),
		diskCollector:       collector.NewDiskCollector([]string{DiskCollectorPath}),
		memCollector:        collector.NewMemCollector(),
		peerStatusCollector: collector.NewPeerStatusCollector(),
		trafficCollector:    collector.NewTrafficCollector(),
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
				mw.collectAndSetCPUMetrics()
				mw.collectAndSetMemMetrics()
				mw.collectAndSetDiskMetrics()
			case <-mw.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (mw *MetricWorker) StartPeerStatusMetrics(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// 1. 调用 Peer 状态采集器（注意这里要换成你对应的 Collector 实例）
				metrics, err := mw.peerStatusCollector.Collect()
				if err != nil {
					// log.Printf("failed to collect peer status: %v", err)
					continue
				}

				// 2. 核心操作：在写入这一批次数据前，先重置 Gauge，防止已下线的节点残留
				internal.PeerStatus.Reset()
				// 如果你顺便采集了流量，也在这里重置对应的 Gauge/Counter
				// exporter.PeerBytesTransmit.Reset()
				// exporter.PeerBytesReceive.Reset()

				// 3. 遍历并设置指标
				for _, m := range metrics {
					val, ok := m.Value().(float64)
					if !ok {
						continue
					}

					labels := m.Labels()

					// 根据指标名称分发数据
					switch m.Name() {
					case "peer_status":
						// 确保这里的 Label 顺序与你定义 PeerStatus 时一致
						// 建议：peer_id, ip, alias
						internal.PeerStatus.WithLabelValues(
							labels["peer_id"],
							labels["ip"],
							labels["alias"],
						).Set(val)

						//TODO should Add traffic datas
						//case "peer_receive_bytes":
						//	exporter.PeerBytesReceive.WithLabelValues(
						//		labels["peer_id"],
						//		labels["ip"],
						//		labels["alias"],
						//	).Set(val)
						//
						//case "peer_transmit_bytes":
						//	exporter.PeerBytesTransmit.WithLabelValues(
						//		labels["peer_id"],
						//		labels["ip"],
						//		labels["alias"],
						//	).Set(val)
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

func (mw *MetricWorker) collectAndSetCPUMetrics() {
	metrics, err := mw.cpuCollector.Collect()
	if err != nil {
		// 记录日志，不要让程序崩掉
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("failed to get hostname: %v", err)
		return
	}

	for _, m := range metrics {
		val, ok := m.Value().(float64)
		if !ok {
			continue
		}

		switch m.Name() {
		case "cpu_usage_total":
			internal.NodeCpuUsage.WithLabelValues("ws-01", hostname).Set(val)

		case "cpu_usage_core":
			internal.NodeCoreUsage.WithLabelValues("ws-01", hostname).Set(val)
		}
	}
}

func (mw *MetricWorker) collectAndSetMemMetrics() {
	metrics, err := mw.memCollector.Collect()
	if err != nil {
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("failed to get hostname: %v", err)
		return
	}

	for _, m := range metrics {
		val, ok := m.Value().(float64)
		if !ok {
			continue
		}

		labels := m.Labels()

		switch m.Name() {
		case "memory_total":
			internal.NodeMemUsage.WithLabelValues("ws-01", hostname).Set(val)

		case "memory_used":
			if labels == nil {
				labels = map[string]string{"type": "used"}
			} else {
				labels["type"] = "used"
			}
			internal.NodeMemUsage.WithLabelValues("ws-01", hostname, "used").Set(val)

		case "memory_free":
			if labels == nil {
				labels = map[string]string{"type": "free"}
			} else {
				labels["type"] = "free"
			}
			internal.NodeMemUsage.WithLabelValues("ws-01", hostname, "free").Set(val)

		case "memory_used_percent":
			internal.NodeMemUsage.WithLabelValues("ws-01", hostname, "percent").Set(val)
		}
	}
}

func (mw *MetricWorker) collectAndSetDiskMetrics() {
	metrics, err := mw.diskCollector.Collect()
	if err != nil {
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("failed to get hostname: %v", err)
		return
	}

	for _, m := range metrics {
		val, ok := m.Value().(float64)
		if !ok {
			continue
		}

		labels := m.Labels()
		path := labels["path"]
		if path == "" {
			path = "/"
		}

		switch m.Name() {
		case "disk_total":
			internal.WorkspaceResourceTotal.WithLabelValues("ws-01", hostname, "disk").Set(val)

		case "disk_used":
			internal.WorkspaceResourceUsage.WithLabelValues("ws-01", hostname, "disk").Set(val)

		case "disk_free":
			// 如果需要监控空闲磁盘空间
			// internal.DiskFree.WithLabelValues("ws-01", "macbook-pro.local", path).Set(val)

		case "disk_used_percent":
			// 如果需要监控磁盘使用率
			// internal.DiskUsagePercent.WithLabelValues("ws-01", "macbook-pro.local", path).Set(val)
		}
	}
}

func (mw *MetricWorker) StartTrafficMetrics(ctx context.Context, interval time.Duration) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("failed to get hostname: %v", err)
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				metrics, err := mw.trafficCollector.Collect()
				if err != nil {
					continue
				}

				for _, m := range metrics {
					val, ok := m.Value().(float64)
					if !ok {
						continue
					}

					labels := m.Labels()
					peerID := labels["peer"]
					if peerID == "" {
						peerID = "unknown"
					}

					switch m.Name() {
					case "traffic_in":
						internal.PeerTrafficBytes.WithLabelValues(
							"default_ws",
							hostname,
							peerID,
							"rx",
						).Add(val)

					case "traffic_out":
						internal.PeerTrafficBytes.WithLabelValues(
							"default_ws",
							hostname,
							peerID,
							"tx",
						).Add(val)

					case "all_traffic_in":
						internal.WorkspaceResourceUsage.WithLabelValues(
							"default_ws",
							hostname,
							"bandwidth_rx",
						).Set(val)

					case "all_traffic_out":
						internal.WorkspaceResourceUsage.WithLabelValues(
							"default_ws",
							hostname,
							"bandwidth_tx",
						).Set(val)
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
