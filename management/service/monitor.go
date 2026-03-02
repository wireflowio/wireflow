package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"wireflow/internal/log"
	"wireflow/monitor"
	"wireflow/pkg/utils"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type MonitorService interface {
	GetTopologySnapshot(ctx context.Context) ([]monitor.PeerSnapshot, error)
}

type monitorService struct {
	api     v1.API
	log     *log.Logger
	timeout time.Duration
}

// ... existing code ...

type MonitorServiceOptions struct {
	// Address Prometheus / VictoriaMetrics PromQL API 地址
	// 例："http://localhost:8428"
	Address string

	// Timeout 单次查询超时；当 ctx 本身未设置 deadline 时生效
	Timeout time.Duration

	// Logger 可选：不传则使用默认 logger
	Logger *log.Logger
}

func NewMonitorService(address string) (MonitorService, error) {
	// 兼容旧签名：内部转到 Options 版本
	return NewMonitorServiceWithOptions(MonitorServiceOptions{
		Address: address,
		Timeout: 5 * time.Second,
	})
}

func NewMonitorServiceWithOptions(opts MonitorServiceOptions) (MonitorService, error) {
	if opts.Address == "" {
		return nil, fmt.Errorf("monitor service: empty address")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = log.GetLogger("vm-service")
	}

	client, err := api.NewClient(api.Config{
		Address: opts.Address,
	})
	if err != nil {
		return nil, err
	}

	return &monitorService{
		api:     v1.NewAPI(client),
		log:     opts.Logger,
		timeout: opts.Timeout,
	}, nil
}

// ... existing code ...

//// ensureTimeout：如果 ctx 没有 deadline，则注入默认超时；否则原样返回
//func (v *monitorService) ensureTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
//	if _, ok := ctx.Deadline(); ok {
//		return ctx, func() {}
//	}
//	return context.WithTimeout(ctx, v.timeout)
//}
//
//// queryInstant 执行一次 PromQL Instant Query，并统一处理 warnings
//func (v *monitorService) queryInstant(ctx context.Context, promql string, ts time.Time) (model.Value, error) {
//	ctx, cancel := v.ensureTimeout(ctx)
//	defer cancel()
//
//	val, warnings, err := v.api.Query(ctx, promql, ts)
//	if err != nil {
//		return nil, err
//	}
//	for _, w := range warnings {
//		// 避免 fmt.Printf，统一走 logger
//		v.log.Warn("promql warning", "warning", w, "query", promql)
//	}
//	return val, nil
//}

// GetPeerStatus 获取所有 Peer 的拓扑状态
func (v *monitorService) GetTopologySnapshot(ctx context.Context) ([]monitor.PeerSnapshot, error) {
	// 1. 查询所有以 wireflow_node_ 开头的指标
	query := `last_over_time({__name__=~"wireflow_node_.*"}[5m])`
	vector, err := v.QueryByTime(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[string]*monitor.PeerSnapshot)

	for _, s := range vector {
		nodeID := string(s.Metric["node_id"])
		metricName := string(s.Metric["__name__"])
		val := float64(s.Value)

		// 初始化节点
		if _, ok := nodeMap[nodeID]; !ok {
			nodeMap[nodeID] = &monitor.PeerSnapshot{
				ID:          nodeID,
				Name:        string(s.Metric["node_id"]),
				InternalIP:  string(s.Metric["ip"]),
				Status:      "online",
				HealthLevel: "success",
				Metrics:     make(map[string]string),
			}
		}

		// 2. 自动格式化并存入 Map
		// 我们去掉前缀 "wireflow_node_" 让前端拿到的 Key 更简洁
		shortName := strings.TrimPrefix(metricName, "wireflow_node_")
		nodeMap[nodeID].Metrics[shortName] = utils.AutoFormat(metricName, val)

		// 3. 特殊逻辑：根据 CPU 自动判定健康度
		if shortName == "cpu_usage_percent" {
			if val > 80 {
				nodeMap[nodeID].HealthLevel = "warning"
			}
			if val > 95 {
				nodeMap[nodeID].HealthLevel = "error"
			}
		}
	}

	// 转为切片
	var result []monitor.PeerSnapshot
	for _, node := range nodeMap {
		result = append(result, *node)
	}
	return result, nil
}

// QueryByTime 执行瞬时查询 (Instant Query)
// query: PromQL 语句，例如 `last_over_time(peer_status[5m])`
// t: 目标时间点。传入 time.Now() 查当前，传入过去的时间戳则查历史。
func (v *monitorService) QueryByTime(ctx context.Context, query string, t time.Time) (model.Vector, error) {
	// 1. 调用底层的 v1.API。注意：Query 接口返回的是指定时间点 t 的“快照”
	result, warnings, err := v.api.Query(ctx, query, t)
	if err != nil {
		return nil, fmt.Errorf("promql query error: %v", err)
	}

	// 2. 打印 VM 返回的潜在警告（如查询超时、数据部分缺失）
	for _, w := range warnings {
		fmt.Printf("VM Warning: %v\n", w)
	}

	// 3. 类型断言。Instant Query 的结果通常是 Vector (瞬时向量)
	// 如果你查的是一个不存在的指标，这里会返回一个空的 Vector 而不是 error
	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T, expected model.Vector", result)
	}

	return vector, nil
}
