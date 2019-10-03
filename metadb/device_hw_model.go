package metadb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

type devHWInfoObj struct {
	DeviceID     string
	HWType       string
	SerialNumber string
	PartNumber   string
}

func (p *peppamonMetaDB) PersistsDeviceHWInventory(devHWInventory []map[string]interface{}, node string) error {

	// Sanitize Data First
	// Ensure Telemetry data from device and DB are in sync
	errSanitize := p.sanitizeDeviceHWInventory(devHWInventory, node)
	if errSanitize != nil {
		logging.PeppaMonLog("error",
			"Failed to sanitize device_hw_info for node %v : %v", node, errSanitize)
	}

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO device_hw_info
  								  (device_id, timestamps, hardware_type, hardware_part_number, hardware_description,
								  hardware_device_name, hardware_field_replaceable, hardware_version, serial_number)
                                  VALUES 
								  ($1, $2, $3, $4, $5, $6, $7, $8, $9)
								  ON CONFLICT (device_id, hardware_type, hardware_part_number, serial_number)
								  DO UPDATE SET
								  hardware_description = EXCLUDED.hardware_description,
								  hardware_device_name = EXCLUDED.hardware_device_name,
								  hardware_field_replaceable = EXCLUDED.hardware_field_replaceable,
							      hardware_version = EXCLUDED.hardware_version,
					              timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	b := &pgx.Batch{}

	for _, cp := range devHWInventory {

		b.Queue(sqlQuery,

			cp["node"].(string),
			cp["timestamps"].(int64),
			cp["hw-type"].(string),
			cp["part-number"].(string),
			cp["hw-description"].(string),
			cp["dev-name"].(string),
			cp["field-replaceable"].(bool),
			cp["version"].(string),
			cp["serial-number"].(string),
		)
	}

	// Send Batch SQL Query
	r := p.db.SendBatch(ctxTimeout, b)

	// Close Batch at the end of function
	defer func() {
		errCloseBatch := r.Close()
		if errCloseBatch != nil {
			logging.PeppaMonLog("error",
				"Failed to close SQL Batch Job with error %v", errCloseBatch)
		}
	}()

	c, errSendBatch := r.Exec()

	if errSendBatch != nil {
		return errSendBatch
	}

	if c.RowsAffected() < 1 {
		return fmt.Errorf("no insertion of row while executing query %v", sqlQuery)
	}

	return nil
}

func (p *peppamonMetaDB) fetchAllDeviceHWInventory(node string) ([]devHWInfoObj, error) {

	var devHWObjSlice []devHWInfoObj

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `SELECT device_id, hardware_type, hardware_part_number, serial_number
				      FROM device_hw_info
                      WHERE device_id = $1`

	defer cancelQuery()

	rows, err := p.db.Query(ctxTimeout, sqlQuery, node)

	if err != nil {
		return nil, err

	}

	defer rows.Close()

	for rows.Next() {

		hwInfo := devHWInfoObj{}

		err = rows.Scan(
			&hwInfo.DeviceID,
			&hwInfo.HWType,
			&hwInfo.PartNumber,
			&hwInfo.SerialNumber,
		)

		if err != nil {
			return nil, err
		}
		devHWObjSlice = append(devHWObjSlice, hwInfo)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return devHWObjSlice, nil

}

func (p *peppamonMetaDB) deleteDeviceHWInventory(dev, hwType, hwPartNum, hwSerialNum string) error {

	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `DELETE FROM device_hw_info
					  WHERE device_id = $1 
				      AND hardware_type = $2
					  AND hardware_part_number = $3
                      AND serial_number = $4
				     `

	defer cancelQuery()

	cTag, err := p.db.Exec(ctxTimeout, sqlQuery, dev, hwType, hwPartNum, hwSerialNum)

	if err != nil {

		return err
	}

	if cTag.RowsAffected() == 0 {

		return fmt.Errorf("failed to sanitize HW Info Deletion %v on device %v", hwType, dev)
	}

	return nil
}

func (p *peppamonMetaDB) sanitizeDeviceHWInventory(devHWInfo []map[string]interface{}, node string) error {

	allDBdevHWInfo, err := p.fetchAllDeviceHWInventory(node)

	if err != nil {
		return err
	}

	var foundHWInfoIndex []int

	// Loop through DB Hardware Inventory and add their indexes for those found
	for _, deviceHWInfo := range devHWInfo {
		for idx, dbHWInfo := range allDBdevHWInfo {

			// If we found a match, continue to next iteration
			if v, ok := deviceHWInfo["hw-type"].(string); ok && v == dbHWInfo.HWType {
				if v, ok := deviceHWInfo["part-number"].(string); ok && v == dbHWInfo.PartNumber {
					if v, ok := deviceHWInfo["serial-number"].(string); ok && v == dbHWInfo.SerialNumber {
						foundHWInfoIndex = append(foundHWInfoIndex, idx)
					}

				}

			}
		}
	}

	// Delete IP SLA from DB not part of the device anymore
	for idx, dbHWInfo := range allDBdevHWInfo {

		if !binarySearchSanitizeDB(foundHWInfoIndex, idx) {
			err := p.deleteDeviceHWInventory(node, dbHWInfo.HWType, dbHWInfo.PartNumber, dbHWInfo.SerialNumber)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
