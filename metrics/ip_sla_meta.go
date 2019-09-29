package metrics

import (
	"strings"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-sla.yang
	IPSLAConfigYANGEncodingPath = "Cisco-IOS-XE-native:native/ip/Cisco-IOS-XE-sla:sla/entry"

	// IP SLA Entry ID
	yangIPSLAConfigID = "number"

	// IP SLA Destination IP (ICMP Echo)
	yangIPSLAConfigDestination = "destination"

	// IP SLA Destination IP (UDP Jitter)
	yangIPSLAConfigDestAddr = "dest-addr"

	// IP SLA Destination Port
	yangIPSLAConfigDestPort = "portno"

	// IP SLA Source IP (UDP Jitter)
	yangIPSLAConfigSourceIP = "source-ip"

	// IP SLA Source Port (UDP Jitter)
	yangIPSLAConfigSourcePort = "source-port"

	// IP SLA Type Of Service
	yangIPSLAConfigTOS = "tos"

	// IP SLA Probe Frequency
	yangIPSLAConfigProbeFrequency = "frequency"

	// IP SLA Probe Request Size
	yangIPSLAConfigProbeReqSize = "request-data-size"

	// IP SLA Tag
	yangIPSLAConfigTag = "tag"

	// IP SLA HTTP GET
	yangIPSLAConfigHTTPGet = "get"

	// IP SLA HTTP Version
	yangIPSLAConfigVRF = "vrf"

	// IP SLA HTTP RAW
	yangIPSLAConfigHTTPRaw = "raw"

	// IP SLA HTTP URL
	yangIPSLAConfigHTTPUrl = "url"

	// IP SLA HTTP NS
	yangIPSLAConfigHTTPNS = "name-server"

	// IP SLA HTTP Version
	yangIPSLAConfigHTTPVersion = "version"

	// IP SLA HTTP Version
	yangIPSLAConfigHTTPProxy = "proxy"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     IPSLAConfigYANGEncodingPath,
		RecordMetricFunc: parseIPSlaConfigPB,
	})
}

func parseIPSlaConfigPB(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	IPSLAConfigSlice := make([]map[string]interface{}, 0, len(msg.DataGpbkv))

	for _, p := range msg.DataGpbkv {

		IPSLAConfig := map[string]interface{}{
			"node_id":          node,
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

		if p.Fields[0].Fields[0].GetName() == yangIPSLAConfigID {
			val := extractGPBKVNativeTypeFromOneof(p.Fields[0].Fields[0], true)
			IPSLAConfig["sla_number"] = int(val.(float64))
		}

		// Ignore IP SLA if not configured with entry type
		if len(p.Fields[1].Fields) == 0 {
			continue
		}

		IPSLAConfig["sla_type"] = p.Fields[1].Fields[0].GetName()

		for _, slaField := range p.Fields[1].Fields[0].Fields {

			switch slaField.GetName() {

			case yangIPSLAConfigDestination:
				IPSLAConfig["destination_ip"] = extractGPBKVNativeTypeFromOneof(slaField, false)
			case yangIPSLAConfigDestAddr:
				IPSLAConfig["destination_ip"] = extractGPBKVNativeTypeFromOneof(slaField, false)
			case yangIPSLAConfigDestPort:
				val := extractGPBKVNativeTypeFromOneof(slaField, true)
				IPSLAConfig["destination_port"] = int(val.(float64))
			case yangIPSLAConfigSourcePort:
				val := extractGPBKVNativeTypeFromOneof(slaField, true)
				IPSLAConfig["source_port"] = int(val.(float64))
			case yangIPSLAConfigSourceIP:
				IPSLAConfig["source_ip"] = extractGPBKVNativeTypeFromOneof(slaField, false)
			case yangIPSLAConfigTOS:
				val := extractGPBKVNativeTypeFromOneof(slaField, true)
				IPSLAConfig["dscp"] = convTOStoDSCP(int(val.(float64)))
			case yangIPSLAConfigProbeFrequency:
				val := extractGPBKVNativeTypeFromOneof(slaField, true)
				IPSLAConfig["frequency"] = int(val.(float64))
			case yangIPSLAConfigProbeReqSize:
				val := extractGPBKVNativeTypeFromOneof(slaField, true)
				IPSLAConfig["req_data_size"] = int(val.(float64))
			case yangIPSLAConfigTag:
				val := extractGPBKVNativeTypeFromOneof(slaField, false)
				cos, dstHost := convIPSlaTagToDesc(val.(string))
				IPSLAConfig["class_of_service"], IPSLAConfig["destination_host"] = cos, dstHost
			case yangIPSLAConfigVRF:
				IPSLAConfig["vrf"] = extractGPBKVNativeTypeFromOneof(slaField, false)

			// Handle Specific fields for HTTP IP SLA
			case yangIPSLAConfigHTTPGet:
				for _, slaHTTP := range slaField.Fields {
					switch slaHTTP.GetName() {
					case yangIPSLAConfigHTTPUrl:
						IPSLAConfig["http_url"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigSourceIP:
						IPSLAConfig["source_ip"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigHTTPNS:
						IPSLAConfig["http_dns_server"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigHTTPVersion:
						IPSLAConfig["http_version"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigSourcePort:
						val := extractGPBKVNativeTypeFromOneof(slaHTTP, true)
						IPSLAConfig["source_port"] = int(val.(float64))
					case yangIPSLAConfigHTTPProxy:
						IPSLAConfig["http_proxy"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					}
				}
			case yangIPSLAConfigHTTPRaw:
				for _, slaHTTP := range slaField.Fields {
					switch slaHTTP.GetName() {
					case yangIPSLAConfigHTTPUrl:
						IPSLAConfig["http_url"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigSourceIP:
						IPSLAConfig["source_ip"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigHTTPNS:
						IPSLAConfig["http_dns_server"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigHTTPVersion:
						IPSLAConfig["http_version"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					case yangIPSLAConfigSourcePort:
						val := extractGPBKVNativeTypeFromOneof(slaHTTP, true)
						IPSLAConfig["source_port"] = int(val.(float64))
					case yangIPSLAConfigHTTPProxy:
						IPSLAConfig["http_proxy"] = extractGPBKVNativeTypeFromOneof(slaHTTP, false)
					}
				}
			}

		}
		IPSLAConfigSlice = append(IPSLAConfigSlice, IPSLAConfig)
	}
	go func() {
		if len(IPSLAConfigSlice) > 0 {
			err := metadb.DBInstance.PersistsIPSlaConfigMetadata(IPSLAConfigSlice, node)

			if err != nil {
				logging.PeppaMonLog("error",
					"Failed to insert IP SLA Config metadata into DB: %v for Node %v", err, node)
			}
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
