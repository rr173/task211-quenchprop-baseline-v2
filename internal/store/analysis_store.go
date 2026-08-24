package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"task211-quenchprop/internal/model"
)

// AnalysisStore 封装分析包的持久化。
type AnalysisStore struct {
	db *sql.DB
}

// NewAnalysisStore 构造分析包存储。
func NewAnalysisStore(db *sql.DB) *AnalysisStore { return &AnalysisStore{db: db} }

// Create 插入一个分析包。
func (s *AnalysisStore) Create(a *model.AnalysisPackage) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	a.Version = 1
	segs := strings.Join(a.AffectedSegments, ",")
	_, err := s.db.Exec(
		`INSERT INTO analyses
		 (id,experiment_id,status,onset_channel_id,onset_time,propagation_speed,affected_segments,protection_window_ms,breach,summary,created_at,superseded_by,version)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.ExperimentID, a.Status, a.OnsetChannelID, a.OnsetTime, a.PropagationSpeed,
		segs, a.ProtectionWindowMs, boolToInt(a.Breach), a.Summary,
		a.CreatedAt.UTC().Format(time.RFC3339), a.SupersededBy, a.Version)
	if err != nil {
		return fmt.Errorf("insert analysis: %w", err)
	}
	return nil
}

// Get 按 ID 读取分析包。
func (s *AnalysisStore) Get(id string) (*model.AnalysisPackage, error) {
	row := s.db.QueryRow(
		`SELECT id,experiment_id,status,onset_channel_id,onset_time,propagation_speed,affected_segments,
		        protection_window_ms,breach,summary,created_at,superseded_by,version
		 FROM analyses WHERE id=?`, id)
	return scanAnalysis(row)
}

// ListByExperiment 列出某试验的全部分析包（含已替代）。
func (s *AnalysisStore) ListByExperiment(expID string) ([]*model.AnalysisPackage, error) {
	rows, err := s.db.Query(
		`SELECT id,experiment_id,status,onset_channel_id,onset_time,propagation_speed,affected_segments,
		        protection_window_ms,breach,summary,created_at,superseded_by,version
		 FROM analyses WHERE experiment_id=? ORDER BY created_at DESC`, expID)
	if err != nil {
		return nil, fmt.Errorf("list analyses: %w", err)
	}
	defer rows.Close()
	var out []*model.AnalysisPackage
	for rows.Next() {
		a, err := scanAnalysis(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// LatestPublished 返回某试验最新已发布的分析包（用于替代判定）。
func (s *AnalysisStore) LatestPublished(expID string) (*model.AnalysisPackage, error) {
	row := s.db.QueryRow(
		`SELECT id,experiment_id,status,onset_channel_id,onset_time,propagation_speed,affected_segments,
		        protection_window_ms,breach,summary,created_at,superseded_by,version
		 FROM analyses WHERE experiment_id=? AND status=? ORDER BY created_at DESC LIMIT 1`,
		expID, model.AnaPublished)
	return scanAnalysis(row)
}

// UpdateStatus 更新分析包状态（状态机迁移 + 乐观锁）。
func (s *AnalysisStore) UpdateStatus(id, status string, version int64) error {
	res, err := s.db.Exec(`UPDATE analyses SET status=?,version=? WHERE id=? AND version=?`,
		status, version+1, id, version)
	if err != nil {
		return fmt.Errorf("update analysis status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrConflict
	}
	return nil
}

// Supersede 将某分析包标记为已替代，并记录取代者 ID。
func (s *AnalysisStore) Supersede(id, byID string, version int64) error {
	res, err := s.db.Exec(`UPDATE analyses SET status=?,superseded_by=?,version=? WHERE id=? AND version=?`,
		model.AnaSuperseded, byID, version+1, id, version)
	if err != nil {
		return fmt.Errorf("supersede analysis: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrConflict
	}
	return nil
}

func scanAnalysis(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.AnalysisPackage, error) {
	var (
		a                       model.AnalysisPackage
		segs                    string
		createdAt               string
		breachInt               int
	)
	if err := scanner.Scan(
		&a.ID, &a.ExperimentID, &a.Status, &a.OnsetChannelID, &a.OnsetTime, &a.PropagationSpeed,
		&segs, &a.ProtectionWindowMs, &breachInt, &a.Summary, &createdAt, &a.SupersededBy, &a.Version); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan analysis: %w", err)
	}
	a.Breach = breachInt != 0
	if segs != "" {
		a.AffectedSegments = strings.Split(segs, ",")
	} else {
		a.AffectedSegments = []string{}
	}
	if t, perr := time.Parse(time.RFC3339, createdAt); perr == nil {
		a.CreatedAt = t
	}
	return &a, nil
}
