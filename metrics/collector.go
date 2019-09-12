package metrics

import (
	"sync"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var CiscoMetricRegistrar []CiscoTelemetryMetric

// Source represents the cache key for a metric
// Addr corresponds to the Telemetry client IP Socket and Path to the YANG schema path
type Source struct {
	Addr string
	Path string
}

type Collector struct {
	Mutex   *sync.Mutex
	Metrics map[Source]*DeviceGroupedMetrics
}

type DeviceUnaryMetric struct {
	Mutex  *sync.Mutex
	Metric prometheus.Metric
}

type DeviceGroupedMetrics struct {
	Mutex   *sync.Mutex
	Metrics []DeviceUnaryMetric
}

type CiscoTelemetryMetric struct {
	EncodingPath     string
	RecordMetricFunc func(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics)
}

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

	// IOSd CPU Metrics iosd_cpu_interval.go
	ch <- cpu5Sec
	ch <- cpu1Min
	ch <- cpu5Min

	// Interface Metrics interfaces_stats.go
	ch <- ifStatsInOctets
	ch <- ifStatsOutOctets
	ch <- ifStatsNumFlaps
	ch <- ifStatsCRCErrorsIn
	ch <- ifStatsOutDiscardPkts
	ch <- ifStatsInDiscardPkts
	ch <- ifStatsOutErrorPkts
	ch <- ifStatsInErrorPkts
	ch <- ifStatsOutBroadcastPkts
	ch <- ifStatsInBroadcastPkts
	ch <- ifStatsOutUnicastPkts
	ch <- ifStatsInUnicastPkts
	ch <- ifStatsOutMulticastPkts
	ch <- ifStatsInMulticastPkts

	// QoS Metrics qos_stats.go
	ch <- ifStatsQoSClassMapClassifiedBytes
	ch <- ifStatsQoSClassMapClassifiedPackets
	ch <- ifStatsQoSClassMapQueueOutputBytes
	ch <- ifStatsQoSClassMapQueueOutputPackets
	ch <- ifStatsQoSClassMapQueueSizeBytes
	ch <- ifStatsQoSClassMapQueueSizePackets
	ch <- ifStatsQoSClassMapQueueDropBytes
	ch <- ifStatsQoSClassMapQueueDropPackets

	// IOSd Memory metrics iosd_memory_utilization.go
	ch <- iosdTotalMemory
	ch <- iosdUsedMemory
	ch <- iosdFreeMemory

	// BGP Metrics bgp_metrics.go
	ch <- bgpIpv4NeighborPrefixesRcvd
	ch <- bgpGlobalMeta

	// EIGRP Adjancecy Status eigrp_adjacency.go
	ch <- eigrpAdjStatus

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

	return promTimestamp
}
