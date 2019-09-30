package metrics

import (
	"strings"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

// Declare Metrics Descriptors
var (
	ifStatsQoSClassMapClassifiedBytes = prometheus.NewDesc(
		"cisco_iosxe_qos_class_map_classfied_bytes",
		"The number of total bytes which filtered to the classifier-entry",
		[]string{"node", "interface", "direction", "policy_map", "class_map", "parent_path"},
		nil,
	)

	ifStatsQoSClassMapQueueOutputBytes = prometheus.NewDesc(
		"cisco_iosxe_qos_class_map_queued_bytes",
		"The number of bytes transmitted from queue",
		[]string{"node", "interface", "direction", "policy_map", "class_map", "parent_path"},
		nil,
	)

	ifStatsQoSClassMapQueueSizeBytes = prometheus.NewDesc(
		"cisco_iosxe_qos_class_map_queue_size_bytes",
		"The number of bytes currently buffered",
		[]string{"node", "interface", "direction", "policy_map", "class_map", "parent_path"},
		nil,
	)

	ifStatsQoSClassMapQueueDropBytes = prometheus.NewDesc(
		"cisco_iosxe_qos_class_map_queue_drops_bytes",
		"The total number of bytes dropped",
		[]string{"node", "interface", "direction", "policy_map", "class_map", "parent_path"},
		nil,
	)

	//ifStatsQoSClassMapMarkedPkts = promauto.NewGaugeVec(
	//
	//	prometheus.GaugeOpts{
	//		Name: "cisco_iosxe_qos_class_map_marked_packets",
	//		Help: "The total number of bytes dropped",
	//	},
	//	[]string{
	//		"node",
	//		"interface",
	//		"direction",
	//		"policy_map",
	//		"class_map",
	//		"parent_path",
	//		"marking_scheme",
	//		"marking_value",
	//	},
	//)
)

const (
	// Direction fo the traffic flow either inbound or outbound
	yangQoSDirection = "direction"

	// Policy entry name
	yangQoSPolicyName = "policy-name"

	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-interfaces-oper.yang#L2253
	yangQoSDiffservClassifierEntries = "diffserv-target-classifier-stats"

	// Classifier Entry Name
	yangQoSClassMapName = "classifier-entry-name"

	// Path of the Classifier Entry in a hierarchical policy
	yangQoSClassMapParentPath = "parent-path"

	// Diffserv classifier statistics
	yangQoSClassifierEntryStats = "classifier-entry-stats"

	// Number of total packets which filtered to the classifier-entry
	yangQoSClassifiedPackets = "classified-pkts"

	// Number of total bytes which filtered to the classifier-entry
	yangQoSClassifiedBytes = "classified-bytes"

	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-interfaces-oper.yang#L2074
	// Queuing Counters
	yangQoSClassifierQueueStats = "queuing-stats"

	// Number of packets transmitted from queue
	yangQoSClassifierQueueOutputPackets = "output-pkts"

	// Number of bytes transmitted from queue
	yangQoSClassifierQueueOutputBytes = "output-bytes"

	// Number of packets currently buffered
	yangQoSClassifierQueueBufferedPackets = "queue-size-pkts"

	// Number of bytes currently buffered
	yangQoSClassifierQueueBufferedBytes = "queue-size-bytes"

	// Total number of packets dropped
	yangQoSClassifierQueueDroppedPackets = "drop-pkts"

	// Total number of bytes dropped
	yangQoSClassifierQueueDroppedBytes = "drop-bytes"
)

func instrumentQoSStats(fields []*telemetry.TelemetryField, ifName string, node string, dm *DeviceGroupedMetrics,
	t time.Time) {

	var direction string
	var policyName string

	for _, f := range fields {
		switch f.GetName() {

		case yangQoSDirection:
			direction = extractGPBKVNativeTypeFromOneof(f, false).(string)

		case yangQoSPolicyName:
			policyName = extractGPBKVNativeTypeFromOneof(f, false).(string)

		case yangQoSDiffservClassifierEntries:

			var classMapName string
			var parentPath string

			for _, classMeta := range f.Fields {

				switch classMeta.GetName() {
				case yangQoSClassMapName:
					classMapName = extractGPBKVNativeTypeFromOneof(classMeta, false).(string)

				case yangQoSClassMapParentPath:
					parentPathRawString := extractGPBKVNativeTypeFromOneof(classMeta, false).(string)
					parentPathSplit := strings.Split(parentPathRawString, " ")
					parentPath = parentPathSplit[len(parentPathSplit)-2]

				// Handle Packets/Bytes classified within the class map
				case yangQoSClassifierEntryStats:

					for _, classifierStat := range classMeta.Fields {
						switch classifierStat.GetName() {

						case yangQoSClassifiedBytes:
							val := extractGPBKVNativeTypeFromOneof(classifierStat, true)

							CreatePromMetric(
								val,
								ifStatsQoSClassMapClassifiedBytes,
								prometheus.CounterValue,
								dm, t,
								node,
								ifName,
								direction,
								policyName,
								classMapName,
								parentPath,
							)

						}
					}

				// Handle Packets/Bytes Queue statistics within the class map
				case yangQoSClassifierQueueStats:

					for _, queueStat := range classMeta.Fields {
						switch queueStat.GetName() {

						case yangQoSClassifierQueueOutputBytes:

							val := extractGPBKVNativeTypeFromOneof(queueStat, true)

							CreatePromMetric(
								val,
								ifStatsQoSClassMapQueueOutputBytes,
								prometheus.CounterValue,
								dm, t,
								node,
								ifName,
								direction,
								policyName,
								classMapName,
								parentPath,
							)

						case yangQoSClassifierQueueBufferedBytes:

							val := extractGPBKVNativeTypeFromOneof(queueStat, true)

							CreatePromMetric(
								val,
								ifStatsQoSClassMapQueueSizeBytes,
								prometheus.CounterValue,
								dm, t,
								node,
								ifName,
								direction,
								policyName,
								classMapName,
								parentPath,
							)

						case yangQoSClassifierQueueDroppedBytes:

							val := extractGPBKVNativeTypeFromOneof(queueStat, true)

							CreatePromMetric(
								val,
								ifStatsQoSClassMapQueueDropBytes,
								prometheus.CounterValue,
								dm, t,
								node,
								ifName,
								direction,
								policyName,
								classMapName,
								parentPath,
							)

						}
					}
				}
			}

		}
	}

}
