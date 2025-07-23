package collector

import (
	"fmt"
	"golang.zx2c4.com/wireguard/wgctrl"
	"time"
)

type TrafficCollector struct {
}

func (t *TrafficCollector) Name() string {
	return "TrafficCollector"
}

func (t *TrafficCollector) Collect() ([]Metric, error) {
	// 模拟收集流量数据
	metrics := []Metric{
		NewSimpleMetric("traffic_in", 1000, map[string]string{"unit": "bytes"}, time.Now()),
		NewSimpleMetric("traffic_out", 500, map[string]string{"unit": "bytes"}, time.Now()),
	}

	// 在这里可以添加实际的流量收集逻辑
	ticker := time.NewTimer(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// get traffic data from wireguard
			ctr, _ := wgctrl.New()
			devices, _ := ctr.Devices()
			if len(devices) > 0 {
				peers := devices[0].Peers
				var allTrafficeIn int64
				var allTrafficeOut int64
				for _, peer := range peers {
					allTrafficeIn += peer.ReceiveBytes
					allTrafficeOut += peer.TransmitBytes
					metrics = append(metrics, NewSimpleMetric(
						fmt.Sprintf("%s_%s", peer.PublicKey, "traffic_in"),
						peer.ReceiveBytes,
						map[string]string{"peer": peer.PublicKey.String()},
						time.Now(),
					))
					metrics = append(metrics, NewSimpleMetric(
						fmt.Sprintf("%s_%s", peer.PublicKey, "traffic_out"), peer.TransmitBytes,
						map[string]string{"peer": peer.PublicKey.String()},
						time.Now(),
					))
				}

				metrics = append(metrics, NewSimpleMetric(
					"all_traffic_in",
					allTrafficeIn,
					map[string]string{"device": devices[0].Name},
					time.Now(),
				))

				metrics = append(metrics, NewSimpleMetric(
					"all_traffic_out",
					allTrafficeOut,
					map[string]string{"device": devices[0].Name},
					time.Now(),
				))

			}

		}
	}

	return metrics, nil
}
