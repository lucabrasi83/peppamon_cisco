package metrics

import (
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	iosdTotalMemory = prometheus.NewDesc(
		"cisco_iosxe_iosd_total_memory_bytes",
		"The IOSd daemon total memory",
		[]string{"node"},
		nil,
	)

	iosdUsedMemory = prometheus.NewDesc(
		"cisco_iosxe_iosd_used_memory_bytes",
		"The IOSd daemon used memory",
		[]string{"node"},
		nil,
	)

	iosdFreeMemory = prometheus.NewDesc(
		"cisco_iosxe_iosd_free_memory_bytes",
		"The IOSd daemon free memory",
		[]string{"node"},
		nil,
	)
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-memory-oper.yang
	iosDMemoryYANGEncodingPath = "Cisco-IOS-XE-memory-oper:memory-statistics/memory-statistic"

	// Total memory in the pool (bytes)
	yangIOSdTotalMemory = "total-memory"

	// Total used memory in the pool (bytes)
	yangIOSdUsedMemory = "used-memory"

	// Total free memory in the pool (bytes)
	yangIOSdFreeMemory = "free-memory"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     iosDMemoryYANGEncodingPath,
		RecordMetricFunc: ParsePBMsgIOSdMemoryUsage,
	})
}

func ParsePBMsgIOSdMemoryUsage(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	// Look specifically for Processor memory pool
	for _, field := range msg.DataGpbkv[0].Fields[1].Fields {
		switch field.GetName() {
		case yangIOSdTotalMemory:
			val := extractGPBKVNativeTypeFromOneof(field, true)

			CreatePromMetric(
				val,
				iosdTotalMemory,
				prometheus.GaugeValue,
				dm, t,
				node,
			)

		case yangIOSdUsedMemory:
			val := extractGPBKVNativeTypeFromOneof(field, true)

			CreatePromMetric(
				val,
				iosdUsedMemory,
				prometheus.GaugeValue,
				dm, t,
				node,
			)

		case yangIOSdFreeMemory:
			val := extractGPBKVNativeTypeFromOneof(field, true)

			CreatePromMetric(
				val,
				iosdFreeMemory,
				prometheus.GaugeValue,
				dm, t,
				node,
			)
		}
	}
}
