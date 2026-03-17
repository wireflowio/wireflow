// Copyright 2025 The Wireflow Authors, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/mem"
	"sync"
	"time"
	"wireflow/internal/infra"
)

type MemoryCollector struct {
	peerManager *infra.PeerManager // 保持和你项目架构一致

	// 缓存数据
	mu         sync.RWMutex
	memStats   *mem.VirtualMemoryStat
	lastUpdate time.Time
}

func (c *MemoryCollector) Name() string {
	return MemUsage
}

func NewMemCollector() *MemoryCollector {
	c := &MemoryCollector{
		peerManager: infra.NewPeerManager(),
	}
	// 启动后台异步采集协程
	go c.runSnapshotter()
	return c
}

func (c *MemoryCollector) runSnapshotter() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := mem.VirtualMemory()
		if err != nil {
			continue
		}

		c.mu.Lock()
		c.memStats = stats
		c.lastUpdate = time.Now()
		c.mu.Unlock()
	}
}

func (c *MemoryCollector) Collect() ([]Metric, error) {
	c.mu.RLock()
	stats := c.memStats
	c.mu.RUnlock()

	if stats == nil {
		return nil, fmt.Errorf("memory stats not available yet")
	}

	metrics := make([]Metric, 0)
	now := time.Now()

	metrics = append(metrics, NewSimpleMetric(
		"memory_total",
		float64(stats.Total),
		nil,
		now,
		"memory total",
	))

	metrics = append(metrics, NewSimpleMetric(
		"memory_used",
		float64(stats.Used),
		nil,
		now,
		"memory used",
	))

	metrics = append(metrics, NewSimpleMetric(
		"memory_free",
		float64(stats.Free),
		nil,
		now,
		"memory free",
	))

	metrics = append(metrics, NewSimpleMetric(
		"memory_used_percent",
		stats.UsedPercent,
		nil,
		now,
		"memory used_percent",
	))

	return metrics, nil
}
