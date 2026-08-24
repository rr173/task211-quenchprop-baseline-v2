// Package httpapi 暴露低温超导磁体淬灭传播分析服务的 HTTP 接口。
package httpapi

import (
	"net/http"

	"task211-quenchprop/internal/service"
)

// API 为 HTTP 处理器聚合，包装服务层。
type API struct {
	svc *service.App
}

// NewRouter 构造 HTTP 路由，挂载全部端点（均带 /api 前缀）。
func NewRouter(app *service.App) *http.ServeMux {
	h := &API{svc: app}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("POST /api/topologies", h.CreateTopology)
	mux.HandleFunc("GET /api/topologies/{id}", h.GetTopology)

	mux.HandleFunc("POST /api/experiments", h.CreateExperiment)
	mux.HandleFunc("GET /api/experiments", h.ListExperiments)
	mux.HandleFunc("GET /api/experiments/{id}", h.GetExperiment)
	mux.HandleFunc("POST /api/experiments/{id}/transition", h.TransitionExperiment)
	mux.HandleFunc("POST /api/experiments/{id}/channels", h.AddChannel)
	mux.HandleFunc("GET /api/experiments/{id}/channels", h.ListChannels)
	mux.HandleFunc("POST /api/experiments/{id}/topology", h.AttachTopology)
	mux.HandleFunc("POST /api/experiments/{id}/waveforms", h.IngestWaveform)
	mux.HandleFunc("GET /api/experiments/{id}/waveforms", h.ListWaveforms)
	mux.HandleFunc("POST /api/experiments/{id}/calibrate", h.Calibrate)
	mux.HandleFunc("POST /api/experiments/{id}/analyze", h.Analyze)
	mux.HandleFunc("GET /api/experiments/{id}/analyses", h.ListAnalyses)
	mux.HandleFunc("GET /api/experiments/{id}/anomalies", h.ListAnomalies)

	mux.HandleFunc("POST /api/analyses/{id}/transition", h.TransitionAnalysis)
	mux.HandleFunc("GET /api/analyses/{id}", h.GetAnalysis)
	mux.HandleFunc("POST /api/anomalies/{id}/transition", h.TransitionAnomaly)
	mux.HandleFunc("GET /api/anomalies/{id}", h.GetAnomaly)

	return mux
}
