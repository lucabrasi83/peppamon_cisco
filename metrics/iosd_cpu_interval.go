package metrics

import (
	"sync"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cpu5Sec = prometheus.NewDesc(
		"cisco_iosxe_iosd_cpu_busy_5_sec_percentage",
		"The IOSd daemon CPU busy percentage over the last 5 seconds",
		[]string{"node"},
		nil, // constant labels
	)

	cpu1Min = prometheus.NewDesc(
		"cisco_iosxe_iosd_cpu_busy_1_min_percentage",
		"The IOSd daemon CPU busy percentage over the last minute",
		[]string{"node"},
		nil,
	)

	cpu5Min = prometheus.NewDesc(
		"cisco_iosxe_iosd_cpu_busy_5_min_percentage",
		"The IOSd daemon CPU busy percentage over the last 5 minutes",
		[]string{"node"},
		nil,
	)
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-process-cpu-oper.yang
	CPUYANGEncodingPath = "Cisco-IOS-XE-process-cpu-oper:cpu-usage/cpu-utilization"

	// Busy percentage in last 5-seconds
	yangCPUPBFiveSecFieldName = "five-seconds"

	// Busy percentage in last one minute
	yangCPUPBOneMinFieldName = "one-minute"

	// Busy percentage in last five minutes
	yangCPUPBFiveMinFieldName = "five-minutes"

	yangCPUUsageProcesses = "cpu-usage-process"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     CPUYANGEncodingPath,
		RecordMetricFunc: ParsePBMsgCPUBusyPercent,
	})
}

func ParsePBMsgCPUBusyPercent(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics) {

	var ProcCPUSlice []map[string]interface{}

	for _, cBusyInterval := range msg.DataGpbkv[0].Fields[1].Fields {

		switch cBusyInterval.GetName() {

		case yangCPUPBFiveSecFieldName:

			val := cBusyInterval.GetUint32Value()

			metricMutex := &sync.Mutex{}
			m := DeviceUnaryMetric{Mutex: metricMutex}

			m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
				prometheus.MustNewConstMetric(
					cpu5Sec,
					prometheus.GaugeValue,
					float64(val),
					msg.GetNodeIdStr(),
				))

			dm.Mutex.Lock()
			dm.Metrics = append(dm.Metrics, m)
			dm.Mutex.Unlock()

		case yangCPUPBOneMinFieldName:

			val := cBusyInterval.GetUint32Value()

			metricMutex := &sync.Mutex{}
			m := DeviceUnaryMetric{Mutex: metricMutex}

			m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg), prometheus.MustNewConstMetric(
				cpu1Min,
				prometheus.GaugeValue,
				float64(val),
				msg.GetNodeIdStr(),
			))

			dm.Mutex.Lock()
			dm.Metrics = append(dm.Metrics, m)
			dm.Mutex.Unlock()

		case yangCPUPBFiveMinFieldName:

			val := cBusyInterval.GetUint32Value()

			metricMutex := &sync.Mutex{}
			m := DeviceUnaryMetric{Mutex: metricMutex}

			m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg), prometheus.MustNewConstMetric(
				cpu5Min,
				prometheus.GaugeValue,
				float64(val),
				msg.GetNodeIdStr(),
			))
			dm.Mutex.Lock()
			dm.Metrics = append(dm.Metrics, m)
			dm.Mutex.Unlock()

		case yangCPUUsageProcesses:

			// Store the CPU Processes attributes in Struct
			procObj := parseCPUProcMeta(
				cBusyInterval.Fields,
				msg.GetNodeIdStr(),
			)

			// Add the CPU Process attributes slice to send for SQL Batch Job
			ProcCPUSlice = append(ProcCPUSlice, procObj)
		}

	}
	// Insert CPU Processes Usage Metadata in Batch SQL query
	go recordCPUProcMeta(ProcCPUSlice, msg.GetNodeIdStr())
}
