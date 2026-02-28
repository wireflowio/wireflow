package monitor

import (
	"context"
	"net/http"
	"time"
	"wireflow/internal/infra"
	"wireflow/internal/log"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MonitorRunner struct {
	log   *log.Logger
	peers *infra.PeerManager
}

func NewMonitorRunner(peers *infra.PeerManager) *MonitorRunner {
	return &MonitorRunner{
		log:   log.GetLogger("monitor"),
		peers: peers,
	}
}

func (r *MonitorRunner) Run() {
	// 1. 初始化监控服务器 (暴露 /metrics)
	http.Handle("/metrics", promhttp.Handler())

	// 2. 启动后台采集协程
	worker := NewMetricWorker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 链路探测：每 15 秒一次
	worker.StartLinkProbing(ctx, 15*time.Second)

	// 系统指标：每 1 分钟一次
	worker.StartSystemMetrics(ctx, 10*time.Second)

	// 3. 主线程 hold 住
	// 注意：ListenAndServe 是阻塞的，所以必须放在最后，或者另开协程
	// 但通常推荐主线程守在 HTTP 服务上，方便接收系统信号
	addr := ":9586"
	println("Wireflow Web Server started on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}

}
