package metrics

import (
	"sync"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var CiscoMetricRegistrar []CiscoTelemetryMetric

// Source represents the cache key for a metric
// Addr corresponds to the Telemetry NodeID and Path to the YANG schema path
type Source struct {
	NodeID string
	Path   string
}

// Collector represents the Peppamon Telemetry collector that will carry all the metrics collected
type Collector struct {
	Mutex   *sync.Mutex
	Metrics map[Source]*DeviceGroupedMetrics
}

// DeviceUnaryMetric represents a single Device Metric
type DeviceUnaryMetric struct {
	Mutex  *sync.Mutex
	Metric prometheus.Metric
}

// DeviceGroupedMetrics represents a set of grouped metrics
type DeviceGroupedMetrics struct {
	Mutex   *sync.Mutex
	Metrics []DeviceUnaryMetric
}

// CiscoTelemetryMetric represents a Cisco IOS-XE telemetry metric sent in protocol buffer format
type CiscoTelemetryMetric struct {
	EncodingPath     string
	RecordMetricFunc func(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics)
}

// NewCollector will create a new instance of a Peppamon Collector
func NewCollector() *Collector {

	mu := &sync.Mutex{}

	return &Collector{
		Mutex:   mu,
		Metrics: make(map[Source]*DeviceGroupedMetrics),
	}
}

// Describe method will write metrics descriptors within Prometheus Desc channel
// and implements prometheus.Collector interface
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {

	// Copy current descriptors in case consumer channel is slow
	var metricDescriptors []*prometheus.Desc

	c.Mutex.Lock()
	for _, source := range c.Metrics {
		for _, metric := range source.Metrics {
			metricDescriptors = append(metricDescriptors, metric.Metric.Desc())
		}
	}
	c.Mutex.Unlock()

	for _, desc := range metricDescriptors {
		ch <- desc
	}
}

// Collect method implements prometheus.Collector interface and is executed upon each scrape
// Telemetry Metric cache is sent to Prometheus channel consumer.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {

	// Copy current metrics so we don't lock for very long if channel consumer is slow.
	var metrics []prometheus.Metric
	c.Mutex.Lock()
	for _, deviceMetrics := range c.Metrics {
		for _, metric := range deviceMetrics.Metrics {
			metric.Mutex.Lock()
			metrics = append(metrics, metric.Metric)
			metric.Mutex.Unlock()
		}

	}
	c.Mutex.Unlock()

	for _, metric := range metrics {
		ch <- metric
	}

}

// convTelemetryTimestampToTime is a helper function that processes Timestamps from Telemetry messages and convert
// as a Time type
func convTelemetryTimestampToTime(msg *telemetry.Telemetry) time.Time {
	// Extract Timestamp from Telemetry message
	msgTimestamps := msg.GetMsgTimestamp()
	promTimestamp := time.Unix(int64(msgTimestamps)/1000, 0)

	return promTimestamp.UTC()
}
