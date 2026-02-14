package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Pusher wraps Prometheus Pushgateway push operations.
type Pusher struct {
	url      string
	username string
	password string
}

// NewPusher creates a new Pusher for the given Pushgateway URL.
func NewPusher(url, username, password string) *Pusher {
	return &Pusher{url: url, username: username, password: password}
}

// Push pushes the given collectors to Pushgateway under the specified job and grouping keys.
func (p *Pusher) Push(job string, grouping map[string]string, collectors ...prometheus.Collector) error {
	pusher := push.New(p.url, job)
	for k, v := range grouping {
		pusher = pusher.Grouping(k, v)
	}
	for _, c := range collectors {
		pusher = pusher.Collector(c)
	}
	if p.username != "" {
		pusher = pusher.BasicAuth(p.username, p.password)
	}
	if err := pusher.Push(); err != nil {
		return fmt.Errorf("pushgateway push failed: %w", err)
	}
	return nil
}

// TestConnection tests connectivity to the Pushgateway by pushing a dummy metric.
func (p *Pusher) TestConnection() error {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "opshub_pushgateway_test",
		Help: "Test metric for connectivity check",
	})
	g.Set(1)
	return p.Push("opshub_test", nil, g)
}
