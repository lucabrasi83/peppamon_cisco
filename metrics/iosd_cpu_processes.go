package metrics

import (
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16111/Cisco-IOS-XE-process-cpu-oper.yang

	// Process-ID of the process
	yangCPUProcPID = "pid"

	// The name of the process
	yangCPUProcName = "name"

	// Average Run-time of this process (uSec)
	yangCPUProcAvgRunTime = "avg-run-time"

	// Busy percentage in last 5-seconds
	yangCPUProcBusy5Sec = "five-seconds"

	// Busy percentage in last 1 minute
	yangCPUProcBusy1Min = "one-minute"

	// Busy percentage in last 5 minutes
	yangCPUProcBusy5Min = "five-minutes"
)

func parseCPUProcMeta(fields []*telemetry.TelemetryField, node string, t time.Time) map[string]interface{} {

	ProcCPUObj := make(map[string]interface{})

	timestamps := t.Unix()

	for _, field := range fields {

		switch field.GetName() {

		case yangCPUProcPID:
			ProcCPUObj["node_id"] = node
			ProcCPUObj["timestamps"] = timestamps
			ProcCPUObj["pid"] = extractGPBKVNativeTypeFromOneof(field, true)

		case yangCPUProcName:
			ProcCPUObj["proc_name"] = extractGPBKVNativeTypeFromOneof(field, false)

		case yangCPUProcAvgRunTime:
			ProcCPUObj["proc_avg_runtime"] = extractGPBKVNativeTypeFromOneof(field, true)

		case yangCPUProcBusy5Sec:
			ProcCPUObj["cpu_proc_busy_avg_5_sec"] = extractGPBKVNativeTypeFromOneof(field, true)

		case yangCPUProcBusy1Min:
			ProcCPUObj["cpu_proc_busy_avg_1_min"] = extractGPBKVNativeTypeFromOneof(field, true)

		case yangCPUProcBusy5Min:
			ProcCPUObj["cpu_proc_busy_avg_5_min"] = extractGPBKVNativeTypeFromOneof(field, true)
		}

	}
	return ProcCPUObj
}

func recordCPUProcMeta(p []map[string]interface{}, node string) {

	err := metadb.DBInstance.PersistsCPUProcMetadata(p)

	if err != nil {
		logging.PeppaMonLog(
			"error",
			"Failed to insert CPU processes metadata for node %v: %v", node, err)
	}
}
