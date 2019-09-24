package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-sla.yang
	IpSlaConfigYANGEncodingPath = "Cisco-IOS-XE-native:native/ip/Cisco-IOS-XE-sla:sla/entry"

	// IP SLA Entry ID
	yangIpSlaConfigID = "number"

	// IP SLA Destination IP (ICMP Echo)
	yangIpSlaConfigDestination = "destination"

	// IP SLA Destination IP (UDP Jitter)
	yangIpSlaConfigDestAddr = "dest-addr"

	// IP SLA Destination Port
	yangIpSlaConfigDestPort = "portno"

	// IP SLA Source IP (UDP Jitter)
	yangIpSlaConfigSourceIP = "source-ip"

	// IP SLA Source Port (UDP Jitter)
	yangIpSlaConfigSourcePort = "source-port"

	// IP SLA Type Of Service
	yangIpSlaConfigTOS = "tos"

	// IP SLA Probe Frequency
	yangIpSlaConfigProbeFrequency = "frequency"

	// IP SLA Probe Request Size
	yangIpSlaConfigProbeReqSize = "request-data-size"

	// IP SLA Tag
	yangIpSlaConfigTag = "tag"

	// IP SLA HTTP GET
	yangIpSlaConfigHTTPGet = "get"

	// IP SLA HTTP Version
	yangIpSlaConfigVRF = "vrf"

	// IP SLA HTTP RAW
	yangIpSlaConfigHTTPRaw = "raw"

	// IP SLA HTTP URL
	yangIpSlaConfigHTTPUrl = "url"

	// IP SLA HTTP NS
	yangIpSlaConfigHTTPNS = "name-server"

	// IP SLA HTTP Version
	yangIpSlaConfigHTTPVersion = "version"

	// IP SLA HTTP Version
	yangIpSlaConfigHTTPProxy = "proxy"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     IpSlaConfigYANGEncodingPath,
		RecordMetricFunc: parseIPSlaConfigPB,
	})
}

func parseIPSlaConfigPB(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time) {

	IPSlaConfigSlice := make([]map[string]interface{}, 0, len(msg.DataGpbkv))

	for _, p := range msg.DataGpbkv {

		IPSlaConfig := map[string]interface{}{
			"node_id":          msg.GetNodeIdStr(),
			"sla_number":       0,
			"sla_type":         "N/A",
			"destination_ip":   "N/A",
			"source_ip":        "N/A",
			"source_port":      0,
			"destination_port": 0,
			"class_of_service": "N/A",
			"destination_host": "N/A",
			"dscp":             "N/A",
			"req_data_size":    0,
			"frequency":        60,
			"vrf":              "Global",
			"http_proxy":       "N/A",
			"http_url":         "N/A",
			"http_version":     "N/A",
			"http_dns_server":  "N/A",
			"timestamps":       t.Unix(),
		}

		if p.Fields[0].Fields[0].GetName() == yangIpSlaConfigID {
			IPSlaConfig["sla_number"] = int(p.Fields[0].Fields[0].GetUint32Value())
		}

		// Ignore IP SLA if not configured with entry type
		if len(p.Fields[1].Fields) == 0 {
			continue
		}

		IPSlaConfig["sla_type"] = p.Fields[1].Fields[0].GetName()

		for _, slaField := range p.Fields[1].Fields[0].Fields {

			switch slaField.GetName() {

			case yangIpSlaConfigDestination:
				IPSlaConfig["destination_ip"] = slaField.GetStringValue()
			case yangIpSlaConfigDestAddr:
				IPSlaConfig["destination_ip"] = slaField.GetStringValue()
			case yangIpSlaConfigDestPort:
				IPSlaConfig["destination_port"] = int(slaField.GetUint32Value())
			case yangIpSlaConfigSourcePort:
				IPSlaConfig["source_port"] = int(slaField.GetUint32Value())
			case yangIpSlaConfigSourceIP:
				IPSlaConfig["source_ip"] = slaField.GetStringValue()
			case yangIpSlaConfigTOS:
				IPSlaConfig["dscp"] = convTOStoDSCP(int(slaField.GetUint32Value()))
			case yangIpSlaConfigProbeFrequency:
				IPSlaConfig["frequency"] = int(slaField.GetUint32Value())
			case yangIpSlaConfigProbeReqSize:
				IPSlaConfig["req_data_size"] = int(slaField.GetUint32Value())
			case yangIpSlaConfigTag:
				cos, dstHost := convIPSlaTagToDesc(slaField.GetStringValue())
				IPSlaConfig["class_of_service"], IPSlaConfig["destination_host"] = cos, dstHost
			case yangIpSlaConfigVRF:
				IPSlaConfig["vrf"] = slaField.GetStringValue()

			// Handle Specific fields for HTTP IP SLA
			case yangIpSlaConfigHTTPGet:
				for _, slaHTTP := range slaField.Fields {
					switch slaHTTP.GetName() {
					case yangIpSlaConfigHTTPUrl:
						IPSlaConfig["http_url"] = slaHTTP.GetStringValue()
					case yangIpSlaConfigSourceIP:
						IPSlaConfig["source_ip"] = slaField.GetStringValue()
					case yangIpSlaConfigHTTPNS:
						IPSlaConfig["http_dns_server"] = slaHTTP.GetStringValue()
					case yangIpSlaConfigHTTPVersion:
						IPSlaConfig["http_version"] = slaHTTP.GetStringValue()
					case yangIpSlaConfigSourcePort:
						IPSlaConfig["source_port"] = int(slaField.GetUint32Value())
					case yangIpSlaConfigHTTPProxy:
						IPSlaConfig["http_proxy"] = slaHTTP.GetStringValue()
					}
				}
			case yangIpSlaConfigHTTPRaw:
				for _, slaHTTP := range slaField.Fields {
					switch slaHTTP.GetName() {
					case yangIpSlaConfigHTTPUrl:
						IPSlaConfig["http_url"] = slaHTTP.GetStringValue()
					case yangIpSlaConfigSourceIP:
						IPSlaConfig["source_ip"] = slaField.GetStringValue()
					case yangIpSlaConfigHTTPNS:
						IPSlaConfig["http_dns_server"] = slaHTTP.GetStringValue()
					case yangIpSlaConfigHTTPVersion:
						IPSlaConfig["http_version"] = slaHTTP.GetStringValue()
					case yangIpSlaConfigSourcePort:
						IPSlaConfig["source_port"] = int(slaField.GetUint32Value())
					case yangIpSlaConfigHTTPProxy:
						IPSlaConfig["http_proxy"] = slaHTTP.GetStringValue()
					}
				}
			}

		}
		IPSlaConfigSlice = append(IPSlaConfigSlice, IPSlaConfig)
	}
	go func() {
		err := metadb.DBInstance.PersistsIPSlaConfigMetadata(IPSlaConfigSlice, msg.GetNodeIdStr())

		if err != nil {
			logging.PeppaMonLog("error",
				fmt.Sprintf("Failed to insert IP SLA Config metadata into DB: %v for Node %v", err, msg.GetNodeIdStr()))
		}

	}()

}

// convTOStoDSCP is a convenience function to convert the ToS value as a DSCP and returns the DSCP class
func convTOStoDSCP(tos int) string {

	// DSCP Decimal Value
	dscpDecimal := tos >> 2

	dscpMapDecToClass := map[int]string{
		8:  "CS1",
		10: "AF11",
		12: "AF12",
		14: "AF13",
		16: "CS2",
		18: "AF21",
		20: "AF22",
		22: "AF23",
		24: "CS3",
		26: "AF31",
		28: "AF32",
		30: "AF33",
		32: "CS4",
		34: "AF41",
		36: "AF42",
		38: "AF43",
		40: "CS5",
		46: "EF",
		48: "CS6",
		56: "CS7",
	}

	return dscpMapDecToClass[dscpDecimal]

}

// convIPSlaTagToDesc is a helper function to convert an IP SLA Tag into a meaningful description
// It is decomposing the standard model tag in the configuration to describe the IP SLA
// Class Of Service and Destination Hostname
func convIPSlaTagToDesc(tag string) (string, string) {

	if tag == "" {
		return "N/A", "N/A"
	}

	tagSplit := strings.Split(tag, "_")

	if len(tagSplit) == 3 && strings.Contains(tagSplit[0], "COS") {
		cos := tagSplit[0]
		dstHost := strings.Split(tagSplit[2], ":")[0]

		return cos, dstHost
	}

	return "N/A", "N/A"

}
