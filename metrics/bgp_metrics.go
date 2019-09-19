package metrics

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	bgpIpv4NeighborPrefixesRcvd = prometheus.NewDesc(
		"cisco_iosxe_bgp_neighbor_prefixes_received",
		"The number of prefixes received from BGP IPv4 unicast peer",
		[]string{"node", "neighbor_id", "address_family", "vrf"},
		nil,
	)

	bgpGlobalMeta = prometheus.NewDesc(
		"cisco_iosxe_bgp_global_meta",
		"The number of prefixes received from BGP IPv4 unicast peer",
		[]string{"node", "local_neighbor_id", "local_as"},
		nil,
	)
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-bgp-oper.yang
	BgpOperYANGEncodingPath = "Cisco-IOS-XE-bgp-oper:bgp-state-data/address-families/address-family"

	// Router ID
	yangBgpRouterID = "router-id"

	// Total Prefix entry statistics
	yangBgpTotalPrefixes = "prefixes"

	// Total Paths
	yangBgpTotalPaths = "path"

	// Local AS Number
	yangBgpLocalASNumber = "local-as"

	// BGP Address Family
	yangBgpAddressFamily = "afi-safi"

	// VRF Name
	yangBgpVRFName = "vrf-name"

	// BGP Neighor Details
	yangBgpNeighborSummary = "bgp-neighbor-summary"

	// BGP Neighbor Prefixes Received
	yangBgpNeighborPrefixesReceived = "prefixes-received"

	// BGP Neighbor Remote AS Number
	yangBGPNeighborRemoteASNumber = "as"

	// BGP Neighbor Uptime
	yangBGPNeighborUpTime = "up-time"

	// BGP Neighbor ID
	yangBGPNeighborID = "id"

	// BGP Neighbor Status
	yangBGPNeighborStatus = "state"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     BgpOperYANGEncodingPath,
		RecordMetricFunc: parseBgpIpv4UnicastPB,
	})
}

func parseBgpIpv4UnicastPB(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics) {

	var BgpIpv4NeighborsSlice []map[string]interface{}

	var BgpIpv4AFISlice []map[string]interface{}

	// Keep function scope variables of BGP Local Neighbor ID and AS for Prometheus metric
	var bgpLocalNeighborID string
	var bgpLocalAS uint32

	for _, p := range msg.DataGpbkv {

		BgpIpv4AFIObj := make(map[string]interface{})

		timestamps := time.Now().Unix()

		for _, bgpAFIVRF := range p.Fields[0].Fields {
			switch bgpAFIVRF.GetName() {
			case yangBgpAddressFamily:
				BgpIpv4AFIObj["node_id"] = msg.GetNodeIdStr()
				BgpIpv4AFIObj["timestamps"] = timestamps
				BgpIpv4AFIObj["bgp_address_family_type"] = bgpAFIVRF.GetStringValue()

			case yangBgpVRFName:

				BgpIpv4AFIObj["bgp_address_family_vrf"] = strings.Replace(
					bgpAFIVRF.GetStringValue(), "default", "Global", 1)

			}
		}

		for _, bgpAFIMeta := range p.Fields[1].Fields {

			switch bgpAFIMeta.GetName() {
			case yangBgpRouterID:
				BgpIpv4AFIObj["bgp_router_id"] = bgpAFIMeta.GetStringValue()
				bgpLocalNeighborID = bgpAFIMeta.GetStringValue()

			case yangBgpLocalASNumber:
				BgpIpv4AFIObj["bgp_local_as"] = bgpAFIMeta.GetUint32Value()
				bgpLocalAS = bgpAFIMeta.GetUint32Value()

			case yangBgpTotalPrefixes:
				BgpIpv4AFIObj["bgp_afi_total_prefixes"] = bgpAFIMeta.Fields[0].GetUint64Value()

			case yangBgpTotalPaths:
				BgpIpv4AFIObj["bgp_afi_total_paths"] = bgpAFIMeta.Fields[0].GetUint64Value()

			case yangBgpNeighborSummary:

				BgpIpv4NeighborObj := make(map[string]interface{})

				// Fetch metadata related to BGP Peer Status
				for _, bgpNei := range bgpAFIMeta.Fields {

					switch bgpNei.GetName() {
					case yangBGPNeighborID:
						BgpIpv4NeighborObj["node_id"] = msg.GetNodeIdStr()
						BgpIpv4NeighborObj["timestamps"] = timestamps
						BgpIpv4NeighborObj["neighbor_id"] = bgpNei.GetStringValue()

					case yangBGPNeighborUpTime:
						BgpIpv4NeighborObj["neighbor_uptime"] = bgpNei.GetStringValue()

					case yangBgpNeighborPrefixesReceived:
						BgpIpv4NeighborObj["neighbor_prefixes_received"] = bgpNei.GetUint64Value()

					case yangBGPNeighborRemoteASNumber:
						BgpIpv4NeighborObj["neighbor_remote_as"] = bgpNei.GetUint32Value()

					case yangBGPNeighborStatus:
						BgpIpv4NeighborObj["neighbor_status"] = mapBgpNeighborFSMToInteger(bgpNei.GetStringValue())
					}
					BgpIpv4NeighborObj["address_family_type"] = BgpIpv4AFIObj["bgp_address_family_type"]
					BgpIpv4NeighborObj["address_family_vrf"] = BgpIpv4AFIObj["bgp_address_family_vrf"]

				}
				if neighborID, ok := BgpIpv4NeighborObj["neighbor_id"]; ok {

					// Instrument BGP Prefixes Received per neighbor

					metricMutex := &sync.Mutex{}
					m := DeviceUnaryMetric{Mutex: metricMutex}

					m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg),
						prometheus.MustNewConstMetric(
							bgpIpv4NeighborPrefixesRcvd,
							prometheus.GaugeValue,
							float64(BgpIpv4NeighborObj["neighbor_prefixes_received"].(uint64)),
							msg.GetNodeIdStr(),
							neighborID.(string),
							BgpIpv4NeighborObj["address_family_type"].(string),
							BgpIpv4AFIObj["bgp_address_family_vrf"].(string),
						))

					dm.Mutex.Lock()
					dm.Metrics = append(dm.Metrics, m)
					dm.Mutex.Unlock()

					BgpIpv4NeighborsSlice = append(BgpIpv4NeighborsSlice, BgpIpv4NeighborObj)
				}

			}

		}
		BgpIpv4AFISlice = append(BgpIpv4AFISlice, BgpIpv4AFIObj)
	}

	metricMutex := &sync.Mutex{}
	m := DeviceUnaryMetric{Mutex: metricMutex}

	m.Metric = prometheus.NewMetricWithTimestamp(convTelemetryTimestampToTime(msg), prometheus.MustNewConstMetric(
		bgpGlobalMeta,
		prometheus.GaugeValue,
		1,
		msg.GetNodeIdStr(),
		bgpLocalNeighborID,
		strconv.Itoa(int(bgpLocalAS)),
	))

	dm.Mutex.Lock()
	dm.Metrics = append(dm.Metrics, m)
	dm.Mutex.Unlock()

	// Handle BGP Peers Metadata persistence in separate Go Routine
	go func() {
		err := metadb.DBInstance.PersistsBgpPeersMetadata(BgpIpv4NeighborsSlice, msg.GetNodeIdStr())

		if err != nil {
			logging.PeppaMonLog(
				"error",
				fmt.Sprintf(
					"Failed to insert BGP Peers metadata for node %v : %v", msg.GetNodeIdStr(), err))
		}
	}()

	// Handle BGP AFI Metadata persistence in separate Go Routine
	go func() {
		err := metadb.DBInstance.PersistsBgpAfiMetadata(BgpIpv4AFISlice, msg.GetNodeIdStr())

		if err != nil {
			logging.PeppaMonLog(
				"error",
				fmt.Sprintf(
					"Failed to insert BGP AFI metadata for node %v : %v", msg.GetNodeIdStr(), err))
		}
	}()

}

// mapBgpNeighborFSMToInteger is a helper function to map the neighbor FSM status to an integer for Grafana dashboards
func mapBgpNeighborFSMToInteger(status string) string {

	// Neighbor FSM Status mapped to Integer for Grafana cell coloring
	// Workaround until Grafana allows mapping of colors to string values
	ifStatusMap := map[string]string{
		"fsm-idle":        "0",
		"fsm-connect":     "1",
		"fsm-active":      "2",
		"fsm-opensent":    "3",
		"fsm-openconfirm": "4",
		"fsm-established": "5",
	}

	return ifStatusMap[status]
}
