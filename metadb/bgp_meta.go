package metadb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

type bgpPeerDBObject struct {
	DeviceID       string
	NeighborID     string
	AFIType        string
	Vrf            string
	NeighborStatus string
	UpTime         string
	RemoteAS       uint32
}

type bgpAFIDBObject struct {
	DeviceID          string
	AddressFamilyType string
	Vrf               string
	TotalPrefixes     uint32
	TotalPaths        uint32
}

// PersistsBgpAfiMetadata will save the BGP Address Family metadata in the Telemetry Meta DB
func (p *peppamonMetaDB) PersistsBgpAfiMetadata(bgpAfiMeta []map[string]interface{}, node string) error {

	// Sanitize Data First
	// Ensure Telemetry data from device and DB are in sync
	errSanitize := p.sanitizeBgpAFIs(bgpAfiMeta, node)
	if errSanitize != nil {
		logging.PeppaMonLog("error",
			"Failed to sanitize bgp_afi_meta for node %v : %v", node, errSanitize)
	}

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO bgp_afi_meta
  								  (device_id, timestamps, afi_type, vrf_name, 
                                  total_prefixes, total_paths)
                                  VALUES ($1, $2, $3, $4, $5, $6)
								  ON CONFLICT (device_id, afi_type, vrf_name)
								  DO UPDATE SET
								  total_prefixes = EXCLUDED.total_prefixes,
								  total_paths = EXCLUDED.total_paths,
							      timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	b := &pgx.Batch{}

	for _, cp := range bgpAfiMeta {

		b.Queue(sqlQuery,

			cp["node_id"],
			cp["timestamps"],
			cp["bgp_address_family_type"],
			cp["bgp_address_family_vrf"],
			cp["bgp_afi_total_prefixes"],
			cp["bgp_afi_total_paths"],
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

func (p *peppamonMetaDB) PersistsBgpPeersMetadata(bgpPeers []map[string]interface{}, node string) error {

	// Sanitize Data First
	// Ensure Telemetry data from device and DB are in sync
	errSanitize := p.sanitizeBgpPeers(bgpPeers, node)
	if errSanitize != nil {
		logging.PeppaMonLog("error",
			"Failed to sanitize bgp_peers_meta for node %v : %v", node, errSanitize)
	}

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO bgp_neighbors_meta
  								  (device_id, neighbor_id, address_family_type, timestamps, 
                                  address_family_vrf, neighbor_status, uptime, remote_as)
                                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
								  ON CONFLICT (device_id, neighbor_id, address_family_type, address_family_vrf)
								  DO UPDATE SET
								  neighbor_status = EXCLUDED.neighbor_status,
						          uptime = EXCLUDED.uptime,
								  remote_as = EXCLUDED.remote_as,
							      timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	b := &pgx.Batch{}

	for _, cp := range bgpPeers {

		b.Queue(sqlQuery,

			cp["node_id"],
			cp["neighbor_id"],
			cp["address_family_type"],
			cp["timestamps"],
			cp["address_family_vrf"],
			cp["neighbor_status"],
			cp["neighbor_uptime"],
			cp["neighbor_remote_as"],
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

func (p *peppamonMetaDB) fetchAllBgpPeers(node string) ([]bgpPeerDBObject, error) {

	var bgpPeersSlice []bgpPeerDBObject

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `SELECT device_id, address_family_type, address_family_vrf, neighbor_id,
				      remote_as, neighbor_status, uptime 
				      FROM bgp_neighbors_meta
                      WHERE device_id = $1`

	defer cancelQuery()

	rows, err := p.db.Query(ctxTimeout, sqlQuery, node)

	if err != nil {

		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		peer := bgpPeerDBObject{}

		err = rows.Scan(
			&peer.DeviceID,
			&peer.AFIType,
			&peer.Vrf,
			&peer.NeighborID,
			&peer.RemoteAS,
			&peer.NeighborStatus,
			&peer.UpTime,
		)

		if err != nil {

			return nil, err
		}
		bgpPeersSlice = append(bgpPeersSlice, peer)
	}
	err = rows.Err()
	if err != nil {

		return nil, err
	}

	return bgpPeersSlice, nil

}

func (p *peppamonMetaDB) fetchAllBgpAFI(node string) ([]bgpAFIDBObject, error) {

	var bgpAFIsSlice []bgpAFIDBObject

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `SELECT device_id, afi_type, vrf_name, total_prefixes, total_paths
				      FROM bgp_afi_meta
                      WHERE device_id = $1`

	defer cancelQuery()

	rows, err := p.db.Query(ctxTimeout, sqlQuery, node)

	if err != nil {

		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		afi := bgpAFIDBObject{}

		err = rows.Scan(
			&afi.DeviceID,
			&afi.AddressFamilyType,
			&afi.Vrf,
			&afi.TotalPrefixes,
			&afi.TotalPaths,
		)

		if err != nil {

			return nil, err
		}
		bgpAFIsSlice = append(bgpAFIsSlice, afi)
	}
	err = rows.Err()
	if err != nil {

		return nil, err
	}

	return bgpAFIsSlice, nil

}

func (p *peppamonMetaDB) deleteBgpPeers(dev, peer, afiType, vrf string) error {
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `DELETE FROM bgp_neighbors_meta
					  WHERE device_id = $1 
				      AND neighbor_id = $2
				      AND address_family_type = $3
					  AND address_family_vrf = $4
				     `

	defer cancelQuery()

	cTag, err := p.db.Exec(ctxTimeout, sqlQuery, dev, peer, afiType, vrf)

	if err != nil {

		return err
	}

	if cTag.RowsAffected() == 0 {

		return fmt.Errorf("failed to sanitize BGP peers %v on device %v", peer, dev)
	}

	return nil
}

func (p *peppamonMetaDB) deleteBgpAFI(dev, afiType, vrf string) error {

	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	const sqlQuery = `DELETE FROM bgp_afi_meta
					  WHERE device_id = $1 
				      AND afi_type = $2
				      AND vrf_name = $3
				     `

	defer cancelQuery()

	cTag, err := p.db.Exec(ctxTimeout, sqlQuery, dev, afiType, vrf)

	if err != nil {

		return err
	}

	if cTag.RowsAffected() == 0 {

		return fmt.Errorf("failed to sanitize BGP AFI %v on device %v", afiType, dev)
	}

	return nil
}

func (p *peppamonMetaDB) sanitizeBgpPeers(bgpPeers []map[string]interface{}, node string) error {

	allDBBgpPeers, err := p.fetchAllBgpPeers(node)

	if err != nil {
		return err
	}

	var foundpeersIndex []int

	// Loop through DB BGP peers and add their indexes for those found
	for _, devicePeer := range bgpPeers {
		for idx, dbPeer := range allDBBgpPeers {

			// If we found a match, continue to next iteration
			if devicePeer["neighbor_id"] == dbPeer.NeighborID &&
				devicePeer["address_family_type"] == dbPeer.AFIType &&
				devicePeer["address_family_vrf"] == dbPeer.Vrf {

				foundpeersIndex = append(foundpeersIndex, idx)
			}
		}
	}

	// Delete Peers from DB not part of the device anymore
	for idx, dbPeer := range allDBBgpPeers {

		if !binarySearchSanitizeDB(foundpeersIndex, idx) {
			err := p.deleteBgpPeers(dbPeer.DeviceID, dbPeer.NeighborID, dbPeer.AFIType, dbPeer.Vrf)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *peppamonMetaDB) sanitizeBgpAFIs(bgpPeers []map[string]interface{}, node string) error {

	allDBBgpAfis, err := p.fetchAllBgpAFI(node)

	if err != nil {
		return err
	}

	var foundAfisIndex []int

	// Loop through DB BGP AFI's and add their indexes for those found
	for _, deviceAfi := range bgpPeers {
		for idx, dbAfi := range allDBBgpAfis {

			// If we found a match, continue to next iteration
			if deviceAfi["afi_type"] == dbAfi.AddressFamilyType &&
				deviceAfi["vrf_name"] == dbAfi.Vrf {

				foundAfisIndex = append(foundAfisIndex, idx)
			}
		}
	}

	// Delete AFI's from DB not part of the device anymore
	for idx, dbAfi := range allDBBgpAfis {

		if !binarySearchSanitizeDB(foundAfisIndex, idx) {
			err := p.deleteBgpAFI(dbAfi.DeviceID, dbAfi.AddressFamilyType, dbAfi.Vrf)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
