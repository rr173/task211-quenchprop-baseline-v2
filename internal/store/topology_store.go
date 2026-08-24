package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"task211-quenchprop/internal/model"
)

// TopologyStore 封装线圈拓扑的持久化。
type TopologyStore struct {
	db *sql.DB
}

// NewTopologyStore 构造拓扑存储。
func NewTopologyStore(db *sql.DB) *TopologyStore { return &TopologyStore{db: db} }

// Save 写入拓扑及其分段与邻接边（整表替换该拓扑的拓扑结构）。
func (s *TopologyStore) Save(t *model.CoilTopology) error {
	if err := t.Validate(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	raw, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal topology: %w", err)
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO topologies (id,name,raw_json) VALUES (?,?,?)`,
		t.ID, t.Name, string(raw)); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM topology_segments WHERE topology_id=?`, t.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM topology_edges WHERE topology_id=?`, t.ID); err != nil {
		return err
	}
	for _, seg := range t.Segments {
		if _, err := tx.Exec(
			`INSERT INTO topology_segments (id,topology_id,label,position) VALUES (?,?,?,?)`,
			seg.ID, t.ID, seg.Label, seg.Position); err != nil {
			return err
		}
	}
	for _, e := range t.Edges {
		if _, err := tx.Exec(
			`INSERT INTO topology_edges (id,topology_id,from_id,to_id,distance) VALUES (?,?,?,?,?)`,
			e.ID, t.ID, e.FromID, e.ToID, e.Distance); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Get 读取拓扑（由 raw_json 重建）。
func (s *TopologyStore) Get(id string) (*model.CoilTopology, error) {
	var raw string
	if err := s.db.QueryRow(`SELECT raw_json FROM topologies WHERE id=?`, id).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get topology: %w", err)
	}
	var t model.CoilTopology
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("unmarshal topology: %w", err)
	}
	return &t, nil
}

// Exists 判断拓扑是否存在。
func (s *TopologyStore) Exists(id string) (bool, error) {
	var c int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM topologies WHERE id=?`, id).Scan(&c); err != nil {
		return false, err
	}
	return c > 0, nil
}
