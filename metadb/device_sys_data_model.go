package metadb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

func (p *peppamonMetaDB) PersistsDeviceSYSData(devSYSData []map[string]interface{}, node string) error {

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO device_sys_data
  								  (device_id, timestamps, last_seen_epoch, boot_time_epoch, sw_version)
                                  VALUES 
								  ($1, $2, $3, $4, $5)
								  ON CONFLICT (device_id)
								  DO UPDATE SET
								  last_seen_epoch = EXCLUDED.last_seen_epoch,
								  boot_time_epoch = EXCLUDED.boot_time_epoch,
								  sw_version = EXCLUDED.sw_version,
					              timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	b := &pgx.Batch{}

	for _, cp := range devSYSData {

		b.Queue(sqlQuery,

			cp["node"].(string),
			cp["timestamps"].(int64),
			cp["current-time"].(int64),
			cp["boot-time"].(int64),
			cp["software-version"].(string),
		)
	}

	// Send Batch SQL Query
	r := p.db.SendBatch(ctxTimeout, b)

	// Close Batch at the end of function
	defer func() {
		errCloseBatch := r.Close()
		if errCloseBatch != nil {
			logging.PeppaMonLog("error",
				fmt.Sprintf("Failed to close SQL Batch Job with error %v", errCloseBatch))
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
