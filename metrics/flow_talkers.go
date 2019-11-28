package metrics

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	flowTalkerStatsBytes = prometheus.NewDesc(
		"cisco_iosxe_flexible_netflow_record_bytes",
		"The number of bytes passed through the netflow record",
		[]string{
			"node",
			"source_address",
			"destination_address",
			"interface_input",
			"is_multicast",
			"vrf_id_input",
			"source_port",
			"destination_port",
			"dscp",
			"ip_protocol",
			"interface_output",
		},
		nil,
	)

	flowTalkerStatsPackets = prometheus.NewDesc(
		"cisco_iosxe_flexible_netflow_record_packets",
		"The number of packets passed through the netflow record",
		[]string{
			"node",
			"source_address",
			"destination_address",
			"interface_input",
			"is_multicast",
			"vrf_id_input",
			"source_port",
			"destination_port",
			"dscp",
			"ip_protocol",
			"interface_output",
		},
		nil,
	)
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-flow-monitor-oper.yang
	FlowMonitorTalkersYANGEncodingPath = "Cisco-IOS-XE-flow-monitor-oper:flow-monitors/flow-monitor"

	// Flow record
	yangFlowRecordKey = "flow"

	// Flow record Source IP Address
	yangFlowRecordSourceAddress = "source-address"

	// Flow record Destination IP Address
	yangFlowRecordDestinationAddress = "destination-address"

	// Flow record Destination IP Address
	yangFlowRecordInterfaceInput = "interface-input"

	// Flow record Is Mutlicast
	yangFlowRecordIsMulticast = "is-multicast"

	// Flow record VRF ID input
	yangFlowRecordVRFIDInput = "vrf-id-input"

	// Flow record Source Port
	yangFlowRecordSourcePort = "source-port"

	// Flow record Destination Port
	yangFlowRecordDestinationPort = "destination-port"

	// Flow record Destination Port
	yangFlowRecordIPTOS = "ip-tos"

	// Flow record Destination Port
	yangFlowRecordIPProtocol = "ip-protocol"

	// Flow record Output Interface
	yangFlowRecordInterfaceOutput = "interface-output"

	// Flow record processed bytes
	yangFlowRecordProcessBytes = "bytes"

	// Flow record processed Packets
	yangFlowRecordProcessPackets = "packets"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     FlowMonitorTalkersYANGEncodingPath,
		RecordMetricFunc: parseFlowMonitorTalkersMsg,
	})
}

func parseFlowMonitorTalkersMsg(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	if len(msg.DataGpbkv) == 0 {
		return
	}
	for _, p := range msg.DataGpbkv[0].Fields {

		// Launch recursive function to parse Telemetry PB message and capture desired info
		parseFlowMonitors(p, dm, t, node)
	}

}

func parseFlowMonitors(m *telemetry.TelemetryField, dm *DeviceGroupedMetrics, t time.Time, node string) {
	for _, field := range m.Fields {
		if field.GetName() == yangFlowRecordKey {

			flowObj := map[string]interface{}{
				"node":                           node,
				"timestamps":                     t.Unix(),
				yangFlowRecordSourceAddress:      "N/A",
				yangFlowRecordDestinationAddress: "N/A",
				yangFlowRecordInterfaceInput:     "N/A",
				yangFlowRecordIsMulticast:        "N/A",
				yangFlowRecordVRFIDInput:         "N/A",
				yangFlowRecordSourcePort:         "N/A",
				yangFlowRecordDestinationPort:    "N/A",
				yangFlowRecordIPTOS:              "N/A",
				yangFlowRecordIPProtocol:         "N/A",
				yangFlowRecordInterfaceOutput:    "N/A",
				yangFlowRecordProcessBytes:       0,
				yangFlowRecordProcessPackets:     0,
			}

			for _, flowField := range field.Fields {
				if _, ok := flowObj[flowField.GetName()]; ok {

					flowObj[flowField.GetName()] = extractGPBKVNativeTypeFromOneof(flowField, false)

				}

			}

			// Decode TOS Hex to Int for DSCP conversion
			tosStripX := strings.Replace(flowObj[yangFlowRecordIPTOS].(string), "0x", "", -1)
			tosToInt, _ := strconv.ParseInt(tosStripX, 16, 64)

			// Create Metric for Bytes processed
			CreatePromMetric(
				flowObj[yangFlowRecordProcessBytes].(float64),
				flowTalkerStatsBytes,
				prometheus.CounterValue,
				dm, t,
				node,
				flowObj[yangFlowRecordSourceAddress].(string),
				flowObj[yangFlowRecordDestinationAddress].(string),
				flowObj[yangFlowRecordInterfaceInput].(string),
				flowObj[yangFlowRecordIsMulticast].(string),
				fmt.Sprintf("%.0f", flowObj[yangFlowRecordVRFIDInput].(float64)),
				fmt.Sprintf("%.0f", flowObj[yangFlowRecordSourcePort].(float64)),
				fmt.Sprintf("%.0f", flowObj[yangFlowRecordDestinationPort].(float64)),
				convTOStoDSCP(int(tosToInt)),
				convIPProtocolToName(flowObj[yangFlowRecordIPProtocol].(float64)),
				flowObj[yangFlowRecordInterfaceOutput].(string),
			)

			// Create Metric for Packets processed
			CreatePromMetric(
				flowObj[yangFlowRecordProcessPackets].(float64),
				flowTalkerStatsPackets,
				prometheus.CounterValue,
				dm, t,
				node,
				flowObj[yangFlowRecordSourceAddress].(string),
				flowObj[yangFlowRecordDestinationAddress].(string),
				flowObj[yangFlowRecordInterfaceInput].(string),
				flowObj[yangFlowRecordIsMulticast].(string),
				fmt.Sprintf("%.0f", flowObj[yangFlowRecordVRFIDInput].(float64)),
				fmt.Sprintf("%.0f", flowObj[yangFlowRecordSourcePort].(float64)),
				fmt.Sprintf("%.0f", flowObj[yangFlowRecordDestinationPort].(float64)),
				convTOStoDSCP(int(tosToInt)),
				convIPProtocolToName(flowObj[yangFlowRecordIPProtocol].(float64)),
				flowObj[yangFlowRecordInterfaceOutput].(string),
			)

		}

	}

}
