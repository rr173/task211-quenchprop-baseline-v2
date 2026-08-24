package store

import (
	"database/sql"
	"fmt"
	"time"

	"task211-quenchprop/internal/model"
)

// WaveformStore 封装波形及其采样点的持久化。
type WaveformStore struct {
	db *sql.DB
}

// NewWaveformStore 构造波形存储。
func NewWaveformStore(db *sql.DB) *WaveformStore { return &WaveformStore{db: db} }

// ExistsByFingerprint 判断某指纹的波形是否已存在（幂等判定）。
func (s *WaveformStore) ExistsByFingerprint(fp string) (bool, error) {
	var c int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM waveforms WHERE fingerprint=?`, fp).Scan(&c); err != nil {
		return false, err
	}
	return c > 0, nil
}

// GetByFingerprint 按指纹读取既有波形（含采样点），用于幂等命中。
func (s *WaveformStore) GetByFingerprint(fp string) (*model.Waveform, error) {
	var id string
	if err := s.db.QueryRow(`SELECT id FROM waveforms WHERE fingerprint=?`, fp).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("lookup fingerprint: %w", err)
	}
	return s.Get(id)
}

// Save 写入波形头与全部采样点（调用方需先判定指纹幂等）。
func (s *WaveformStore) Save(w *model.Waveform) error {
	if w.CreatedAt.IsZero() {
		w.CreatedAt = time.Now()
	}
	w.Version = 1
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO waveforms (id,experiment_id,channel_id,sample_rate,fingerprint,calibrated,delay_ns,n_points,created_at,version)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		w.ID, w.ExperimentID, w.ChannelID, w.SampleRate, w.Fingerprint, boolToInt(w.Calibrated),
		w.DelayNs, len(w.Points), w.CreatedAt.UTC().Format(time.RFC3339), w.Version); err != nil {
		return fmt.Errorf("insert waveform: %w", err)
	}
	for i, p := range w.Points {
		if _, err := tx.Exec(
			`INSERT INTO waveform_samples (waveform_id,idx,raw_t,v) VALUES (?,?,?,?)`,
			w.ID, i, p.RawT, p.V); err != nil {
			return fmt.Errorf("insert sample: %w", err)
		}
	}
	return tx.Commit()
}

// Get 按 ID 读取波形（含采样点）。
func (s *WaveformStore) Get(id string) (*model.Waveform, error) {
	var (
		w             model.Waveform
		createdAt     string
		calibratedInt int
	)
	if err := s.db.QueryRow(
		`SELECT id,experiment_id,channel_id,sample_rate,fingerprint,calibrated,delay_ns,n_points,created_at,version
		 FROM waveforms WHERE id=?`, id).Scan(
		&w.ID, &w.ExperimentID, &w.ChannelID, &w.SampleRate, &w.Fingerprint,
		&calibratedInt, &w.DelayNs, &w.Version, &createdAt, &w.Version); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get waveform: %w", err)
	}
	w.Calibrated = calibratedInt != 0
	if t, perr := time.Parse(time.RFC3339, createdAt); perr == nil {
		w.CreatedAt = t
	}
	pts, err := s.loadSamples(id)
	if err != nil {
		return nil, err
	}
	w.Points = pts
	return &w, nil
}

func (s *WaveformStore) loadSamples(waveformID string) ([]model.Sample, error) {
	rows, err := s.db.Query(`SELECT raw_t,v FROM waveform_samples WHERE waveform_id=? ORDER BY idx`, waveformID)
	if err != nil {
		return nil, fmt.Errorf("load samples: %w", err)
	}
	defer rows.Close()
	var pts []model.Sample
	for rows.Next() {
		var p model.Sample
		if err := rows.Scan(&p.RawT, &p.V); err != nil {
			return nil, err
		}
		pts = append(pts, p)
	}
	return pts, rows.Err()
}

// ListByExperiment 列出某试验下全部波形（不含采样点，仅头部）。
func (s *WaveformStore) ListByExperiment(expID string) ([]*model.Waveform, error) {
	rows, err := s.db.Query(
		`SELECT id,experiment_id,channel_id,sample_rate,fingerprint,calibrated,delay_ns,n_points,created_at,version
		 FROM waveforms WHERE experiment_id=? ORDER BY channel_id`, expID)
	if err != nil {
		return nil, fmt.Errorf("list waveforms: %w", err)
	}
	defer rows.Close()
	var out []*model.Waveform
	for rows.Next() {
		var (
			w             model.Waveform
			createdAt     string
			calibratedInt int
		)
		if err := rows.Scan(&w.ID, &w.ExperimentID, &w.ChannelID, &w.SampleRate, &w.Fingerprint,
			&calibratedInt, &w.DelayNs, &w.Version, &createdAt, &w.Version); err != nil {
			return nil, err
		}
		w.Calibrated = calibratedInt != 0
		if t, perr := time.Parse(time.RFC3339, createdAt); perr == nil {
			w.CreatedAt = t
		}
		out = append(out, &w)
	}
	return out, rows.Err()
}

// ListFullByExperiment 列出某试验下全部波形（含采样点，供分析使用）。
func (s *WaveformStore) ListFullByExperiment(expID string) ([]*model.Waveform, error) {
	heads, err := s.ListByExperiment(expID)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Waveform, 0, len(heads))
	for _, h := range heads {
		full, err := s.Get(h.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, full)
	}
	return out, nil
}

// UpdateCalibration 回写校准结果（延迟与校准标记）。
func (s *WaveformStore) UpdateCalibration(id string, delayNs float64, calibrated bool) error {
	if _, err := s.db.Exec(`UPDATE waveforms SET delay_ns=?,calibrated=?,version=version+1 WHERE id=?`,
		delayNs, boolToInt(calibrated), id); err != nil {
		return fmt.Errorf("update calibration: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
