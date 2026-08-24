// Package store 负责领域实体的 SQLite 持久化（modernc.org/sqlite，WAL 模式）。
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Open 打开（必要时创建）SQLite 数据库，启用 WAL 与超时，并执行迁移。
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS experiments (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			magnet_sn TEXT NOT NULL,
			topology_id TEXT NOT NULL,
			operator TEXT NOT NULL,
			sample_rate REAL NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			sealed_at TEXT,
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS topologies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			raw_json TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS topology_segments (
			id TEXT PRIMARY KEY,
			topology_id TEXT NOT NULL,
			label TEXT NOT NULL,
			position REAL NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS topology_edges (
			id TEXT PRIMARY KEY,
			topology_id TEXT NOT NULL,
			from_id TEXT NOT NULL,
			to_id TEXT NOT NULL,
			distance REAL NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY,
			experiment_id TEXT NOT NULL,
			channel_index INTEGER NOT NULL,
			label TEXT NOT NULL,
			kind TEXT NOT NULL,
			segment_id TEXT NOT NULL,
			status TEXT NOT NULL,
			position REAL NOT NULL,
			delay_ns REAL NOT NULL,
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS waveforms (
			id TEXT PRIMARY KEY,
			experiment_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			sample_rate REAL NOT NULL,
			fingerprint TEXT NOT NULL UNIQUE,
			calibrated INTEGER NOT NULL,
			delay_ns REAL NOT NULL,
			n_points INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS waveform_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			waveform_id TEXT NOT NULL,
			idx INTEGER NOT NULL,
			raw_t REAL NOT NULL,
			v REAL NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_wsamples_wid ON waveform_samples(waveform_id, idx)`,
		`CREATE TABLE IF NOT EXISTS anomalies (
			id TEXT PRIMARY KEY,
			experiment_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			sample_start INTEGER NOT NULL,
			sample_end INTEGER NOT NULL,
			detected_at REAL NOT NULL,
			type TEXT NOT NULL,
			confidence REAL NOT NULL,
			severity REAL NOT NULL,
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS analyses (
			id TEXT PRIMARY KEY,
			experiment_id TEXT NOT NULL,
			status TEXT NOT NULL,
			onset_channel_id TEXT NOT NULL,
			onset_time REAL NOT NULL,
			propagation_speed REAL NOT NULL,
			affected_segments TEXT NOT NULL,
			protection_window_ms REAL NOT NULL,
			breach INTEGER NOT NULL,
			summary TEXT NOT NULL,
			created_at TEXT NOT NULL,
			superseded_by TEXT NOT NULL,
			version INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_exp_status ON experiments(status)`,
		`CREATE INDEX IF NOT EXISTS idx_ch_exp ON channels(experiment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_wf_exp ON waveforms(experiment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_an_exp ON anomalies(experiment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_an_ch ON anomalies(channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_an_type ON anomalies(type)`,
		`CREATE INDEX IF NOT EXISTS idx_ana_exp ON analyses(experiment_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// NowRFC3339 返回当前时间的 RFC3339 字符串。
func NowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }
