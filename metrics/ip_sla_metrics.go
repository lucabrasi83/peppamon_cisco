package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-ip-sla-oper.yang

	IpSlaOperYANGEncodingPath = "Cisco-IOS-XE-ip-sla-oper:ip-sla-stats/sla-oper-entry"

	// IP SLA Oper Type
	yangIpSlaOperType = "oper-type"

	// IP SLA Oper ID
	yangIpSlaOperID = "oper-id"

	// IP SLA RTT Info
	yangIPSlaOperRTTInfo = "rtt-info"

	// IP SLA Latest RTT
	yangIPSlaOperLatestRTT = "latest-rtt"

	// IP SLA Latest RTT
	yangIPSlaOperLatestRTT2 = "latestrtt"

	// IP SLA Latest RTT
	yangIPSlaOperRTT = "rtt"

	// IP SLA Latest RTT Known
	yangIPSlaOperRTTnown = "rtt-known"

	// IP SLA Oper Type ICMP Echo
	yangIpSlaOperTypeICMPEcho = "oper-type-icmp-echo"

	// IP SLA Oper Type ICMP Echo
	yangIpSlaOperTypeUDPJitter = "oper-type-udp-jitter"

	// IP SLA Oper Type ICMP Echo
	yangIpSlaOperTypeHTTP = "oper-type-http"

	// IP SLA Success Count
	yangIpSlaOperSuccessCount = "success-count"

	// IP SLA Failure Count
	yangIpSlaOperFailureCount = "failure-count"

	// IP SLA Time Values
	yangIpSlaOperTimeValues = "sla-time-values"

	// IP SLA One Way Latency
	yangIpSlaOperOneWayLatency = "oneway-latency"

	// IP SLA One Way Jitter
	yangIpSlaOperOneWayJitter = "jitter"

	// IP SLA Min Value
	yangIpSlaOperMinValue = "min"

	// IP SLA Avg Value
	yangIpSlaOperAvgValue = "avg"

	// IP SLA Max Value
	yangIpSlaOperMaxValue = "max"

	// IP SLA Source To Destination
	yangIpSlaOperSD = "sd"

	// IP SLA Destination to Source
	yangIpSlaDS = "ds"

	// IP SLA One Way Packet Loss
	yangIpSlaOperPacketLoss = "packet-loss"

	// IP SLA SD Packet Loss
	yangIpSlaOperSDPacketLoss = "sd-loss"

	// IP SLA Packet Loss Period Count
	yangIpSlaOperPacketLossPeriodCount = "loss-period-count"

	// IP SLA SD Packet Loss
	yangIpSlaOperDSPacketLoss = "ds-loss"

	// IP SLA HTTP Specific Stats
	yangIpSlaOperHTTPSpecStats = "http-specific-stats"

	// IP SLA HTTP Stats
	yangIpSlaOperHTTPStats = "http-stats"

	// IP SLA Status Code
	yangIpSlaOperHTTPStatusCode = "status-code"

	// IP SLA HTTP DNS RTT
	yangIpSlaOperHTTPDNSRTT = "dns-rtt"

	// IP SLA HTTP Transaction RTT
	yangIpSlaOperHTTPTransactionRTT = "transaction-rtt"
)

var (
	ipSLAProbeRTT = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_rtt_msec",
		"The IP SLA probe reported Round Trip Time in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeFailureCount = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_failure_count",
		"The IP SLA probe failure count",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeSuccessCount = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_success_count",
		"The IP SLA probe success count",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbePacketLossSD = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_packet_loss_sd",
		"The IP SLA probe packet loss source to destination",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbePacketLossDS = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_packet_loss_ds",
		"The IP SLA probe packet loss count destination to source",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayLatencyMinSD = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_min_latency_sd_msec",
		"The IP SLA probe minimum one way latency source to destination in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayLatencyMinDS = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_min_latency_ds_msec",
		"The IP SLA probe minimum one way latency destination to source in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayLatencyAvgSD = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_avg_latency_sd_msec",
		"The IP SLA probe average one way latency source to destination in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayLatencyAvgDS = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_avg_latency_ds_msec",
		"The IP SLA probe average one way latency destination to source in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayLatencyMaxSD = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_max_latency_sd_msec",
		"The IP SLA probe maximum one way latency source to destination in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayLatencyMaxDS = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_max_latency_ds_msec",
		"The IP SLA probe maximum one way latency destination to source in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayJitterMinSD = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_min_jitter_sd_msec",
		"The IP SLA probe minimum jitter source to destination in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayJitterMinDS = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_min_jitter_ds_msec",
		"The IP SLA probe minimum jitter destination to source in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayJitterMaxDS = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_max_jitter_ds_msec",
		"The IP SLA probe maximum jitter destination to source in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
	ipSLAProbeOneWayJitterMaxSD = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_max_jitter_sd_msec",
		"The IP SLA probe maximum jitter source to destination in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayJitterAvgSD = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_avg_jitter_sd_msec",
		"The IP SLA probe average jitter source to destination in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeOneWayJitterAvgDS = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_avg_jitter_ds_msec",
		"The IP SLA probe average jitter destination to source in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
	ipSLAProbeHTTPStatusCode = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_status_code",
		"The IP SLA probe HTTP Status Code",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
	ipSLAProbeHTTPTransactionRTT = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_transaction_rtt_msec",
		"The IP SLA probe HTTP Transaction Round Trip Time in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
	ipSLAProbeHTTPDNSRTT = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_dns_rtt_msec",
		"The IP SLA probe HTTP DNS lookup Round Trip Time in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     IpSlaOperYANGEncodingPath,
		RecordMetricFunc: parseIPSlaMetricsPB,
	})
}

func parseIPSlaMetricsPB(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time) {

	for _, p := range msg.DataGpbkv {

		var slaEntryID string
		var slaType string

		if p.Fields[0].Fields[0].GetName() == yangIpSlaOperID {

			id := int(p.Fields[0].Fields[0].GetUint32Value())
			slaEntryID = strconv.Itoa(id)
		}

		for _, slaFields := range p.Fields[1].Fields {

			switch slaFields.GetName() {
			case yangIpSlaOperType:
				slaType = slaFields.GetStringValue()
			case yangIpSlaOperSuccessCount:
				slaSuccessCount := float64(slaFields.GetUint32Value())

				metricMutex := &sync.Mutex{}
				m := DeviceUnaryMetric{Mutex: metricMutex}

				m.Metric = prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(
					ipSLAProbeSuccessCount,
					prometheus.GaugeValue,
					slaSuccessCount,
					msg.GetNodeIdStr(),
					slaEntryID,
					slaType,
				))
				dm.Mutex.Lock()
				dm.Metrics = append(dm.Metrics, m)
				dm.Mutex.Unlock()

			case yangIpSlaOperFailureCount:
				slaFailureCount := float64(slaFields.GetUint32Value())

				metricMutex := &sync.Mutex{}
				m := DeviceUnaryMetric{Mutex: metricMutex}

				m.Metric = prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(
					ipSLAProbeFailureCount,
					prometheus.GaugeValue,
					slaFailureCount,
					msg.GetNodeIdStr(),
					slaEntryID,
					slaType,
				))
				dm.Mutex.Lock()
				dm.Metrics = append(dm.Metrics, m)
				dm.Mutex.Unlock()

			case yangIPSlaOperRTTInfo:
				if slaType == yangIpSlaOperTypeICMPEcho || slaType == yangIpSlaOperTypeHTTP {

					if slaFields.Fields[0].Fields[0].GetStringValue() == yangIPSlaOperRTTnown {

						val := slaFields.Fields[0].Fields[1].GetUint64Value()

						metricMutex := &sync.Mutex{}
						m := DeviceUnaryMetric{Mutex: metricMutex}

						m.Metric = prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(
							ipSLAProbeRTT,
							prometheus.GaugeValue,
							float64(val),
							msg.GetNodeIdStr(),
							slaEntryID,
							slaType,
						))
						dm.Mutex.Lock()
						dm.Metrics = append(dm.Metrics, m)
						dm.Mutex.Unlock()
					}
				}
			}
		}

	}

}
