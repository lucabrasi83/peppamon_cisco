package metrics

import (
	"sync"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
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
	RecordMetricFunc func(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string)
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

// CreatePromMetric will create on the fly the Prometheus metric
func CreatePromMetric(
	val interface{},
	desc *prometheus.Desc,
	mt prometheus.ValueType,
	dm *DeviceGroupedMetrics,
	t time.Time,
	labels ...string) {

	if v, ok := val.(float64); !ok {
		logging.PeppaMonLog("error",
			"Metric %v value %v not float64. Skipping it.", *desc, v)
		return
	}

	metricMutex := &sync.Mutex{}
	m := DeviceUnaryMetric{Mutex: metricMutex}
	m.Metric = prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(
		desc,
		mt,
		val.(float64),
		labels...,
	))
	dm.Mutex.Lock()
	dm.Metrics = append(dm.Metrics, m)
	dm.Mutex.Unlock()
}

// Support function to extract the value field from K/V Field.
func extractGPBKVNativeTypeFromOneof(field *telemetry.TelemetryField, num bool) interface{} {

	switch field.ValueByType.(type) {
	case *telemetry.TelemetryField_BytesValue:
		if !num {
			return field.GetBytesValue()
		}
	case *telemetry.TelemetryField_StringValue:
		if !num {
			return field.GetStringValue()
		}
	case *telemetry.TelemetryField_BoolValue:
		if !num {
			return field.GetBoolValue()
		}
	case *telemetry.TelemetryField_Uint32Value:
		return float64(field.GetUint32Value())
	case *telemetry.TelemetryField_Uint64Value:
		return float64(field.GetUint64Value())
	case *telemetry.TelemetryField_Sint32Value:
		return float64(field.GetSint32Value())
	case *telemetry.TelemetryField_Sint64Value:
		return float64(field.GetSint64Value())
	case *telemetry.TelemetryField_DoubleValue:
		return field.GetDoubleValue()
	case *telemetry.TelemetryField_FloatValue:
		return float64(field.GetFloatValue())
	}

	return nil
}
