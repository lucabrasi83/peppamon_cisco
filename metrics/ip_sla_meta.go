package metrics

import "github.com/lucabrasi83/peppamon_cisco/proto/telemetry"

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-sla.yang
	IpSlaConfigYANGEncodingPath = "Cisco-IOS-XE-native:native/ip/Cisco-IOS-XE-sla:sla/entry"

	// IP SLA Entry ID
	yangIpSlaConfigID = "number"

	// IP SLA Type icmp-echo
	yangIpSlaConfigICMPEcho = "icmp-echo"

	// IP SLA Type udp-jitter
	yangIpSlaConfigUDPJitter = "udp-jitter"

	// IP SLA Type http
	yangIpSlaConfigHTTP = "http"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     IpSlaConfigYANGEncodingPath,
		RecordMetricFunc: parseIPSlaConfigPB,
	})
}

func parseIPSlaConfigPB(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics) {

	var IPSlaConfigSlice []map[string]interface{}

	for _, p := range msg.DataGpbkv {
		for _, slaConfigContent := range p.Fields {

		}
	}
}
