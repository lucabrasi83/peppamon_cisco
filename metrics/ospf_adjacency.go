package metrics

import (
	"strconv"
	"strings"
	"sync"

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

func parseOSPFAdjMsg(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics) {

	for _, p := range msg.DataGpbkv {

		// Loop through OSPF instance data
		ospfAdjObj := make(map[string]interface{})

		for _, ospfInstance := range p.Fields[0].Fields {
			switch ospfInstance.GetName() {
			case yangOspfInstanceID:
				ospfAdjObj["instance_id"] = strconv.Itoa(int(ospfInstance.GetUint32Value()))
			case yangOspfAreaID:
				ospfAdjObj["area_id"] = strconv.Itoa(int(ospfInstance.GetUint32Value()))
			case yangOspfAdjInterface:
				ospfAdjObj["interface"] = ospfInstance.GetStringValue()
			case yangOspfAdjNeighborID:
				ospfAdjObj["neighbor_id"] = intToIP4(int64(ospfInstance.GetUint32Value()))

			}
		}
		for _, ospfNbrStatus := range p.Fields[1].Fields {
			switch ospfNbrStatus.GetName() {

			case yangOspfAdjNeighborAddress:
				ospfAdjObj["neighbor_ip"] = ospfNbrStatus.GetStringValue()
			case yangOspfAdjNeighborState:
				ospfAdjObj["neighbor_status"] = ospfNbrStatusToNum(ospfNbrStatus.GetStringValue())

			}
		}

		// Instrument OSPF Adjacency Status

		metricMutex := &sync.Mutex{}
		m := DeviceUnaryMetric{Mutex: metricMutex}

		m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg), prometheus.MustNewConstMetric(
			ospfAdjStatus,
			prometheus.GaugeValue,
			ospfAdjObj["neighbor_status"].(float64),
			msg.GetNodeIdStr(),
			ospfAdjObj["neighbor_id"].(string),
			ospfAdjObj["neighbor_ip"].(string),
			ospfAdjObj["instance_id"].(string),
			ospfAdjObj["interface"].(string),
			ospfAdjObj["area_id"].(string),
		))

		dm.Mutex.Lock()
		dm.Metrics = append(dm.Metrics, m)
		dm.Mutex.Unlock()

	}

}

// intToIP4 is a helper function that converts base 10 IP address into a string
func intToIP4(ipInt int64) string {

	// need to do two bit shifting and “0xff” masking
	b0 := strconv.FormatInt((ipInt>>24)&0xff, 10)
	b1 := strconv.FormatInt((ipInt>>16)&0xff, 10)
	b2 := strconv.FormatInt((ipInt>>8)&0xff, 10)
	b3 := strconv.FormatInt(ipInt&0xff, 10)

	ipOctets := []string{b0, b1, b2, b3}

	return strings.Join(ipOctets, ".")
}

func ospfNbrStatusToNum(status string) float64 {

	nbrStatusMap := map[string]float64{
		"ospf-nbr-down":           1,
		"ospf-nbr-attempt":        2,
		"ospf-nbr-init":           3,
		"ospf-nbr-two-way":        4,
		"ospf-nbr-exchange-start": 5,
		"ospf-nbr-exchange":       6,
		"ospf-nbr-loading":        7,
		"ospf-nbr-full":           8,
	}

	return nbrStatusMap[status]
}
