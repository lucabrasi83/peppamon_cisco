package metadb

import (
	"context"
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
	_, err := p.db.PrepareEx(ctxTimeout, "mem_proc_meta_query", sqlQuery, nil)

	if err != nil {
		return err
	}

	b := p.db.BeginBatch()

	for _, cp := range memProc {

		b.Queue("mem_proc_meta_query",
			[]interface{}{
				cp["node_id"],
				cp["timestamps"],
				cp["process_name"],
				cp["pid"],
				cp["allocated_memory"],
				cp["freed_memory"],
				cp["holding_memory"],
			},
			nil, nil)
	}

	// Send Batch SQL Query
	errSendBatch := b.Send(ctxTimeout, nil)

	if errSendBatch != nil {
		return errSendBatch
	}

	// Execute Batch SQL Query
	errExecBatch := b.Close()
	if errExecBatch != nil {

		return errExecBatch
	}

	return nil
}
