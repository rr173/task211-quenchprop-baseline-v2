package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/service"
)

// 请求 DTO ----------------------------------------------------------------

type experimentReq struct {
	Name       string  `json:"name"`
	MagnetSN   string  `json:"magnet_sn"`
	Operator   string  `json:"operator"`
	TopologyID string  `json:"topology_id"`
	SampleRate float64 `json:"sample_rate"`
}

type transitionReq struct {
	To string `json:"to"`
}

type channelReq struct {
	Index     int     `json:"index"`
	Label     string  `json:"label"`
	Kind      string  `json:"kind"`
	SegmentID string  `json:"segment_id"`
	Status    string  `json:"status"`
	Position  float64 `json:"position"`
	DelayNs   float64 `json:"delay_ns"`
}

type topoSegmentReq struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Position float64 `json:"position"`
}

type topoEdgeReq struct {
	ID       string  `json:"id"`
	FromID   string  `json:"from_id"`
	ToID     string  `json:"to_id"`
	Distance float64 `json:"distance"`
}

type topologyReq struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Segments []topoSegmentReq `json:"segments"`
	Edges    []topoEdgeReq    `json:"edges"`
}

type sampleReq struct {
	RawT float64 `json:"raw_t"`
	V    float64 `json:"v"`
}

type waveformReq struct {
	ChannelID  string      `json:"channel_id"`
	SampleRate float64     `json:"sample_rate"`
	Points     []sampleReq `json:"points"`
}

type calibrateReq struct {
	RefChannelID string `json:"ref_channel_id"`
}

type analyzeReq struct {
	Threshold     float64 `json:"threshold"`
	MinSlope      float64 `json:"min_slope"`
	CriticalSegID string  `json:"critical_seg_id"`
	MarginMs      float64 `json:"margin_ms"`
	MaxProtectMs  float64 `json:"max_protect_ms"`
}

// 响应辅助 ----------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid json body: "+err.Error()))
		return false
	}
	var trailing interface{}
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("request body must contain one JSON document")
		}
		writeErr(w, http.StatusBadRequest, errors.New("invalid json body: "+err.Error()))
		return false
	}
	return true
}

func errStatus(err error) int {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrConflict), errors.Is(err, model.ErrInvalidState),
		errors.Is(err, model.ErrSealed), errors.Is(err, model.ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, model.ErrInvalidArgument), errors.Is(err, model.ErrBadInput),
		errors.Is(err, model.ErrRateMismatch), errors.Is(err, model.ErrTimeAxisBroken),
		errors.Is(err, model.ErrUnknownChannel), errors.Is(err, model.ErrInsufficientData):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// Health 健康检查。
func (h *API) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CreateTopology 创建拓扑。
func (h *API) CreateTopology(w http.ResponseWriter, r *http.Request) {
	var req topologyReq
	if !readJSON(w, r, &req) {
		return
	}
	t := &model.CoilTopology{ID: req.ID, Name: req.Name}
	for _, s := range req.Segments {
		t.Segments = append(t.Segments, model.TopoSegment{ID: s.ID, Label: s.Label, Position: s.Position})
	}
	for _, e := range req.Edges {
		t.Edges = append(t.Edges, model.TopoEdge{ID: e.ID, FromID: e.FromID, ToID: e.ToID, Distance: e.Distance})
	}
	if err := h.svc.AddTopology(t); err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// GetTopology 读取拓扑。
func (h *API) GetTopology(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetTopology(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// CreateExperiment 创建试验。
func (h *API) CreateExperiment(w http.ResponseWriter, r *http.Request) {
	var req experimentReq
	if !readJSON(w, r, &req) {
		return
	}
	e, err := h.svc.CreateExperiment(service.ExperimentInput{
		Name: req.Name, MagnetSN: req.MagnetSN, Operator: req.Operator,
		TopologyID: req.TopologyID, SampleRate: req.SampleRate,
	})
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// ListExperiments 列出试验。
func (h *API) ListExperiments(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListExperiments()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// GetExperiment 读取试验。
func (h *API) GetExperiment(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.GetExperiment(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// TransitionExperiment 试验状态迁移。
func (h *API) TransitionExperiment(w http.ResponseWriter, r *http.Request) {
	var req transitionReq
	if !readJSON(w, r, &req) {
		return
	}
	e, err := h.svc.TransitionExperiment(r.PathValue("id"), req.To)
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// AddChannel 新增通道。
func (h *API) AddChannel(w http.ResponseWriter, r *http.Request) {
	var req channelReq
	if !readJSON(w, r, &req) {
		return
	}
	c := &model.Channel{
		Index: req.Index, Label: req.Label, Kind: req.Kind, SegmentID: req.SegmentID,
		Status: req.Status, Position: req.Position, DelayNs: req.DelayNs,
	}
	if err := h.svc.AddChannel(r.PathValue("id"), c); err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// ListChannels 列出通道。
func (h *API) ListChannels(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListChannels(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// AttachTopology 将拓扑绑定到试验（替换试验的拓扑引用）。
func (h *API) AttachTopology(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TopologyID string `json:"topology_id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if _, err := h.svc.GetTopology(req.TopologyID); err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	e, err := h.svc.GetExperiment(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	e.TopologyID = req.TopologyID
	if err := h.svc.UpdateExperimentTopology(e); err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// IngestWaveform 摄入波形。
func (h *API) IngestWaveform(w http.ResponseWriter, r *http.Request) {
	var req waveformReq
	if !readJSON(w, r, &req) {
		return
	}
	pts := make([]model.Sample, len(req.Points))
	for i, p := range req.Points {
		pts[i] = model.Sample{RawT: p.RawT, V: p.V}
	}
	res, err := h.svc.IngestWaveform(r.PathValue("id"), req.ChannelID, req.SampleRate, pts)
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// ListWaveforms 列出波形。
func (h *API) ListWaveforms(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListWaveforms(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// Calibrate 校准。
func (h *API) Calibrate(w http.ResponseWriter, r *http.Request) {
	var req calibrateReq
	if !readJSON(w, r, &req) {
		return
	}
	delays, err := h.svc.Calibrate(r.PathValue("id"), req.RefChannelID)
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"delays_ns": delays})
}

// Analyze 分析。
func (h *API) Analyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeReq
	if !readJSON(w, r, &req) {
		return
	}
	res, err := h.svc.Analyze(r.PathValue("id"), service.AnalyzeOptions{
		Threshold: req.Threshold, MinSlope: req.MinSlope, CriticalSegID: req.CriticalSegID,
		MarginMs: req.MarginMs, MaxProtectMs: req.MaxProtectMs,
	})
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ListAnalyses 列出分析包。
func (h *API) ListAnalyses(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAnalyses(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ListAnomalies 列出异常段。
func (h *API) ListAnomalies(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAnomalies(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// GetAnalysis 读取分析包。
func (h *API) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	an, err := h.svc.GetAnalysis(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, an)
}

// TransitionAnalysis 分析包状态迁移。
func (h *API) TransitionAnalysis(w http.ResponseWriter, r *http.Request) {
	var req transitionReq
	if !readJSON(w, r, &req) {
		return
	}
	var an *model.AnalysisPackage
	var err error
	if req.To == model.AnaPublished {
		an, err = h.svc.PublishAnalysis(r.PathValue("id"))
	} else {
		an, err = h.svc.TransitionAnalysis(r.PathValue("id"), req.To)
	}
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, an)
}

// GetAnomaly 读取异常段。
func (h *API) GetAnomaly(w http.ResponseWriter, r *http.Request) {
	an, err := h.svc.GetAnomaly(r.PathValue("id"))
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, an)
}

// TransitionAnomaly 异常段状态迁移。
func (h *API) TransitionAnomaly(w http.ResponseWriter, r *http.Request) {
	var req transitionReq
	if !readJSON(w, r, &req) {
		return
	}
	an, err := h.svc.TransitionAnomaly(r.PathValue("id"), req.To)
	if err != nil {
		writeErr(w, errStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, an)
}
