// Package service 编排领域服务：试验、采集、校准、分析、发布全链路。
package service

import (
	"database/sql"

	"task211-quenchprop/internal/store"
)

// Config 为分析流水线的可调参数。
type Config struct {
	CalibrationThreshold float64 // 淬灭起点判定阈值（量测值）
	MinSlope             float64 // 最小上升斜率（量测值/秒）
	ProtectionMarginMs   float64 // 保护窗口安全裕度（毫秒）
	MaxProtectionMs      float64 // 磁体保护系统最大容忍窗口（毫秒）
	DefaultCriticalSeg   string  // 缺省关键分段 ID
}

// DefaultConfig 返回一套合理的默认参数。
func DefaultConfig() Config {
	return Config{
		CalibrationThreshold: 0.5,
		MinSlope:             0.05,
		ProtectionMarginMs:   50,
		MaxProtectionMs:      500,
		DefaultCriticalSeg:   "seg-crit",
	}
}

// App 为服务聚合根，持有存储与各业务方法。
type App struct {
	db     *sql.DB
	exp    *store.ExperimentStore
	topo   *store.TopologyStore
	chanS  *store.ChannelStore
	wf     *store.WaveformStore
	anom   *store.AnomalyStore
	ana    *store.AnalysisStore
	cfg    Config
}

// NewApp 构造服务聚合根。
func NewApp(db *sql.DB, cfg Config) *App {
	return &App{
		db:    db,
		exp:   store.NewExperimentStore(db),
		topo:  store.NewTopologyStore(db),
		chanS: store.NewChannelStore(db),
		wf:    store.NewWaveformStore(db),
		anom:  store.NewAnomalyStore(db),
		ana:   store.NewAnalysisStore(db),
		cfg:   cfg,
	}
}

// DB 暴露底层数据库连接（供需要事务的调用方使用）。
func (a *App) DB() *sql.DB { return a.db }
