package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"strconv"
)

type PrometheusStorage struct {
	database string
}

// Store push data to pushgateway, prometheus will pull data from the pushgateway.
func (s *PrometheusStorage) Store(metrics []Metric) error {
	for _, m := range metrics {
		data := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: m.Name(),
			Help: m.Help(),
		})

		if _, b := m.Value().(float64); b {
			data.Set(m.Value().(float64))
		} else {
			value, _ := strconv.ParseFloat(fmt.Sprintf("%d", m.Value().(int64)), 64)
			data.Set(value)
		}
		data.SetToCurrentTime()

		if err := push.New("http://pushgateway.linkany.io:9091", "linkany-metrics").
			Collector(data).
			Grouping("linkany", "metric").
			Push(); err != nil {
			fmt.Println("Could not push completion time to Pushgateway:", err)
		}
	}

	return nil
}

func (s *PrometheusStorage) Query(query Query) ([]Metric, error) {
	return nil, nil
}
