package metrics

import (
	"strconv"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ospfAdjStatus = prometheus.NewDesc(
		"cisco_iosxe_ospf_adjacency_status",
		"The current state of the OSPF adjacency",
		[]string{"node", "neighbor_id", "neighbor_ip", "ospf_instance_id", "interface", "area_id"},
		nil,
	)
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-ospf-oper.yang
	OspfAdjOperYANGEncodingPath = "Cisco-IOS-XE-ospf-oper:ospf-oper-data/ospfv2-instance/ospfv2-area/ospfv2-interface/ospfv2-neighbor"

	// OSPF Instance ID
	yangOspfInstanceID = "instance-id"

	// OSPF Area ID
	yangOspfAreaID = "area-id"

	// OSPF Adjacency Interface
	yangOspfAdjInterface = "name"

	// OSPF Neighbor ID
	yangOspfAdjNeighborID = "nbr-id"

	// OSPF Neighbor Address
	yangOspfAdjNeighborAddress = "address"

	yangOspfAdjNeighborState = "state"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     OspfAdjOperYANGEncodingPath,
		RecordMetricFunc: parseOSPFAdjMsg,
	})
}

func parseOSPFAdjMsg(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	for _, p := range msg.DataGpbkv {

		// Loop through OSPF instance data
		ospfAdjObj := make(map[string]interface{})

		for _, ospfInstance := range p.Fields[0].Fields {
			switch ospfInstance.GetName() {
			case yangOspfInstanceID:
				val := extractGPBKVNativeTypeFromOneof(ospfInstance, true)
				ospfAdjObj["instance_id"] = strconv.Itoa(int(val.(float64)))

			case yangOspfAreaID:
				val := extractGPBKVNativeTypeFromOneof(ospfInstance, true)
				ospfAdjObj["area_id"] = strconv.Itoa(int(val.(float64)))
			case yangOspfAdjInterface:
				val := extractGPBKVNativeTypeFromOneof(ospfInstance, false)
				ospfAdjObj["interface"] = val.(string)
			case yangOspfAdjNeighborID:
				val := extractGPBKVNativeTypeFromOneof(ospfInstance, true)
				ipv4 := intToIP4(int64(val.(float64)))
				ospfAdjObj["neighbor_id"] = ipv4

			}
		}
		for _, ospfNbrStatus := range p.Fields[1].Fields {
			switch ospfNbrStatus.GetName() {

			case yangOspfAdjNeighborAddress:
				ospfAdjObj["neighbor_ip"] = extractGPBKVNativeTypeFromOneof(ospfNbrStatus, false)
			case yangOspfAdjNeighborState:
				val := extractGPBKVNativeTypeFromOneof(ospfNbrStatus, false)
				ospfAdjObj["neighbor_status"] = ospfNbrStatusToNum(val.(string))

			}
		}

		// Instrument OSPF Adjacency Status
		CreatePromMetric(
			ospfAdjObj["neighbor_status"],
			ospfAdjStatus,
			prometheus.GaugeValue,
			dm, t,
			node,
			ospfAdjObj["neighbor_id"].(string),
			ospfAdjObj["neighbor_ip"].(string),
			ospfAdjObj["instance_id"].(string),
			ospfAdjObj["interface"].(string),
			ospfAdjObj["area_id"].(string),
		)

	}

}
