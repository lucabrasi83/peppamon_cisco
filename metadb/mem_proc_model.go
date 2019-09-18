package metadb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
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

	// Prepare SQL Statement in DB for Batch
	//_, err := p.db.PrepareEx(ctxTimeout, "mem_proc_meta_query", sqlQuery, nil)
	//
	//if err != nil {
	//	return err
	//}
	//
	//b := p.db.BeginBatch()
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
	// errSendBatch := b.Send(ctxTimeout, nil)
	r := p.db.SendBatch(ctxTimeout, b)
	c, errSendBatch := r.Exec()

	if errSendBatch != nil {
		return errSendBatch
	}

	if c.RowsAffected() < 1 {
		return fmt.Errorf("no insertion of row while executing query %v", sqlQuery)
	}

	// Execute Batch SQL Query
	errExecBatch := r.Close()
	if errExecBatch != nil {

		return errExecBatch
	}

	return nil
}
