package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ifStatsInOctets = prometheus.NewDesc(
		"cisco_iosxe_if_stats_in_octets",
		"The number of inbound octets processed by the interface",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsOutOctets = prometheus.NewDesc(
		"cisco_iosxe_if_stats_out_octets",
		"The number of outbound octets processed by the interface",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsNumFlaps = prometheus.NewDesc(
		"cisco_iosxe_if_stats_num_flaps",
		"The number of times the interface state transitioned between up and down",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsCRCErrorsIn = prometheus.NewDesc(
		"cisco_iosxe_if_stats_num_crc_errors",
		"Number of receive error events due to FCS/CRC check failure",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsOutDiscardPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_out_discard_packets",
		"Number of outbound packets discarded",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsInDiscardPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_in_discard_packets",
		"Number of inbound packets discarded",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsOutErrorPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_out_error_packets",
		"Number of outbound packets that container errors",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsInErrorPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_in_error_packets",
		"Number of inbound packets that container errors",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsOutBroadcastPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_out_broadcast_packets",
		"Number of outbound broadcast packets processed",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsInBroadcastPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_in_broadcast_packets",
		"Number of inbound broadcast packets processed",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsOutUnicastPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_out_unicast_packets",
		"Number of outbound unicast packets processed",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsInUnicastPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_in_unicast_packets",
		"Number of inbound unicast packets processed",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsOutMulticastPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_out_multicast_packets",
		"Number of outbound multicast packets processed",
		[]string{"node", "interface"},
		nil,
	)

	ifStatsInMulticastPkts = prometheus.NewDesc(
		"cisco_iosxe_if_stats_in_multicast_packets",
		"Number of inbound multicast packets processed",
		[]string{"node", "interface"},
		nil,
	)
)

// Define YANG Schema Node we're interested in instrumenting
const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-interfaces-oper.yang
	IfStatsYANGEncodingPath = "Cisco-IOS-XE-interfaces-oper:interfaces/interface"

	// The total number of octets in IP packets for the specified address family that the device
	//  supplied to the lower layers for transmission.  This includes packets generated locally and those forwarded by
	//  the device."
	yangIfStatsOctetsOut = "out-octets"

	//  The total number of octets received on the interface, including framing characters.
	//  Discontinuities in the value of this counter can occur at re-initialization of the management system,
	//  and at other times as indicated by the value of discontinuity-time
	yangIfStatsOctetsIn = "in-octets"

	// The number of times the interface state transitioned between up and down
	yangIfNumFlaps = "num-flaps"

	// Number of receive error events due to FCS/CRC check failure
	yangIfCRCErrorsIn = "in-crc-errors"

	// The number of outbound packets that were chosen to be discarded even though no errors had been detected
	// to prevent their being transmitted.  One possible reason for discarding such a packet
	// could be to free up buffer space
	yangIfStatsOutDiscards = "out-discards"

	// The number of output IP packets for the specified address family, for which no problems were
	// encountered to prevent their continued processing,
	// but were discarded (e.g., for lack of buffer space).
	yangIfStatsOutv4Discards = "out-discarded-pkts"

	// The number of inbound packets that were chosen to be discarded even though no errors had been detected
	// to prevent their being deliverable to a higher-layer protocol.
	// One possible reason for discarding such a packet could be to free up buffer space
	yangIfStatsInDiscards = "in-discards"

	// The number of input IP packets for the specified address family, for which no problems were
	// encountered to prevent their continued processing,
	// but were discarded (e.g., for lack of buffer space).
	yangIfStatsInv4Discards = "in-discarded-pkts"

	// The number of inbound packets that contained errors preventing them from being
	// deliverable to a higher-layer protocol.
	yangIfStatsInErrors = "in-errors"

	// Number of packets discarded due to errors for the specified address family, including errors in the
	// header, no route found to the destination, invalid address, unknown protocol, etc.
	yangIfStatsInv4Errors = "in-error-pkts"

	// The number of outbound packets that could not be transmitted because of errors.
	yangIfStatsOutErrors = "out-errors"

	// Number of IP packets for the specified address family locally generated
	// and discarded due to errors, including no route found to the IP destination.
	yangIfStatsOutv4Errors = "out-error-pkts"

	// The total number of packets that higher-level protocols requested be transmitted,
	// and that were addressed to a broadcast address at this sub-layer, including those
	// that were discarded or not sent.
	yangIfStatsOutBroadcastPkts = "out-broadcast-pkts"

	// The number of packets, delivered by this sub-layer to a higher (sub-)layer,
	// that were addressed to a broadcast address at this sub-layer.
	yangIfStatsInBroadcastPkts = "in-broadcast-pkts"

	// The number of packets, delivered by this sub-layer to a
	// higher (sub-)layer, that were not addressed to a
	// multicast or broadcast address at this sub-layer
	yangIfStatsInUnicastPkts = "in-unicast-pkts"

	// The total number of packets that higher-level protocols
	// requested be transmitted, and that were not addressed
	// to a multicast or broadcast address at this sub-layer,
	// including those that were discarded or not sent
	yangIfStatsOutUnicastPkts = "out-unicast-pkts"

	// The total number of packets that higher-level protocols
	// requested be transmitted, and that were addressed to a
	// multicast address at this sub-layer, including those
	// that were discarded or not sent.  For a MAC-layer
	// protocol, this includes both Group and Functional addresses
	yangIfStatsOutMulticastPkts = "out-multicast-pkts"

	// The number of packets, delivered by this sub-layer to a
	// higher (sub-)layer, that were addressed to a multicast
	// address at this sub-layer.  For a MAC-layer protocol,
	// this includes both Group and Functional addresses
	yangIfStatsInMulticastPkts = "in-multicast-pkts"

	// VRF to which this interface belongs to. If the interface is not in a VRF then it is 'Global'
	IfVRF = "vrf"

	// IPv4 address configured on interface
	IfIPv4Address = "ipv4"

	// IPv4 Subnet Mask
	IfIPv4Mask = "ipv4-subnet-mask"

	// Interface description
	IfDescription = "description"

	// Maximum transmission unit
	IfMTU = "mtu"

	// An estimate of the interface's current bandwidth in bits per second.  For interfaces that do not vary in
	// bandwidth or for those where no accurate estimation can be made,
	// this node should contain the nominal bandwidth.
	// For interfaces that have no concept of bandwidth, this node is not present
	IfSpeed = "speed"

	// The desired state of the interface.
	// This leaf has the same read semantics as ifAdminStatus
	IfAdminStatus = "admin-status"

	// The current operational state of the interface.
	// This leaf has the same semantics as ifOperStatus
	IfOperStatus = "oper-status"

	// The time the interface entered its current operational state. If the current state was entered prior to the last re-initialization of the local network management subsystem, then this node is not present
	IfLastChange = "last-change"

	// The interface's address at its protocol sub-layer.  For  example, for an 802.x interface, this object normally
	// contains a Media Access Control (MAC) address.  The interface's media-specific modules must define the bit
	// and byte ordering and the format of the value of this object.  For interfaces that do not have such an address
	// (e.g., a serial line), this node is not present
	IfPhysicalAddress = "phys-address"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     IfStatsYANGEncodingPath,
		RecordMetricFunc: ParsePBMsgInterfaceStats,
	})
}

func ParsePBMsgInterfaceStats(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics) {

	//Store Interfaces Metadata in slice
	ifMetaSlice := make([]map[string]interface{}, 0, len(msg.DataGpbkv))

	// Loop through the interface name keys
	for _, i := range msg.DataGpbkv {

		interfaceName := i.Fields[0].Fields[0].GetStringValue()

		timestamps := time.Now().Unix()

		// Extract CPE Hostname
		nodeName := msg.GetNodeIdStr()

		// Instrument interface metadata
		ifMeta := recordInterfaceMeta(i.Fields[1].Fields, interfaceName, nodeName, timestamps)

		// Loop through the statistics leafs
		for _, m := range i.Fields[1].Fields {

			// If interface is Ethernet sub-interface, need to loop through v4-protocol-stats to get the rate statistics
			if m.GetName() == "v4-protocol-stats" && strings.Contains(interfaceName, ".") {

				for _, subIfStats := range m.Fields {
					switch subIfStats.GetName() {
					case yangIfStatsOctetsOut:

						val := subIfStats.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutOctets,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOctetsIn:

						val := subIfStats.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInOctets,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsInv4Errors:

						val := subIfStats.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInErrorPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOutv4Errors:

						val := subIfStats.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutErrorPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsInv4Discards:

						val := subIfStats.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInDiscardPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOutv4Discards:

						val := subIfStats.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutDiscardPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()
					}
				}
				break
			}

			// Metrics for physical interfaces
			if m.GetName() == "statistics" && !strings.Contains(interfaceName, ".") {

				// Loop through individual interface statistics leafs
				for _, sts := range m.Fields {

					switch sts.GetName() {

					case yangIfStatsOctetsOut:

						val := sts.GetUint32Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutOctets,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOctetsIn:

						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInOctets,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfNumFlaps:

						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsNumFlaps,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfCRCErrorsIn:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsCRCErrorsIn,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOutDiscards:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutDiscardPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsInDiscards:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInDiscardPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsInErrors:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInErrorPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOutErrors:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutErrorPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOutBroadcastPkts:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutBroadcastPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsInBroadcastPkts:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInBroadcastPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOutUnicastPkts:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutUnicastPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsInUnicastPkts:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInUnicastPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsOutMulticastPkts:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsOutMulticastPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					case yangIfStatsInMulticastPkts:
						val := sts.GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
							prometheus.MustNewConstMetric(
								ifStatsInMulticastPkts,
								prometheus.CounterValue,
								float64(val),
								nodeName,
								interfaceName,
							))

						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()

					}
				}

			}
			if m.GetName() == "diffserv-info" {

				go instrumentQoSStats(m.Fields, interfaceName, nodeName, dm, convTelemetryTimestampToTime(msg))

			}

		}
		ifMetaSlice = append(ifMetaSlice, ifMeta)
	}

	// Persist Interface Metadata in Meta DB within a separate goroutine
	go func() {

		err := metadb.DBInstance.PersistsInterfaceMetadata(ifMetaSlice, msg.GetNodeIdStr())

		if err != nil {
			logging.PeppaMonLog(
				"error",
				fmt.Sprintf("failed to insert interface metadata in DB for node %v, error: %v", msg.GetNodeIdStr(),
					err))
		}
	}()
}

func recordInterfaceMeta(fields []*telemetry.TelemetryField, ifName string, node string,
	t int64) map[string]interface{} {

	ifMetaMap := make(map[string]interface{})

	ifMetaMap["node_id"] = node
	ifMetaMap["timestamps"] = t
	ifMetaMap["if_name"] = ifName

	for _, f := range fields {
		switch f.GetName() {
		case IfVRF:

			if f.GetStringValue() == "" {
				ifMetaMap["vrf"] = "Global"
			} else {
				ifMetaMap["vrf"] = f.GetStringValue()
			}

		case IfDescription:

			if f.GetStringValue() == "" {
				ifMetaMap["description"] = "No description"
			} else {

				ifMetaMap["description"] = f.GetStringValue()
			}

		case IfIPv4Address:

			ifMetaMap["ipv4_address"] = f.GetStringValue()

		case IfSpeed:
			ifMetaMap["speed"] = f.GetUint64Value()

		case IfIPv4Mask:

			ifMetaMap["ipv4_subnet_mask"] = f.GetStringValue()

		case IfPhysicalAddress:
			ifMetaMap["physical_address"] = f.GetStringValue()

		case IfAdminStatus:
			ifMetaMap["admin_status"] = mapIfStatusToInteger(f.GetStringValue())

		case IfOperStatus:
			ifMetaMap["oper_status"] = mapIfStatusToInteger(f.GetStringValue())

		case IfMTU:
			ifMetaMap["mtu"] = f.GetUint32Value()

		case IfLastChange:
			ifMetaMap["last_change"] = f.GetStringValue()
		}

		// Avoid Panic when converting IPv4 string to net.IPNet object
		if _, ok := ifMetaMap["ipv4_address"]; !ok {
			ifMetaMap["ipv4_address"] = "0.0.0.0"
		}

		if _, ok := ifMetaMap["ipv4_subnet_mask"]; !ok {
			ifMetaMap["ipv4_subnet_mask"] = "0.0.0.0"
		}
	}

	return ifMetaMap

}

// mapIfStatusToInteger is a helper function to map the interface status to an integer for Grafana dashboards
func mapIfStatusToInteger(status string) string {

	// Interface Status (Admin/Oper) mapped to Integer for Grafana cell coloring
	// Workaround until Grafana allows mapping of colors to string values
	ifStatusMap := map[string]string{
		"if-oper-state-ready":            "100",
		"if-oper-state-lower-layer-down": "0",
		"if-oper-state-invalid":          "0",
		"if-oper-state-no-pass":          "0",
		"if-oper-state-unknown":          "0",
		"if-oper-state-not-present":      "0",
		"if-state-up":                    "100",
		"if-state-down":                  "0",
		"if-state-unknown":               "0",
		"if-state-test":                  "0",
	}

	return ifStatusMap[status]
}
