package metrics

import (
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	eigrpAdjStatus = prometheus.NewDesc(
		"cisco_iosxe_eigrp_adjacency_status",
		"The current state of the EIGRP adjacency",
		[]string{"node", "neighbor_id", "address_family", "vrf", "interface"},
		nil,
	)
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-bgp-oper.yang
	EigrpAdjOperYANGEncodingPath = "Cisco-IOS-XE-eigrp-oper:eigrp-oper-data/eigrp-instance/eigrp-interface/eigrp-nbr"

	// EIGRP Instance AFI
	yangEigrpInstanceAfi = "afi"

	// EIGRP Instance VRF
	yangEigrpInstanceVrf = "vrf-name"

	// EIGRP Interface Adjacency
	yangEigrpAdjInterface = "name"

	// EIGRP Neighbor IP
	yangEigrPAdjNbrIP = "nbr-address"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     EigrpAdjOperYANGEncodingPath,
		RecordMetricFunc: parseEigrpAdjMsg,
	})
}

func parseEigrpAdjMsg(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	for _, p := range msg.DataGpbkv {

		eigrpAdjObj := make(map[string]interface{})

		for _, e := range p.Fields[0].Fields {
			switch e.GetName() {
			case yangEigrpInstanceAfi:
				eigrpAdjObj["afi"] = extractGPBKVNativeTypeFromOneof(e, false)

			case yangEigrpInstanceVrf:
				eigrpAdjObj["vrf"] = extractGPBKVNativeTypeFromOneof(e, false)

				if eigrpAdjObj["vrf"] == "" {
					eigrpAdjObj["vrf"] = "Global"
				}
			case yangEigrpAdjInterface:
				eigrpAdjObj["interface"] = extractGPBKVNativeTypeFromOneof(e, false)

			case yangEigrPAdjNbrIP:
				eigrpAdjObj["neighbor_id"] = extractGPBKVNativeTypeFromOneof(e, false)

			}
		}

		// Instrument EIGRP Adjacency Status
		CreatePromMetric(
			float64(1),
			eigrpAdjStatus,
			prometheus.GaugeValue,
			dm, t,
			node,
			eigrpAdjObj["neighbor_id"].(string),
			eigrpAdjObj["afi"].(string),
			eigrpAdjObj["vrf"].(string),
			eigrpAdjObj["interface"].(string),
		)
	}
}
