package metrics

import (
	"fmt"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-process-memory-oper.yang
	ProcMemoryYANGEncodingPath = "Cisco-IOS-XE-process-memory-oper:memory-usage-processes/memory-usage-process"

	// Process-ID of the process
	yangIOSdMemProcPID = "pid"

	// The name of the process
	yangIOSdMemProcName = "name"

	// Total memory allocated to this process (bytes)
	yangIOSdMemProcAllocated = "allocated-memory"

	// Total memory freed by this process (bytes)
	yangIOSdMemProcFreed = "freed-memory"

	// Total memory currently held by this process (bytes)
	yangIOSdMemProcHolding = "holding-memory"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     ProcMemoryYANGEncodingPath,
		RecordMetricFunc: parseMemoryProcMeta,
	})
}

func parseMemoryProcMeta(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time) {

	var ProcMemObjSlice []map[string]interface{}

	timestamps := time.Now().Unix()

	for _, p := range msg.DataGpbkv {

		ProcMemObj := make(map[string]interface{})

		for _, procMeta := range p.Fields[0].Fields {
			switch procMeta.GetName() {
			case yangIOSdMemProcPID:

				ProcMemObj["node_id"] = msg.GetNodeIdStr()
				ProcMemObj["timestamps"] = timestamps

				ProcMemObj["pid"] = procMeta.GetUint32Value()

			case yangIOSdMemProcName:
				ProcMemObj["process_name"] = procMeta.GetStringValue()
			}
		}

		for _, procMemUsage := range p.Fields[1].Fields {

			switch procMemUsage.GetName() {

			case yangIOSdMemProcAllocated:
				ProcMemObj["allocated_memory"] = procMemUsage.GetUint64Value()

			case yangIOSdMemProcFreed:
				ProcMemObj["freed_memory"] = procMemUsage.GetUint64Value()

			case yangIOSdMemProcHolding:
				ProcMemObj["holding_memory"] = procMemUsage.GetUint64Value()

			}
		}
		ProcMemObjSlice = append(ProcMemObjSlice, ProcMemObj)
	}
	err := metadb.DBInstance.PersistsMemProcMetadata(ProcMemObjSlice)

	if err != nil {
		logging.PeppaMonLog(
			"error",
			fmt.Sprintf("Failed to insert Memory processes metadata for node %v: %v", msg.GetNodeIdStr(), err))
	}
}
