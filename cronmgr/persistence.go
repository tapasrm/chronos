package cronmgr

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// db schema:
// CREATE TABLE IF NOT EXISTS jobs (
//   id TEXT PRIMARY KEY,
//   name TEXT,
//   type TEXT,
//   schedule TEXT,
//   schedule_desc TEXT,
//   enabled INTEGER,
//   config_json TEXT,
//   last_run INTEGER,
//   next_run INTEGER
// );

func openDB(path string) (*sql.DB, error) {
	// github.com/mattn/go-sqlite3 registers the driver name "sqlite3"
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// Create table if not exists
	schema := `CREATE TABLE IF NOT EXISTS jobs (
        id TEXT PRIMARY KEY,
        name TEXT,
        type TEXT,
        schedule TEXT,
        schedule_desc TEXT,
        enabled INTEGER,
        config_json TEXT,
        last_run INTEGER,
        next_run INTEGER
    );`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// SaveAllJobsToDB writes all jobs currently in memory to the SQLite DB (upsert semantics)
func (cm *CronManager) SaveAllJobsToDB(path string) error {
	db, err := openDB(path)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO jobs(id,name,type,schedule,schedule_desc,enabled,config_json,last_run,next_run)
        VALUES(?,?,?,?,?,?,?,?,?)
        ON CONFLICT(id) DO UPDATE SET
          name=excluded.name,
          type=excluded.type,
          schedule=excluded.schedule,
          schedule_desc=excluded.schedule_desc,
          enabled=excluded.enabled,
          config_json=excluded.config_json,
          last_run=excluded.last_run,
          next_run=excluded.next_run`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, job := range cm.jobs {
		cfg, _ := json.Marshal(job.Config)
		var lastRunUnix, nextRunUnix any
		if job.LastRun != nil {
			lastRunUnix = job.LastRun.Unix()
		} else {
			lastRunUnix = nil
		}
		if job.NextRun != nil {
			nextRunUnix = job.NextRun.Unix()
		} else {
			nextRunUnix = nil
		}

		if _, err := stmt.Exec(job.ID, job.Name, string(job.Type), job.Schedule, job.ScheduleDesc, boolToInt(job.Enabled), string(cfg), lastRunUnix, nextRunUnix); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// LoadJobsFromDB reads jobs from sqlite and adds them into the manager (does not start scheduling)
func (cm *CronManager) LoadJobsFromDB(path string) error {
	db, err := openDB(path)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id,name,type,schedule,schedule_desc,enabled,config_json,last_run,next_run FROM jobs`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var loadErrors []error
	var loadedCount int

	for rows.Next() {
		var id, name, typ, schedule, scheduleDesc, configJSON sql.NullString
		var enabled sql.NullInt64
		var lastRun, nextRun sql.NullInt64

		if err := rows.Scan(&id, &name, &typ, &schedule, &scheduleDesc, &enabled, &configJSON, &lastRun, &nextRun); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to scan row: %w", err))
			continue // Continue loading other rows
		}

		j := &Job{
			ID:           id.String,
			Name:         name.String,
			Type:         JobType(typ.String),
			Schedule:     schedule.String,
			ScheduleDesc: scheduleDesc.String,
			Enabled:      intToBool(int(enabled.Int64)),
			Config:       map[string]any{},
		}

		if configJSON.Valid && configJSON.String != "" {
			var cfg map[string]any
			if err := json.Unmarshal([]byte(configJSON.String), &cfg); err == nil {
				j.Config = cfg
			}
		}

		if lastRun.Valid {
			t := time.Unix(lastRun.Int64, 0)
			j.LastRun = &t
		}
		if nextRun.Valid {
			t := time.Unix(nextRun.Int64, 0)
			j.NextRun = &t
		}

		// Add job to manager (this will re-schedule if enabled)
		// Use AddJob which includes validation and scheduling
		if err := cm.AddJob(j); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to add job %s: %w", j.ID, err))
			continue // Continue loading other jobs
		}
		loadedCount++
	}

	if err := rows.Err(); err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("row iteration error: %w", err))
	}

	// Log summary
	if len(loadErrors) > 0 {
		slog.Warn("Some jobs failed to load", "loaded", loadedCount, "errors", len(loadErrors))
		for _, err := range loadErrors {
			slog.Warn("Load error", "error", err)
		}
		// Return a combined error but don't fail completely
		return fmt.Errorf("loaded %d jobs with %d errors: %w", loadedCount, len(loadErrors), loadErrors[0])
	}

	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}
