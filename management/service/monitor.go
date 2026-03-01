package service

import (
	"context"
	"fmt"
	"time"
	"wireflow/internal/log"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type MonitorService interface {
	// GetPeerStatus 获取所有 Peer 的拓扑状态（瞬时向量）。
	// 说明：
	// - 返回值 model.Vector：每个元素是一条 time series 的“当前值”
	// - 通常用于展示在线/离线状态、拓扑连通性等
	GetPeerStatus(ctx context.Context) (model.Vector, error)

	// GetNodeUseAge 获取节点自身资源使用情况（CPU/内存/运行时长）。
	// 返回说明：
	// - 返回值 model.Vector：包含多条指标序列（不同 __name__），每条序列是一个“当前值”
	// - 典型包含：
	//   - wireflow_node_cpu_usage_percent
	//   - wireflow_node_memory_bytes
	//   - wireflow_node_uptime_seconds
	// 注意：
	// - PromQL 返回的是“瞬时向量”，因此建议配合 last_over_time() 获取窗口内最后样本，避免 scrape 间隔导致的空值
	GetNodeUseAge(ctx context.Context) (model.Vector, error)
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

// ensureTimeout：如果 ctx 没有 deadline，则注入默认超时；否则原样返回
func (v *monitorService) ensureTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, v.timeout)
}

// queryInstant 执行一次 PromQL Instant Query，并统一处理 warnings
func (v *monitorService) queryInstant(ctx context.Context, promql string, ts time.Time) (model.Value, error) {
	ctx, cancel := v.ensureTimeout(ctx)
	defer cancel()

	val, warnings, err := v.api.Query(ctx, promql, ts)
	if err != nil {
		return nil, err
	}
	for _, w := range warnings {
		// 避免 fmt.Printf，统一走 logger
		v.log.Warn("promql warning", "warning", w, "query", promql)
	}
	return val, nil
}

// GetPeerStatus 获取所有 Peer 的拓扑状态
func (v *monitorService) GetPeerStatus(ctx context.Context) (model.Vector, error) {
	// 使用 PromQL 查询最新快照：
	// - exporter 里定义的指标名是 wireflow_peer_status
	// - last_over_time 用于取窗口内最后一个样本，避免 scrape 间隔导致的“空洞”
	query := `last_over_time(wireflow_peer_status[5m])`

	result, err := v.queryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}
	return vector, nil
}

func (v *monitorService) GetNodeUseAge(ctx context.Context) (model.Vector, error) {
	// 一次性拉取节点资源使用相关的 3 个指标（用 __name__ 正则匹配）。
	// 这里用 last_over_time 保证即使某次抓取抖动，也能拿到最近窗口内的最后值。
	//
	// 你在 exporter/registry.go 里定义的指标名分别是：
	// - wireflow_node_cpu_usage_percent
	// - wireflow_node_memory_bytes
	// - wireflow_node_uptime_seconds
	query := `
last_over_time({__name__=~"wireflow_node_(cpu_usage_percent|memory_bytes|uptime_seconds)"}[5m])
`

	result, err := v.queryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}
	return vector, nil
}
