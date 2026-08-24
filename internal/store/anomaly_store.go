package store

import (
	"database/sql"
	"fmt"

	"task211-quenchprop/internal/model"
)

// AnomalyStore 封装异常段的持久化。
type AnomalyStore struct {
	db *sql.DB
}

// NewAnomalyStore 构造异常段存储。
func NewAnomalyStore(db *sql.DB) *AnomalyStore { return &AnomalyStore{db: db} }

// Create 插入一个异常段。
func (s *AnomalyStore) Create(a *model.AnomalySegment) error {
	a.Version = 1
	_, err := s.db.Exec(
		`INSERT INTO anomalies (id,experiment_id,channel_id,sample_start,sample_end,detected_at,type,confidence,severity,version)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.ExperimentID, a.ChannelID, a.SampleStart, a.SampleEnd, a.DetectedAt,
		a.Type, a.Confidence, a.Severity, a.Version)
	if err != nil {
		return fmt.Errorf("insert anomaly: %w", err)
	}
	return nil
}

// Get 按 ID 读取异常段。
func (s *AnomalyStore) Get(id string) (*model.AnomalySegment, error) {
	row := s.db.QueryRow(
		`SELECT id,experiment_id,channel_id,sample_start,sample_end,detected_at,type,confidence,severity,version
		 FROM anomalies WHERE id=?`, id)
	return scanAnomaly(row)
}

// ListByExperiment 列出某试验的异常段。
func (s *AnomalyStore) ListByExperiment(expID string) ([]*model.AnomalySegment, error) {
	rows, err := s.db.Query(
		`SELECT id,experiment_id,channel_id,sample_start,sample_end,detected_at,type,confidence,severity,version
		 FROM anomalies WHERE experiment_id=? ORDER BY detected_at`, expID)
	if err != nil {
		return nil, fmt.Errorf("list anomalies: %w", err)
	}
	defer rows.Close()
	var out []*model.AnomalySegment
	for rows.Next() {
		a, err := scanAnomaly(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListByType 列出某试验特定类型的异常段。
func (s *AnomalyStore) ListByType(expID, typ string) ([]*model.AnomalySegment, error) {
	rows, err := s.db.Query(
		`SELECT id,experiment_id,channel_id,sample_start,sample_end,detected_at,type,confidence,severity,version
		 FROM anomalies WHERE experiment_id=? AND type=?`, expID, typ)
	if err != nil {
		return nil, fmt.Errorf("list anomalies by type: %w", err)
	}
	defer rows.Close()
	var out []*model.AnomalySegment
	for rows.Next() {
		a, err := scanAnomaly(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateType 更新异常段类型（状态机迁移 + 乐观锁）。
func (s *AnomalyStore) UpdateType(id, typ string, version int64) error {
	res, err := s.db.Exec(`UPDATE anomalies SET type=?,version=? WHERE id=? AND version=?`,
		typ, version+1, id, version)
	if err != nil {
		return fmt.Errorf("update anomaly type: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrConflict
	}
	return nil
}

func scanAnomaly(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.AnomalySegment, error) {
	var a model.AnomalySegment
	if err := scanner.Scan(
		&a.ID, &a.ExperimentID, &a.ChannelID, &a.SampleStart, &a.SampleEnd, &a.DetectedAt,
		&a.Type, &a.Confidence, &a.Severity, &a.Version); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan anomaly: %w", err)
	}
	return &a, nil
}
