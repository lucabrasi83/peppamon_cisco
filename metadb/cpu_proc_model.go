package metadb

import (
	"context"
)

// PersistsInterfaceMetadata will update the Telemetry Metadata database with interfaces attributes
func (p *peppamonMetaDB) PersistsCPUProcMetadata(cpuProc []map[string]interface{}) error {

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	// SQL Query to insert VA Scan Result per device
	const sqlQuery = `INSERT INTO cpu_processes_meta
  								  (device_id, timestamps, cpu_process_name, cpu_process_pid, 
                                  cpu_proc_avg_runtime, cpu_proc_busy_avg_5_sec, 
								  cpu_proc_busy_avg_1_min, cpu_proc_busy_avg_5_min)
                                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
								  ON CONFLICT (device_id, cpu_process_name)
								  DO UPDATE SET
								  cpu_process_pid = EXCLUDED.cpu_process_pid,
								  cpu_proc_avg_runtime = EXCLUDED.cpu_proc_avg_runtime,
								  cpu_proc_busy_avg_5_sec = EXCLUDED.cpu_proc_busy_avg_5_sec,
							      cpu_proc_busy_avg_1_min = EXCLUDED.cpu_proc_busy_avg_1_min,
								  cpu_proc_busy_avg_5_min = EXCLUDED.cpu_proc_busy_avg_5_min,
                                  timestamps = EXCLUDED.timestamps
								 `

	defer cancelQuery()

	// Prepare SQL Statement in DB for Batch
	_, err := p.db.PrepareEx(ctxTimeout, "cpu_proc_meta_query", sqlQuery, nil)

	if err != nil {
		return err
	}

	b := p.db.BeginBatch()

	for _, cp := range cpuProc {

		b.Queue("cpu_proc_meta_query",
			[]interface{}{
				cp["node_id"],
				cp["timestamps"],
				cp["proc_name"],
				cp["pid"],
				cp["proc_avg_runtime"],
				cp["cpu_proc_busy_avg_5_sec"],
				cp["cpu_proc_busy_avg_1_min"],
				cp["cpu_proc_busy_avg_5_min"],
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
