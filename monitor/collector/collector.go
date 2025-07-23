package collector

import "time"

type MetricCollector interface {
	Collect() ([]Metric, error)
	Name() string
}

type Metric interface {
	Name() string
	Value() interface{}
	Labels() map[string]string
	Timestamp() time.Time
}

type SimpleMetric struct {
	name      string
	value     interface{}
	labels    map[string]string
	timestamp time.Time
}

func NewSimpleMetric(name string, value interface{}, labels map[string]string, timestamp time.Time) *SimpleMetric {
	return &SimpleMetric{
		name:      name,
		value:     value,
		labels:    labels,
		timestamp: timestamp,
	}
}

func (m *SimpleMetric) Name() string {
	return m.name
}

func (m *SimpleMetric) Value() interface{} {
	return m.value
}

func (m *SimpleMetric) Labels() map[string]string {
	return m.labels
}

func (m *SimpleMetric) Timestamp() time.Time {
	return m.timestamp
}

type Query struct{}

type Alert struct{}

type Storage interface {
	Store(metrics []Metric) error
	Query(query Query) ([]Metric, error)
}

type Alerter interface {
	Evaluate(metrics []Metric) ([]Alert, error)
	Send(alerts []Alert) error
}
