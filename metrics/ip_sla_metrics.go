package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-ip-sla-oper.yang

	IPSLAOperYANGEncodingPath = "Cisco-IOS-XE-ip-sla-oper:ip-sla-stats/sla-oper-entry"

	// IP SLA Oper Type
	yangIPSLAOperType = "oper-type"

	// IP SLA Oper ID
	yangIPSLAOperID = "oper-id"

	// IP SLA RTT Info
	yangIPSLAOperRTTInfo = "rtt-info"

	// IP SLA Latest RTT
	yangIPSLAOperLatestRTT = "latest-rtt"

	// IP SLA Latest RTT
	yangIPSLAOperRTT = "rtt"

	// IP SLA RTT Info
	yangIPSLAOperReturnCode = "latest-return-code"

	// IP SLA RTT Info
	yangIPSLALatestStartTime = "latest-oper-start-time"

	// IP SLA Oper Type ICMP Echo
	yangIPSLAOperTypeUDPJitter = "oper-type-udp-jitter"

	// IP SLA Oper Type ICMP Echo
	yangIPSLAOperTypeHTTP = "oper-type-http"

	// IP SLA Success Count
	yangIPSLAOperSuccessCount = "success-count"

	// IP SLA Failure Count
	yangIPSLAOperFailureCount = "failure-count"

	// IP SLA HTTP Stats
	yangIPSLAOperStats = "stats"

	// IP SLA One Way Latency
	yangIPSLAOperOneWayLatency = "oneway-latency"

	// IP SLA One Way Jitter
	yangIPSLAOperOneWayJitter = "jitter"

	// IP SLA Min Value
	yangIPSLAOperMinValue = "min"

	// IP SLA Avg Value
	yangIPSLAOperAvgValue = "avg"

	// IP SLA Max Value
	yangIPSLAOperMaxValue = "max"

	// IP SLA Source To Destination
	yangIPSLAOperSD = "sd"

	// IP SLA Destination to Source
	yangIPSLAOperDS = "ds"

	// IP SLA One Way Packet Loss
	yangIPSLAOperPacketLoss = "packet-loss"

	// IP SLA SD Packet Loss
	yangIPSLAOperSDPacketLoss = "sd-loss"

	// IP SLA Packet Loss Period Count
	yangIPSLAOperPacketLossPeriodCount = "loss-period-count"

	// IP SLA SD Packet Loss
	yangIPSLAOperDSPacketLoss = "ds-loss"

	// IP SLA HTTP Specific Stats
	yangIPSLAOperHTTPSpecStats = "http-specific-stats"

	// IP SLA HTTP Stats
	yangIPSLAOperHTTPStats = "http-stats"

	// IP SLA Status Code
	yangIPSLAOperHTTPStatusCode = "status-code"

	// IP SLA HTTP DNS RTT
	yangIPSLAOperHTTPDNSRTT = "dns-rtt"

	// IP SLA HTTP Transaction RTT
	yangIPSLAOperHTTPTransactionRTT = "transaction-rtt"

	// IP SLA HTTP Errors
	yangIPSLAOperHTTPErrors = "http-errors"

	// IP SLA HTTP Transaction Error
	yangIPSLAOperHTTPTransactionError = "transaction-error"

	// IP SLA HTTP TCP Error
	yangIPSLAOperHTTPTCPError = "tcp-error"

	// IP SLA HTTP DNS Error
	yangIPSLAOperHTTPDNSError = "dns-error"

	// IP SLA HTTP Transaction Timeout
	yangIPSLAOperHTTPTransactionTimeout = "transaction-timeout"

	// IP SLA HTTP TCP Timeout
	yangIPSLAOperHTTPTCPTimeout = "tcp-timeout"

	// IP SLA HTTP DNS Timeout
	yangIPSLAOperHTTPDNSTimeout = "dns-timeout"
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

	ipSLAProbeLatestReturnCode = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_latest_return_code",
		"The IP SLA Latest Return Code",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeLatestOperTime = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_latest_operation_time_epoch",
		"The IP SLA Latest Operation start in epoch time",
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
		"The HTTP IP SLA probe Status Code",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
	ipSLAProbeHTTPTransactionRTT = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_transaction_rtt_msec",
		"The HTTP IP SLA probe Transaction Round Trip Time in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
	ipSLAProbeHTTPDNSRTT = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_dns_rtt_msec",
		"The HTTP IP SLA probe DNS lookup Round Trip Time in milliseconds",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeHTTPTransactionError = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_transaction_error",
		"The HTTP IP SLA probe number of HTTP transaction errors occurred",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeHTTPTCPError = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_tcp_error",
		"The HTTP IP SLA probe number of TCP errors occurred",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeHTTPDNSError = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_dns_error",
		"The HTTP IP SLA probe number of DNS errors occurred",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeHTTPTransactionTimeout = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_transaction_timeout",
		"The HTTP IP SLA probe number of HTTP transaction timeout occurred",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeHTTPTCPTimeout = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_tcp_timeout",
		"The HTTP IP SLA probe number of TCP timeout occurred",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)

	ipSLAProbeHTTPDNSTimeout = prometheus.NewDesc(
		"cisco_iosxe_ip_sla_probe_http_dns_timeout",
		"The HTTP IP SLA probe number of DNS timeout occurred",
		[]string{"node", "sla_entry_id", "sla_type"},
		nil,
	)
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     IPSLAOperYANGEncodingPath,
		RecordMetricFunc: parseIPSlaMetricsPB,
	})
}

func parseIPSlaMetricsPB(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	for _, p := range msg.DataGpbkv {

		var slaEntryID string
		var slaType string

		if p.Fields[0].Fields[0].GetName() == yangIPSLAOperID {

			id := extractGPBKVNativeTypeFromOneof(p.Fields[0].Fields[0], true)
			id = int(id.(float64))
			slaEntryID = strconv.Itoa(id.(int))
		}

		for _, slaFields := range p.Fields[1].Fields {

			switch slaFields.GetName() {
			case yangIPSLAOperType:
				slaType = extractGPBKVNativeTypeFromOneof(slaFields, false).(string)

			case yangIPSLAOperReturnCode:
				val := extractGPBKVNativeTypeFromOneof(slaFields, false).(string)

				if code, ok := convIPSLAReturnCodeToFloat(val); ok {
					CreatePromMetric(
						code,
						ipSLAProbeLatestReturnCode,
						prometheus.GaugeValue,
						dm, t, node, slaEntryID, slaType)
				}

			case yangIPSLALatestStartTime:
				val := extractGPBKVNativeTypeFromOneof(slaFields, false).(string)

				timeObj, err := time.Parse(time.RFC3339, val)

				if err != nil {
					logging.PeppaMonLog("error",
						fmt.Sprintf("Failed to convert IP SLA Start Time"))
				} else {
					CreatePromMetric(
						float64(timeObj.UTC().Unix()),
						ipSLAProbeLatestOperTime,
						prometheus.GaugeValue,
						dm, t, node, slaEntryID, slaType)

				}

			case yangIPSLAOperSuccessCount:
				slaSuccessCount := extractGPBKVNativeTypeFromOneof(slaFields, true)

				CreatePromMetric(
					slaSuccessCount,
					ipSLAProbeSuccessCount,
					prometheus.CounterValue,
					dm, t, node, slaEntryID, slaType)

			case yangIPSLAOperFailureCount:
				slaFailureCount := extractGPBKVNativeTypeFromOneof(slaFields, true)

				CreatePromMetric(
					slaFailureCount,
					ipSLAProbeFailureCount,
					prometheus.CounterValue,
					dm, t, node, slaEntryID, slaType)

				// Round Trip Time for all IP SLA probes types
			case yangIPSLAOperRTTInfo:
				for _, measure := range slaFields.Fields {
					switch measure.GetName() {
					case yangIPSLAOperLatestRTT:
						for _, v := range measure.Fields {
							if v.GetName() == yangIPSLAOperRTT {
								val := extractGPBKVNativeTypeFromOneof(v, true)
								CreatePromMetric(
									val,
									ipSLAProbeRTT,
									prometheus.GaugeValue,
									dm, t, node, slaEntryID, slaType)
							}
						}
					}
				}

			case yangIPSLAOperStats:

				for _, statField := range slaFields.Fields {
					// UDP Jitter IP SLA metrics
					if slaType == yangIPSLAOperTypeUDPJitter {
						switch statField.GetName() {

						// One Way Latency Stats in YANG Schema
						case yangIPSLAOperOneWayLatency:
							for _, measure := range statField.Fields {
								switch measure.GetName() {

								// Source to Destination One Way Latency Stats in YANG Schema
								case yangIPSLAOperSD:
									for _, v := range measure.Fields {
										switch v.GetName() {

										// Source to Destination Minimum One Way Latency Stats in YANG Schema
										case yangIPSLAOperMinValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayLatencyMinSD,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Source to Destination Maximum One Way Latency Stats in YANG Schema
										case yangIPSLAOperMaxValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayLatencyMaxSD,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Source to Destination Average One Way Latency Stats in YANG Schema
										case yangIPSLAOperAvgValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayLatencyAvgSD,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)
										}
									}
									// Destination to Source One Way Latency Stats in YANG Schema
								case yangIPSLAOperDS:
									for _, v := range measure.Fields {
										switch v.GetName() {

										// Destination to Source Minimum One Way Latency Stats in YANG Schema
										case yangIPSLAOperMinValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayLatencyMinDS,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Destination to Source Maximum One Way Latency Stats in YANG Schema
										case yangIPSLAOperMaxValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayLatencyMaxDS,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Destination to Source Average One Way Latency Stats in YANG Schema
										case yangIPSLAOperAvgValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayLatencyAvgDS,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)
										}
									}
								}

							}
							// One Way Jitter Stats in YANG Schema
						case yangIPSLAOperOneWayJitter:
							for _, measure := range statField.Fields {
								switch measure.GetName() {

								// Source to Destination One Way Jitter Stats in YANG Schema
								case yangIPSLAOperSD:
									for _, v := range measure.Fields {
										switch v.GetName() {

										// Source to Destination Minimum One Way Jitter Stats in YANG Schema
										case yangIPSLAOperMinValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayJitterMinSD,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Source to Destination Maximum One Way Jitter Stats in YANG Schema
										case yangIPSLAOperMaxValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayJitterMaxSD,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Source to Destination Average One Way Jitter Stats in YANG Schema
										case yangIPSLAOperAvgValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayJitterAvgSD,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)
										}
									}
									// Destination to Source One Way Jitter Stats in YANG Schema
								case yangIPSLAOperDS:
									for _, v := range measure.Fields {
										switch v.GetName() {

										// Destination to Source Minimum One Way Jitter Stats in YANG Schema
										case yangIPSLAOperMinValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayJitterMinDS,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Destination to Source Maximum One Way Jitter Stats in YANG Schema
										case yangIPSLAOperMaxValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayJitterMaxDS,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Destination to Source Average One Way Jitter Stats in YANG Schema
										case yangIPSLAOperAvgValue:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeOneWayJitterAvgDS,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)
										}
									}
								}

							}

							// Packet Loss Stats in YANG Schema
						case yangIPSLAOperPacketLoss:
							for _, measure := range statField.Fields {
								switch measure.GetName() {

								// Source to Destination Packet Loss Stats in YANG Schema
								case yangIPSLAOperSDPacketLoss:
									for _, v := range measure.Fields {
										switch v.GetName() {

										// Source to Destination Packet Loss Period count Stats in YANG Schema
										case yangIPSLAOperPacketLossPeriodCount:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbePacketLossSD,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)
										}
									}
									// Destination to Source Packet Loss Stats in YANG Schema
								case yangIPSLAOperDSPacketLoss:
									for _, v := range measure.Fields {
										switch v.GetName() {

										// Destination to SourcePacket Loss Stats in YANG Schema
										case yangIPSLAOperPacketLossPeriodCount:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbePacketLossDS,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)
										}
									}
								}

							}

						}
					}
					// HTTP IP SLA metrics in YANG model
					if slaType == yangIPSLAOperTypeHTTP {

						switch statField.GetName() {

						// HTTP IP SLA metrics
						case yangIPSLAOperHTTPSpecStats:

							for _, measure := range statField.Fields {

								switch measure.GetName() {
								// HTTP Stats in YANG Schema
								case yangIPSLAOperHTTPStats:
									for _, v := range measure.Fields {
										switch v.GetName() {
										// HTTP Status Code in YANG Schema
										case yangIPSLAOperHTTPStatusCode:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPStatusCode,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)
										// HTTP DNS RTT in YANG Schema
										case yangIPSLAOperHTTPDNSRTT:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPDNSRTT,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

											// Source to Destination Maximum One Way Latency Stats in YANG Schema
										case yangIPSLAOperHTTPTransactionRTT:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPTransactionRTT,
												prometheus.GaugeValue,
												dm, t, node, slaEntryID, slaType)

										}

									}
									// HTTP Error Stats in YANG Schema
								case yangIPSLAOperHTTPErrors:
									for _, v := range measure.Fields {
										switch v.GetName() {
										// HTTP Transaction Errors
										case yangIPSLAOperHTTPTransactionError:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPTransactionError,
												prometheus.CounterValue,
												dm, t, node, slaEntryID, slaType)
										// HTTP TCP Errors
										case yangIPSLAOperHTTPTCPError:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPTCPError,
												prometheus.CounterValue,
												dm, t, node, slaEntryID, slaType)

											// HTTP DNS Errors
										case yangIPSLAOperHTTPDNSError:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPDNSError,
												prometheus.CounterValue,
												dm, t, node, slaEntryID, slaType)

											// HTTP Transaction Timeouts
										case yangIPSLAOperHTTPTransactionTimeout:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPTransactionTimeout,
												prometheus.CounterValue,
												dm, t, node, slaEntryID, slaType)

											// HTTP TCP Timeouts
										case yangIPSLAOperHTTPTCPTimeout:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPTCPTimeout,
												prometheus.CounterValue,
												dm, t, node, slaEntryID, slaType)

											// HTTP DNS Timeouts
										case yangIPSLAOperHTTPDNSTimeout:
											val := extractGPBKVNativeTypeFromOneof(v, true)
											CreatePromMetric(
												val,
												ipSLAProbeHTTPDNSTimeout,
												prometheus.CounterValue,
												dm, t, node, slaEntryID, slaType)

										}

									}

								}
							}
						}

					}
				}

			}
		}

	}
}

func convIPSLAReturnCodeToFloat(rc string) (float64, bool) {
	returnCodeMap := map[string]float64{
		"ret-code-unknown":             0,
		"ret-code-ok":                  1,
		"ret-code-disconnected":        2,
		"ret-code-busy":                3,
		"ret-code-timeout":             4,
		"ret-code-no-connection":       5,
		"ret-code-internal-error":      6,
		"ret-code-operation-failure":   7,
		"ret-code-code-could-not-find": 8,
	}
	if code, ok := returnCodeMap[rc]; ok {
		return code, true
	}
	return 0, false
}
