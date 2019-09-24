package metrics

import (
	"fmt"
	"strconv"
	"strings"
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
		"BGP Local AS number and router-id",
		[]string{"node", "local_neighbor_id", "local_as"},
		nil,
	)

	bgpIpv4PeerStatus = prometheus.NewDesc(
		"cisco_iosxe_bgp_neighbor_peer_status",
		"The status of a BGP IPv4 unicast peer",
		[]string{"node", "neighbor_id", "address_family", "vrf"},
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

func parseBgpIpv4UnicastPB(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	var BgpIpv4NeighborsSlice []map[string]interface{}

	var BgpIpv4AFISlice []map[string]interface{}

	// Keep function scope variables of BGP Local Neighbor ID and AS for Prometheus metric
	var bgpLocalNeighborID string
	var bgpLocalAS float64

	for _, p := range msg.DataGpbkv {

		BgpIpv4AFIObj := make(map[string]interface{})

		timestamps := t.Unix()

		for _, bgpAFIVRF := range p.Fields[0].Fields {
			switch bgpAFIVRF.GetName() {
			case yangBgpAddressFamily:
				BgpIpv4AFIObj["node_id"] = node
				BgpIpv4AFIObj["timestamps"] = timestamps
				BgpIpv4AFIObj["bgp_address_family_type"] = extractGPBKVNativeTypeFromOneof(bgpAFIVRF, false)

			case yangBgpVRFName:
				val := extractGPBKVNativeTypeFromOneof(bgpAFIVRF, false)
				BgpIpv4AFIObj["bgp_address_family_vrf"] = strings.Replace(val.(string), "default", "Global", 1)

			}
		}

		for _, bgpAFIMeta := range p.Fields[1].Fields {

			switch bgpAFIMeta.GetName() {
			case yangBgpRouterID:
				BgpIpv4AFIObj["bgp_router_id"] = extractGPBKVNativeTypeFromOneof(bgpAFIMeta, false)
				bgpLocalNeighborID = extractGPBKVNativeTypeFromOneof(bgpAFIMeta, false).(string)

			case yangBgpLocalASNumber:
				BgpIpv4AFIObj["bgp_local_as"] = extractGPBKVNativeTypeFromOneof(bgpAFIMeta, true)
				bgpLocalAS = extractGPBKVNativeTypeFromOneof(bgpAFIMeta, true).(float64)

			case yangBgpTotalPrefixes:
				BgpIpv4AFIObj["bgp_afi_total_prefixes"] = extractGPBKVNativeTypeFromOneof(bgpAFIMeta.Fields[0], true)

			case yangBgpTotalPaths:
				BgpIpv4AFIObj["bgp_afi_total_paths"] = extractGPBKVNativeTypeFromOneof(bgpAFIMeta.Fields[0], true)

			case yangBgpNeighborSummary:

				BgpIpv4NeighborObj := make(map[string]interface{})

				// Fetch metadata related to BGP Peer Status
				for _, bgpNei := range bgpAFIMeta.Fields {

					switch bgpNei.GetName() {
					case yangBGPNeighborID:
						BgpIpv4NeighborObj["node_id"] = node
						BgpIpv4NeighborObj["timestamps"] = timestamps
						BgpIpv4NeighborObj["neighbor_id"] = extractGPBKVNativeTypeFromOneof(bgpNei, false)

					case yangBGPNeighborUpTime:
						BgpIpv4NeighborObj["neighbor_uptime"] = extractGPBKVNativeTypeFromOneof(bgpNei, false)

					case yangBgpNeighborPrefixesReceived:
						BgpIpv4NeighborObj["neighbor_prefixes_received"] = extractGPBKVNativeTypeFromOneof(bgpNei, true)

					case yangBGPNeighborRemoteASNumber:
						BgpIpv4NeighborObj["neighbor_remote_as"] = extractGPBKVNativeTypeFromOneof(bgpNei, true)

					case yangBGPNeighborStatus:
						val := extractGPBKVNativeTypeFromOneof(bgpNei, false)
						BgpIpv4NeighborObj["neighbor_status"] = mapBgpNeighborFSMToInteger(val.(string))
					}
					BgpIpv4NeighborObj["address_family_type"] = BgpIpv4AFIObj["bgp_address_family_type"]
					BgpIpv4NeighborObj["address_family_vrf"] = BgpIpv4AFIObj["bgp_address_family_vrf"]

				}
				if neighborID, ok := BgpIpv4NeighborObj["neighbor_id"]; ok {

					// Instrument BGP Prefixes Received per neighbor
					CreatePromMetric(
						BgpIpv4NeighborObj["neighbor_prefixes_received"],
						bgpIpv4NeighborPrefixesRcvd,
						prometheus.GaugeValue,
						dm, t,
						node,
						neighborID.(string),
						BgpIpv4NeighborObj["address_family_type"].(string),
						BgpIpv4AFIObj["bgp_address_family_vrf"].(string),
					)

					// Convert the Peer Status to float64
					peerStatusToFloat, _ := strconv.ParseFloat(BgpIpv4NeighborObj["neighbor_status"].(string), 64)

					// Instrument BGP peer status
					CreatePromMetric(
						peerStatusToFloat,
						bgpIpv4PeerStatus,
						prometheus.GaugeValue,
						dm, t,
						node,
						neighborID.(string),
						BgpIpv4NeighborObj["address_family_type"].(string),
						BgpIpv4AFIObj["bgp_address_family_vrf"].(string),
					)

					BgpIpv4NeighborsSlice = append(BgpIpv4NeighborsSlice, BgpIpv4NeighborObj)
				}

			}

		}
		BgpIpv4AFISlice = append(BgpIpv4AFISlice, BgpIpv4AFIObj)
	}

	// Create BGP local AS and neighbor ID as metric
	CreatePromMetric(
		float64(1),
		bgpGlobalMeta,
		prometheus.GaugeValue,
		dm, t,
		node,
		bgpLocalNeighborID,
		strconv.Itoa(int(bgpLocalAS)),
	)

	// Handle BGP Peers Metadata persistence in separate Go Routine
	go func() {
		err := metadb.DBInstance.PersistsBgpPeersMetadata(BgpIpv4NeighborsSlice, node)

		if err != nil {
			logging.PeppaMonLog(
				"error",
				fmt.Sprintf(
					"Failed to insert BGP Peers metadata for node %v : %v", node, err))
		}
	}()

	// Handle BGP AFI Metadata persistence in separate Go Routine
	go func() {
		err := metadb.DBInstance.PersistsBgpAfiMetadata(BgpIpv4AFISlice, node)

		if err != nil {
			logging.PeppaMonLog(
				"error",
				fmt.Sprintf(
					"Failed to insert BGP AFI metadata for node %v : %v", node, err))
		}
	}()

}

// mapBgpNeighborFSMToInteger is a helper function to map the neighbor FSM status to an integer for Grafana dashboards
func mapBgpNeighborFSMToInteger(status string) string {

	// Neighbor FSM Status mapped to Integer for Grafana cell coloring
	// Workaround until Grafana allows mapping of colors to string values
	peerStatusMap := map[string]string{
		"fsm-idle":        "0",
		"fsm-connect":     "1",
		"fsm-active":      "2",
		"fsm-opensent":    "3",
		"fsm-openconfirm": "4",
		"fsm-established": "5",
	}

	return peerStatusMap[status]
}
