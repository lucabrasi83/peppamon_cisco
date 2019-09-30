package metadb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

func (p *peppamonMetaDB) PersistsIPSlaConfigMetadata(ipSLAMeta []map[string]interface{}, node string) error {

	// Sanitize Data First
	// Ensure Telemetry data from device and DB are in sync
	errSanitize := p.sanitizeIPSLA(ipSLAMeta, node)
	if errSanitize != nil {
		logging.PeppaMonLog("error",
			"Failed to sanitize ip_sla_config_meta for node %v : %v", node, errSanitize)
	}

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO ip_sla_config_meta
  								  (device_id, timestamps, entry_id, destination_ip, 
                                  destination_port, source_ip, source_port,
							      vrf, frequency, type, 
                                  dscp, class_of_service, req_data_size, 
						          http_url, http_version, http_proxy, http_dns_server, destination_host)
                                  VALUES 
								  ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
								  ON CONFLICT (device_id, entry_id)
								  DO UPDATE SET
								  destination_ip = EXCLUDED.destination_ip,
								  destination_port = EXCLUDED.destination_port,
								  source_ip = EXCLUDED.source_ip,
							      source_port = EXCLUDED.source_port,
								  vrf = EXCLUDED.vrf,
								  frequency = EXCLUDED.frequency,
							      type = EXCLUDED.type,
								  dscp = EXCLUDED.dscp,
								  class_of_service = EXCLUDED.class_of_service,
                                  req_data_size = EXCLUDED.req_data_size,
                                  http_url = EXCLUDED.http_url,
								  http_proxy = EXCLUDED.http_proxy,
								  http_dns_server = EXCLUDED.http_dns_server,
								  http_version = EXCLUDED.http_version,
					              timestamps = EXCLUDED.timestamps,
						          destination_host = EXCLUDED.destination_host
								 `

	defer cancelQuery()

	b := &pgx.Batch{}

	for _, cp := range ipSLAMeta {

		b.Queue(sqlQuery,

			cp["node_id"].(string),
			cp["timestamps"].(int64),
			cp["sla_number"].(int),
			cp["destination_ip"].(string),
			cp["destination_port"].(int),
			cp["source_ip"].(string),
			cp["source_port"].(int),
			cp["vrf"].(string),
			cp["frequency"].(int),
			cp["sla_type"].(string),
			cp["dscp"].(string),
			cp["class_of_service"].(string),
			cp["req_data_size"].(int),
			cp["http_url"].(string),
			cp["http_version"].(string),
			cp["http_proxy"].(string),
			cp["http_dns_server"].(string),
			cp["destination_host"].(string),
		)
	}

	// Send Batch SQL Query
	r := p.db.SendBatch(ctxTimeout, b)

	// Close Batch at the end of function
	defer func() {
		errCloseBatch := r.Close()
		if errCloseBatch != nil {
			logging.PeppaMonLog("error",
				"Failed to close SQL Batch Job query %s with error %v", sqlQuery, errCloseBatch)
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

func (p *peppamonMetaDB) fetchAllIPSLA(node string) ([]int, error) {

	var ipSLASlice []int

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `SELECT entry_id
				      FROM ip_sla_config_meta
                      WHERE device_id = $1`

	defer cancelQuery()

	rows, err := p.db.Query(ctxTimeout, sqlQuery, node)

	if err != nil {
		return nil, err

	}

	defer rows.Close()

	for rows.Next() {

		var ipSLAEntry int

		err = rows.Scan(&ipSLAEntry)

		if err != nil {
			return nil, err
		}
		ipSLASlice = append(ipSLASlice, ipSLAEntry)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return ipSLASlice, nil

}

func (p *peppamonMetaDB) deleteIPSLA(dev string, ipSLAEntry int) error {
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `DELETE FROM ip_sla_config_meta
					  WHERE device_id = $1 
				      AND entry_id = $2
				     `

	defer cancelQuery()

	cTag, err := p.db.Exec(ctxTimeout, sqlQuery, dev, ipSLAEntry)

	if err != nil {

		return err
	}

	if cTag.RowsAffected() == 0 {

		return fmt.Errorf("failed to sanitize IP SLA %v during deletion on device %v", ipSLAEntry, dev)
	}

	return nil
}

func (p *peppamonMetaDB) sanitizeIPSLA(devIPSLA []map[string]interface{}, node string) error {

	allDBIPSLA, err := p.fetchAllIPSLA(node)

	if err != nil {
		return err
	}

	var foundIPSLAIndex []int

	// Loop through DB IP SLAs and add their indexes for those found
	for _, deviceIPSLA := range devIPSLA {
		for idx, dbIPSLA := range allDBIPSLA {

			// If we found a match, continue to next iteration
			if deviceIPSLA["sla_number"] == dbIPSLA {
				foundIPSLAIndex = append(foundIPSLAIndex, idx)
			}
		}
	}

	// Delete IP SLA from DB not part of the device anymore
	for idx, dbIPSLA := range allDBIPSLA {

		if !binarySearchSanitizeDB(foundIPSLAIndex, idx) {
			err := p.deleteIPSLA(node, dbIPSLA)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
