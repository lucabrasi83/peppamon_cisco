package metrics

import (
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16121/Cisco-IOS-XE-license.yang
	DeviceLicenseInfoYANGEncodingPath = "Cisco-IOS-XE-native:native/license"

	// Device PID
	yangDeviceLicenseUDI = "udi"

	// Device PID
	yangDeviceLicensePID = "pid"

	// Device Serial Number
	yangDeviceLicenseSN = "sn"

	// Device Boot
	yangDeviceLicenseBoot = "boot"

	// Device Boot Level License
	yangDeviceLicenseBootLevel = "level"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     DeviceLicenseInfoYANGEncodingPath,
		RecordMetricFunc: parseDeviceLicenseInfo,
	})
}

func parseDeviceLicenseInfo(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	licObj := map[string]interface{}{
		"node":                     node,
		"timestamps":               t.Unix(),
		yangDeviceLicensePID:       "N/A",
		yangDeviceLicenseSN:        "N/A",
		yangDeviceLicenseBootLevel: "N/A",
	}

	for _, p := range msg.DataGpbkv[0].Fields[1].Fields {
		switch p.GetName() {
		case yangDeviceLicenseUDI:
			for _, f := range p.Fields {
				switch f.GetName() {
				case yangDeviceLicensePID:
					licObj[yangDeviceLicensePID] = extractGPBKVNativeTypeFromOneof(f, false)
				case yangDeviceLicenseSN:
					licObj[yangDeviceLicenseSN] = extractGPBKVNativeTypeFromOneof(f, false)
				}
			}
		case yangDeviceLicenseBoot:
			licBootLevel := p.Fields[0].Fields[0].GetName()
			licObj[yangDeviceLicenseBootLevel] = licBootLevel
		}
	}
	go func() {
		err := metadb.DBInstance.PersistsDeviceLicenseData(licObj, node)
		if err != nil {
			logging.PeppaMonLog("error",
				"Failed to insert Device License data into DB: %v for Node %v", err, node)
		}
	}()

}
