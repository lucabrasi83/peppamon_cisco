package metrics

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
)

const (
	// The YANG Schema path we're accepting stream
	// https://github.com/YangModels/yang/blob/master/vendor/cisco/xe/16121/Cisco-IOS-XE-device-hardware-oper.yang
	HWInventoryYANGEncodingPath = "Cisco-IOS-XE-device-hardware-oper:device-hardware-data"

	// Device Inventory
	yangHardwareDeviceInventory = "device-inventory"

	// Hardware Type
	yangHardwareType = "hw-type"

	// Hardware Version
	yangHardwareVersion = "version"

	// Hardware Part Number
	yangHardwarePartNumber = "part-number"

	// Hardware Serial Number
	yangHardwareSerialNumber = "serial-number"

	// Hardware Description
	yangHardwareDescription = "hw-description"

	// Hardware Device Name
	yangHardwareDeviceName = "dev-name"

	// Hardware Field Replaceable
	yangHardwareFieldReplaceable = "field-replaceable"

	// Hardware Device System Data
	yangHardwareDeviceSystemData = "device-system-data"

	// Hardware Device Software Version
	yangHardwareDeviceSoftwareVersion = "software-version"

	// Hardware Device Boot Time
	yangHardwareDeviceBootTime = "boot-time"

	// Hardware Device Last Seen
	yangHardwareDeviceLastSeenTime = "current-time"
)

func init() {
	CiscoMetricRegistrar = append(CiscoMetricRegistrar, CiscoTelemetryMetric{
		EncodingPath:     HWInventoryYANGEncodingPath,
		RecordMetricFunc: parseDeviceHardwareMsg,
	})
}

func parseDeviceHardwareMsg(msg *telemetry.Telemetry, dm *DeviceGroupedMetrics, t time.Time, node string) {

	hwObjSlice := make([]map[string]interface{}, 0)
	sysObjSlice := make([]map[string]interface{}, 0)

	for _, p := range msg.DataGpbkv[0].Fields {

		// Channel to write Hardware Info
		chHW := make(chan map[string]interface{})

		// Channel to write System Info
		chSYS := make(chan map[string]interface{})

		// Goroutine to read values in channel

		// Wait Group for goroutine reading from channel
		wg := sync.WaitGroup{}
		wg.Add(2)

		// Go Routine to read on Hardware Info channel
		go func() {
			defer wg.Done()

			for c := range chHW {
				hwObjSlice = append(hwObjSlice, c)
			}

		}()

		// Go Routine to read on System Data channel
		go func() {
			defer wg.Done()

			for c := range chSYS {
				sysObjSlice = append(sysObjSlice, c)
			}

		}()

		// Launch recursive function to parse Telemetry PB message and capture desired info
		recursiveHWInfo(p, t, node, chHW, chSYS)

		// Notify the channel is closed as soon as recursiveHWInfo is done
		close(chHW)

		// Notify the channel is closed as soon as recursiveHWInfo is done
		close(chSYS)

		// Block here until we're done reading values from channel and adding to the hwObjSlice
		wg.Wait()

	}

	if len(hwObjSlice) > 0 {
		go func() {
			err := metadb.DBInstance.PersistsDeviceHWInventory(hwObjSlice, node)
			if err != nil {
				logging.PeppaMonLog("error",
					fmt.Sprintf("Failed to insert Device Hardware inventory data into DB: %v for Node %v", err, node))
			}
		}()
	}

	if len(sysObjSlice) > 0 {
		go func() {
			err := metadb.DBInstance.PersistsDeviceSYSData(sysObjSlice, node)
			if err != nil {
				logging.PeppaMonLog("error",
					fmt.Sprintf("Failed to insert Device System data into DB: %v for Node %v", err, node))
			}
		}()
	}

}

// recursiveHWInfo will perform recursion within the Telemetry message and build the stack
// until we found the Device Hardware key/value content
func recursiveHWInfo(m *telemetry.TelemetryField, t time.Time, node string,
	chHardware chan map[string]interface{}, chSysData chan map[string]interface{}) {

	for _, field := range m.Fields {

		if field.GetName() == yangHardwareDeviceInventory {

			hwObj := map[string]interface{}{
				"node":                       node,
				"timestamps":                 t.Unix(),
				yangHardwareType:             "N/A",
				yangHardwareVersion:          "N/A",
				yangHardwarePartNumber:       "N/A",
				yangHardwareSerialNumber:     "N/A",
				yangHardwareDescription:      "N/A",
				yangHardwareDeviceName:       "N/A",
				yangHardwareFieldReplaceable: false,
			}

			for _, hwField := range field.Fields {

				if _, ok := hwObj[hwField.GetName()]; ok {

					if hwField.GetName() == yangHardwareFieldReplaceable {
						hwObj[hwField.GetName()] = extractGPBKVNativeTypeFromOneof(hwField, false)
					}

					// If no value specified in Telemetry message for strings, use default N/A
					if val, ok := extractGPBKVNativeTypeFromOneof(hwField, false).(string); ok && val != "" {
						hwObj[hwField.GetName()] = val
					}

				}

			}
			chHardware <- hwObj
		} else if field.GetName() == yangHardwareDeviceSystemData {

			sysObj := map[string]interface{}{
				"node":                            node,
				"timestamps":                      t.Unix(),
				yangHardwareDeviceLastSeenTime:    0,
				yangHardwareDeviceSoftwareVersion: "N/A",
				yangHardwareDeviceBootTime:        0,
			}

			for _, hwField := range field.Fields {

				switch hwField.GetName() {
				case yangHardwareDeviceSoftwareVersion:
					val := matchRegexpIOSXEVersion(extractGPBKVNativeTypeFromOneof(hwField, false).(string))
					sysObj[hwField.GetName()] = val

				case yangHardwareDeviceLastSeenTime:
					val := extractGPBKVNativeTypeFromOneof(hwField, false).(string)
					timeObj, err := time.Parse(time.RFC3339, val)

					if err != nil {
						logging.PeppaMonLog("error",
							fmt.Sprintf("Failed to convert yangHardwareDeviceLastSeenTime %v error %v", val, err))
					} else {
						sysObj[hwField.GetName()] = timeObj.UTC().Unix()
					}

				case yangHardwareDeviceBootTime:
					val := extractGPBKVNativeTypeFromOneof(hwField, false).(string)
					timeObj, err := time.Parse(time.RFC3339, val)

					if err != nil {
						logging.PeppaMonLog("error",
							fmt.Sprintf("Failed to convert yangHardwareDeviceBootTime %v error %v", val, err))
					} else {
						sysObj[hwField.GetName()] = timeObj.UTC().Unix()
					}

				}
			}
			chSysData <- sysObj

		}

		// Recursive Function in the YANG model until we get the fields we're looking for
		recursiveHWInfo(field, t, node, chHardware, chSysData)
	}
}

// matchRegexpIOSXEVersion is a convenience function that converts the Cisco 'show version'
// into the actual IOS-XE version
func matchRegexpIOSXEVersion(v string) string {
	r, err := regexp.Compile(`Version (.*?),`)

	if err != nil {
		logging.PeppaMonLog("info",
			fmt.Sprintf("Failed to get IOS-XE Version information with string %v and error %v", v, err))

		return "N/A"
	}

	m := r.FindSubmatch([]byte(v))

	// We expect a byte slice of 2 elements if the Regex matches and return the last element as the matching subgroup
	if len(m) == 2 {
		return string(m[1])
	}
	return v
}
