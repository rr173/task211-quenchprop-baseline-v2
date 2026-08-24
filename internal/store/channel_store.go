package store

import (
	"database/sql"
	"fmt"

	"task211-quenchprop/internal/model"
)

// ChannelStore 封装通道的持久化。
type ChannelStore struct {
	db *sql.DB
}

// NewChannelStore 构造通道存储。
func NewChannelStore(db *sql.DB) *ChannelStore { return &ChannelStore{db: db} }

// Create 插入一个通道。
func (s *ChannelStore) Create(c *model.Channel) error {
	c.Version = 1
	_, err := s.db.Exec(
		`INSERT INTO channels (id,experiment_id,channel_index,label,kind,segment_id,status,position,delay_ns,version)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.ExperimentID, c.Index, c.Label, c.Kind, c.SegmentID, c.Status, c.Position, c.DelayNs, c.Version)
	if err != nil {
		return fmt.Errorf("insert channel: %w", err)
	}
	return nil
}

// Get 按 ID 读取通道。
func (s *ChannelStore) Get(id string) (*model.Channel, error) {
	row := s.db.QueryRow(
		`SELECT id,experiment_id,channel_index,label,kind,segment_id,status,position,delay_ns,version
		 FROM channels WHERE id=?`, id)
	return scanChannel(row)
}

// ListByExperiment 列出某试验下的全部通道。
func (s *ChannelStore) ListByExperiment(expID string) ([]*model.Channel, error) {
	rows, err := s.db.Query(
		`SELECT id,experiment_id,channel_index,label,kind,segment_id,status,position,delay_ns,version
		 FROM channels WHERE experiment_id=? ORDER BY channel_index`, expID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()
	var out []*model.Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateStatus 更新通道状态（乐观锁）。
func (s *ChannelStore) UpdateStatus(id string, status string, version int64) error {
	res, err := s.db.Exec(`UPDATE channels SET status=?,version=? WHERE id=? AND version=?`,
		status, version+1, id, version)
	if err != nil {
		return fmt.Errorf("update channel status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrConflict
	}
	return nil
}

func scanChannel(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Channel, error) {
	var c model.Channel
	if err := scanner.Scan(
		&c.ID, &c.ExperimentID, &c.Index, &c.Label, &c.Kind, &c.SegmentID,
		&c.Status, &c.Position, &c.DelayNs, &c.Version); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan channel: %w", err)
	}
	return &c, nil
}
