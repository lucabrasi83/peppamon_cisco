package metrics

import (
	"sync"

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

func parseEigrpAdjMsg(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics) {

	for _, p := range msg.DataGpbkv {

		eigrpAdjObj := make(map[string]interface{})

		for _, e := range p.Fields[0].Fields {
			switch e.GetName() {
			case yangEigrpInstanceAfi:
				eigrpAdjObj["afi"] = e.GetStringValue()

			case yangEigrpInstanceVrf:
				eigrpAdjObj["vrf"] = e.GetStringValue()

				if eigrpAdjObj["vrf"] == "" {
					eigrpAdjObj["vrf"] = "Global"
				}
			case yangEigrpAdjInterface:
				eigrpAdjObj["interface"] = e.GetStringValue()

			case yangEigrPAdjNbrIP:
				eigrpAdjObj["neighbor_id"] = e.GetStringValue()

			}
		}

		// Instrument EIGRP Adjacency Status

		metricMutex := &sync.Mutex{}
		m := DeviceUnaryMetric{Mutex: metricMutex}

		m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg), prometheus.MustNewConstMetric(
			eigrpAdjStatus,
			prometheus.GaugeValue,
			1,
			msg.GetNodeIdStr(),
			eigrpAdjObj["neighbor_id"].(string),
			eigrpAdjObj["afi"].(string),
			eigrpAdjObj["vrf"].(string),
			eigrpAdjObj["interface"].(string),
		))

		dm.Mutex.Lock()
		dm.Metrics = append(dm.Metrics, m)
		dm.Mutex.Unlock()
	}
}
