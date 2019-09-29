package metadb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

// PersistsMemProcMetadata will save the processes memory utilization in the Telemetry Meta DB
func (p *peppamonMetaDB) PersistsMemProcMetadata(memProc []map[string]interface{}) error {

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO mem_processes_meta
  								  (device_id, timestamps, mem_process_name, mem_process_pid, 
                                  allocated_memory, freed_memory, holding_memory)
                                  VALUES ($1, $2, $3, $4, $5, $6, $7)
								  ON CONFLICT (device_id, mem_process_name)
								  DO UPDATE SET
								  mem_process_pid = EXCLUDED.mem_process_pid,
								  allocated_memory = EXCLUDED.allocated_memory,
								  freed_memory = EXCLUDED.freed_memory,
							      holding_memory = EXCLUDED.holding_memory,
                                  timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	b := &pgx.Batch{}

	for _, cp := range memProc {

		b.Queue(sqlQuery,

			cp["node_id"],
			cp["timestamps"],
			cp["process_name"],
			cp["pid"],
			cp["allocated_memory"],
			cp["freed_memory"],
			cp["holding_memory"],
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
