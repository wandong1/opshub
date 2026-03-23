package metrics

import (
	"fmt"
	"sort"

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

// MetricSample represents a single gauge metric sample with its label names/values.
type MetricSample struct {
	Name        string
	Help        string
	LabelNames  []string
	LabelValues []string
	Value       float64
}

// PushSamples pushes a list of MetricSample to Pushgateway.
// Samples with the same metric name are merged into one GaugeVec,
// supporting multiple label value combinations (e.g. per-step metrics).
// Grouping keys must not overlap with any label name in the samples.
func (p *Pusher) PushSamples(job string, grouping map[string]string, samples []MetricSample) error {
	type gaugeEntry struct {
		vec  *prometheus.GaugeVec
		help string
	}

	// Collect all label names per metric name (union across samples)
	metricLabelNames := make(map[string][]string)
	metricHelp := make(map[string]string)
	for _, s := range samples {
		existing := metricLabelNames[s.Name]
		if len(existing) == 0 {
			metricLabelNames[s.Name] = s.LabelNames
			metricHelp[s.Name] = s.Help
		}
	}

	reg := prometheus.NewRegistry()
	gauges := make(map[string]*prometheus.GaugeVec)

	for name, labelNames := range metricLabelNames {
		g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: metricHelp[name]}, labelNames)
		if err := reg.Register(g); err != nil {
			continue
		}
		gauges[name] = g
	}

	for _, s := range samples {
		g, ok := gauges[s.Name]
		if !ok {
			continue
		}
		g.WithLabelValues(s.LabelValues...).Set(s.Value)
	}

	pusher := push.New(p.url, job).Gatherer(reg)
	for k, v := range grouping {
		pusher = pusher.Grouping(k, v)
	}
	if p.username != "" {
		pusher = pusher.BasicAuth(p.username, p.password)
	}
	if err := pusher.Push(); err != nil {
		return fmt.Errorf("pushgateway push failed: %w", err)
	}
	return nil
}

// Push pushes the given collectors to Pushgateway under the specified job and grouping keys.
// Each call uses an isolated prometheus.Registry to avoid duplicate registration panics
// and grouping label conflicts.
//
// Deprecated: prefer PushSamples which correctly handles multiple label value combinations
// for the same metric name.
func (p *Pusher) Push(job string, grouping map[string]string, collectors ...prometheus.Collector) error {
	// Use a fresh isolated registry per push to avoid:
	// 1. "duplicate metrics collector registration" when same metric name appears multiple times
	// 2. "already contains grouping label" when a metric label matches a grouping key
	reg := prometheus.NewRegistry()
	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			// On duplicate registration, skip the duplicate silently
			continue
		}
	}

	pusher := push.New(p.url, job).Gatherer(reg)
	for k, v := range grouping {
		pusher = pusher.Grouping(k, v)
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

// labelKey returns a canonical string key for a label name/value set (for dedup).
func labelKey(names, values []string) string {
	pairs := make([]string, len(names))
	for i := range names {
		pairs[i] = names[i] + "=" + values[i]
	}
	sort.Strings(pairs)
	result := ""
	for _, p := range pairs {
		result += p + ","
	}
	return result
}
