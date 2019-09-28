package metadb

import (
	"context"
	"fmt"
)

func (p *peppamonMetaDB) PersistsDeviceLicenseData(devLicenseData map[string]interface{}, node string) error {

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO device_license_meta
  								  (device_id, timestamps, product_id, 
						 		   serial_number, boot_license)
                                  VALUES 
								  ($1, $2, $3, $4, $5)
								  ON CONFLICT (device_id)
								  DO UPDATE SET
								  product_id = EXCLUDED.product_id,
								  serial_number = EXCLUDED.serial_number,
								  boot_license = EXCLUDED.boot_license,
					              timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	c, err := p.db.Exec(ctxTimeout,
		sqlQuery, devLicenseData["node"].(string),
		devLicenseData["timestamps"].(int64),
		devLicenseData["pid"].(string),
		devLicenseData["sn"].(string),
		devLicenseData["level"].(string),
	)

	if err != nil {
		return err
	}

	if c.RowsAffected() < 1 {
		return fmt.Errorf("no insertion of row while executing query %v", sqlQuery)
	}

	return nil
}
