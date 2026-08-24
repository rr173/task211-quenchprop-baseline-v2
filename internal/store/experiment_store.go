package store

import (
	"database/sql"
	"fmt"
	"time"

	"task211-quenchprop/internal/model"
)

// ExperimentStore 封装放电试验的持久化操作。
type ExperimentStore struct {
	db *sql.DB
}

// NewExperimentStore 构造试验存储。
func NewExperimentStore(db *sql.DB) *ExperimentStore { return &ExperimentStore{db: db} }

// Create 插入一条放电试验（乐观锁版本从 1 起）。
func (s *ExperimentStore) Create(e *model.DischargeExperiment) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	e.Version = 1
	_, err := s.db.Exec(
		`INSERT INTO experiments (id,name,magnet_sn,topology_id,operator,sample_rate,status,created_at,sealed_at,version)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		e.ID, e.Name, e.MagnetSN, e.TopologyID, e.Operator, e.SampleRate, e.Status,
		e.CreatedAt.UTC().Format(time.RFC3339), nil, e.Version,
	)
	if err != nil {
		return fmt.Errorf("insert experiment: %w", err)
	}
	return nil
}

// Get 按 ID 读取放电试验。
func (s *ExperimentStore) Get(id string) (*model.DischargeExperiment, error) {
	row := s.db.QueryRow(
		`SELECT id,name,magnet_sn,topology_id,operator,sample_rate,status,created_at,sealed_at,version
		 FROM experiments WHERE id=?`, id)
	return scanExperiment(row)
}

// List 列出全部放电试验（按创建时间倒序）。
func (s *ExperimentStore) List() ([]*model.DischargeExperiment, error) {
	rows, err := s.db.Query(
		`SELECT id,name,magnet_sn,topology_id,operator,sample_rate,status,created_at,sealed_at,version
		 FROM experiments ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	defer rows.Close()
	var out []*model.DischargeExperiment
	for rows.Next() {
		e, err := scanExperiment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Update 持久化状态机迁移（带乐观锁版本校验）。
func (s *ExperimentStore) Update(e *model.DischargeExperiment) error {
	next := e.Version + 1
	var sealedAt interface{}
	if e.SealedAt != nil {
		sealedAt = e.SealedAt.UTC().Format(time.RFC3339)
	}
	res, err := s.db.Exec(
		`UPDATE experiments SET name=?,magnet_sn=?,topology_id=?,operator=?,sample_rate=?,status=?,sealed_at=?,version=?
		 WHERE id=? AND version=?`,
		e.Name, e.MagnetSN, e.TopologyID, e.Operator, e.SampleRate, e.Status, sealedAt, next, e.ID, e.Version)
	if err != nil {
		return fmt.Errorf("update experiment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrConflict
	}
	e.Version = next
	return nil
}

// Exists 判断试验是否存在。
func (s *ExperimentStore) Exists(id string) (bool, error) {
	var c int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM experiments WHERE id=?`, id).Scan(&c); err != nil {
		return false, err
	}
	return c > 0, nil
}

func scanExperiment(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.DischargeExperiment, error) {
	var (
		e                                   model.DischargeExperiment
		createdAt, sealedAtSQL              sql.NullString
	)
	if err := scanner.Scan(
		&e.ID, &e.Name, &e.MagnetSN, &e.TopologyID, &e.Operator, &e.SampleRate,
		&e.Status, &createdAt, &sealedAtSQL, &e.Version); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan experiment: %w", err)
	}
	if createdAt.Valid {
		if t, perr := time.Parse(time.RFC3339, createdAt.String); perr == nil {
			e.CreatedAt = t
		}
	}
	if sealedAtSQL.Valid {
		if t, perr := time.Parse(time.RFC3339, sealedAtSQL.String); perr == nil {
			ts := t
			e.SealedAt = &ts
		}
	}
	return &e, nil
}
