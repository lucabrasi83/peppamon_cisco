package metadb

import (
	"context"
	"fmt"
	"net"

	"github.com/jackc/pgx/v4"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

// PersistsInterfaceMetadata will update the Telemetry Metadata database with interfaces attributes
func (p *peppamonMetaDB) PersistsInterfaceMetadata(ifMeta []map[string]interface{}, node string) error {

	// Sanitize Data First
	// Ensure Telemetry data from device and DB are in sync
	errSanitize := p.sanitizeInterfaces(ifMeta, node)
	if errSanitize != nil {
		logging.PeppaMonLog("error",
			"Failed to sanitize interfaces_meta for node %v : %v", node, errSanitize)
	}

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO interface_meta
  								  (device_id, timestamps, interface_name, description, 
                                  ipv4_address, admin_status, oper_status,
							      speed, mtu, physical_address, 
                                  vrf_attached, last_status_change)
                                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
								  ON CONFLICT (device_id, interface_name)
								  DO UPDATE SET
								  description = EXCLUDED.description,
								  ipv4_address = EXCLUDED.ipv4_address,
								  admin_status = EXCLUDED.admin_status,
							      oper_status = EXCLUDED.oper_status,
								  speed = EXCLUDED.speed,
								  mtu = EXCLUDED.mtu,
							      physical_address = EXCLUDED.physical_address,
								  vrf_attached = EXCLUDED.vrf_attached,
								  last_status_change = EXCLUDED.last_status_change,
                                  timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	b := &pgx.Batch{}

	for _, cp := range ifMeta {

		b.Queue(sqlQuery,

			cp["node_id"],
			cp["timestamps"],
			cp["if_name"],
			cp["description"],
			convertStrToIPv4(cp["ipv4_address"].(string), cp["ipv4_subnet_mask"].(string)),
			cp["admin_status"],
			cp["oper_status"],
			cp["speed"],
			cp["mtu"],
			cp["physical_address"],
			cp["vrf"],
			cp["last_change"],
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

func (p *peppamonMetaDB) fetchAllInterfaces(node string) ([]string, error) {

	var interfacesSlice []string

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `SELECT interface_name
				      FROM interface_meta
                      WHERE device_id = $1`

	defer cancelQuery()

	rows, err := p.db.Query(ctxTimeout, sqlQuery, node)

	if err != nil {

		return nil, err
	}

	defer rows.Close()

	for rows.Next() {

		var ifName string

		err = rows.Scan(
			&ifName,
		)

		if err != nil {

			return nil, err
		}
		interfacesSlice = append(interfacesSlice, ifName)
	}
	err = rows.Err()
	if err != nil {

		return nil, err
	}

	return interfacesSlice, nil

}

func (p *peppamonMetaDB) deleteInterfaces(dev, ifName string) error {
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `DELETE FROM interface_meta
					  WHERE device_id = $1 
				      AND interface_name = $2
				     `

	defer cancelQuery()

	cTag, err := p.db.Exec(ctxTimeout, sqlQuery, dev, ifName)

	if err != nil {

		return err
	}

	if cTag.RowsAffected() == 0 {

		return fmt.Errorf("failed to sanitize Interface %v on device %v", ifName, dev)
	}

	return nil
}

func (p *peppamonMetaDB) sanitizeInterfaces(devInterfaces []map[string]interface{}, node string) error {

	allDBInterfaces, err := p.fetchAllInterfaces(node)

	if err != nil {
		return err
	}

	var foundInterfacesIndex []int

	// Loop through DB Device Interfaces and add their indexes for those found
	for _, deviceInterface := range devInterfaces {
		for idx, dbInterface := range allDBInterfaces {

			// If we found a match, continue to next iteration
			if deviceInterface["if_name"] == dbInterface {

				foundInterfacesIndex = append(foundInterfacesIndex, idx)
			}
		}
	}

	// Delete Interfaces from DB not part of the device anymore
	for idx, dbInterface := range allDBInterfaces {

		if !binarySearchSanitizeDB(foundInterfacesIndex, idx) {
			err := p.deleteInterfaces(node, dbInterface)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func convertStrToIPv4(ip string, mask string) net.IPNet {

	maskLength := net.IPMask(net.ParseIP(mask).To4())

	if maskLength == nil {
		maskLength = net.IPMask(net.ParseIP("0.0.0.0").To4())
	}

	ipAddress := net.ParseIP(ip).To4()

	if ipAddress == nil {
		ipAddress = net.ParseIP("0.0.0.0").To4()
	}

	netObj := net.IPNet{
		IP:   ipAddress,
		Mask: maskLength,
	}

	return netObj

}
