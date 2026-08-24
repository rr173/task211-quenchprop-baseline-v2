package service

import (
	"fmt"
	"time"

	"task211-quenchprop/internal/calibration"
	"task211-quenchprop/internal/ingest"
	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/propagation"
	"task211-quenchprop/internal/protection"
	"task211-quenchprop/internal/report"
)

// ExperimentInput 为创建试验的输入。
type ExperimentInput struct {
	Name       string
	MagnetSN   string
	Operator   string
	TopologyID string
	SampleRate float64
}

// AnalyzeOptions 为分析流水线的可选覆盖参数。
type AnalyzeOptions struct {
	RefChannelID  string
	Threshold     float64
	MinSlope      float64
	CriticalSegID string
	MarginMs      float64
	MaxProtectMs  float64
}

// AnalyzeResult 为一次分析的完整结果。
type AnalyzeResult struct {
	Analysis    *model.AnalysisPackage
	Propagation propagation.PropagationResult
	Anomalies   []*model.AnomalySegment
	Calibrated  bool
}

// CreateExperiment 创建一份处于「准备」状态的放电试验。
func (a *App) CreateExperiment(in ExperimentInput) (*model.DischargeExperiment, error) {
	if in.Name == "" || in.MagnetSN == "" || in.TopologyID == "" || in.SampleRate <= 0 {
		return nil, model.ErrInvalidArgument
	}
	if ok, _ := a.topo.Exists(in.TopologyID); !ok {
		return nil, fmt.Errorf("topology %s not found", in.TopologyID)
	}
	e := &model.DischargeExperiment{
		ID:         model.NewID("exp"),
		Name:       in.Name,
		MagnetSN:   in.MagnetSN,
		TopologyID: in.TopologyID,
		Operator:   in.Operator,
		SampleRate: in.SampleRate,
		Status:     model.ExpStatusPrepared,
		CreatedAt:  time.Now(),
	}
	if err := a.exp.Create(e); err != nil {
		return nil, err
	}
	return e, nil
}

// AddTopology 保存一份线圈拓扑（供后续试验引用）。
func (a *App) AddTopology(t *model.CoilTopology) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return a.topo.Save(t)
}

// AddChannel 为试验新增一个通道。
func (a *App) AddChannel(expID string, c *model.Channel) error {
	e, err := a.exp.Get(expID)
	if err != nil {
		return err
	}
	if !e.IsWritable() {
		return model.ErrSealed
	}
	if c.SegmentID == "" {
		return model.ErrInvalidArgument
	}
	c.ID = model.NewID("ch")
	c.ExperimentID = expID
	if c.Status == "" {
		c.Status = model.ChanOnline
	}
	if c.Kind == "" {
		c.Kind = "voltage"
	}
	return a.chanS.Create(c)
}

// IngestWaveform 幂等摄入一条通道波形：自动把试验从「准备」推进到「采集」，
// 校验时间轴与采样率，命中指纹则直接报告重复。
func (a *App) IngestWaveform(expID, channelID string, sampleRate float64, points []model.Sample) (*ingest.Result, error) {
	e, err := a.exp.Get(expID)
	if err != nil {
		return nil, err
	}
	if !e.IsWritable() {
		return nil, model.ErrSealed
	}
	if e.Status == model.ExpStatusPrepared {
		if err := e.Transition(model.ExpStatusCollecting); err != nil {
			return nil, err
		}
		if err := a.exp.Update(e); err != nil {
			return nil, err
		}
	}
	ch, err := a.chanS.Get(channelID)
	if err != nil {
		return nil, err
	}
	if ch.ExperimentID != expID {
		return nil, model.ErrUnknownChannel
	}
	if !ch.IsParticipating() {
		return nil, model.ErrUnknownChannel
	}
	w := &model.Waveform{
		ExperimentID: expID,
		ChannelID:    channelID,
		SampleRate:   sampleRate,
		DelayNs:      ch.DelayNs,
		Points:       points,
	}
	return ingest.Ingest(a.wf, w, e.SampleRate)
}

// Calibrate 以 refChannelID 为基准对各通道波形做延迟校准。
func (a *App) Calibrate(expID, refChannelID string) (map[string]float64, error) {
	wfs, err := a.wf.ListFullByExperiment(expID)
	if err != nil {
		return nil, err
	}
	if len(wfs) == 0 {
		return nil, model.ErrInsufficientData
	}
	delays, err := calibration.CalibrateDelays(wfs, refChannelID)
	if err != nil {
		return nil, err
	}
	for _, w := range wfs {
		if err := a.wf.UpdateCalibration(w.ID, w.DelayNs, w.Calibrated); err != nil {
			return nil, err
		}
	}
	return delays, nil
}

// Analyze 执行完整分析流水线：检测起点、分类异常、沿拓扑传播、计算保护窗口、
// 生成草稿分析包。试验会从「采集」推进到「分析中」。
func (a *App) Analyze(expID string, opts AnalyzeOptions) (result *AnalyzeResult, retErr error) {
	e, err := a.exp.Get(expID)
	if err != nil {
		return nil, err
	}
	if !e.IsWritable() {
		return nil, model.ErrSealed
	}
	startedAnalysis := false
	if e.Status == model.ExpStatusCollecting {
		if err := e.Transition(model.ExpStatusAnalyzing); err != nil {
			return nil, err
		}
		if err := a.exp.Update(e); err != nil {
			return nil, err
		}
		startedAnalysis = true
	}
	defer func() {
		if !startedAnalysis || retErr == nil {
			return
		}
		e.Status = model.ExpStatusCollecting
		e.SealedAt = nil
		if rollbackErr := a.exp.Update(e); rollbackErr != nil {
			retErr = fmt.Errorf("analysis failed (%v); rollback failed: %w", retErr, rollbackErr)
		}
	}()
	if e.Status != model.ExpStatusAnalyzing {
		return nil, fmt.Errorf("experiment must be in analyzing state, got %s", e.Status)
	}

	threshold := opts.Threshold
	if threshold == 0 {
		threshold = a.cfg.CalibrationThreshold
	}
	minSlope := opts.MinSlope
	if minSlope == 0 {
		minSlope = a.cfg.MinSlope
	}
	marginMs := opts.MarginMs
	if marginMs == 0 {
		marginMs = a.cfg.ProtectionMarginMs
	}
	maxProtectMs := opts.MaxProtectMs
	if maxProtectMs == 0 {
		maxProtectMs = a.cfg.MaxProtectionMs
	}
	criticalSeg := opts.CriticalSegID
	if criticalSeg == "" {
		criticalSeg = a.cfg.DefaultCriticalSeg
	}

	channels, err := a.chanS.ListByExperiment(expID)
	if err != nil {
		return nil, err
	}
	topo, err := a.topo.Get(e.TopologyID)
	if err != nil {
		return nil, err
	}
	wfs, err := a.wf.ListFullByExperiment(expID)
	if err != nil {
		return nil, err
	}
	wfByChan := map[string]*model.Waveform{}
	for _, w := range wfs {
		wfByChan[w.ChannelID] = w
	}

	onsets := map[string]float64{}
	var anomalies []*model.AnomalySegment
	onsetCh := ""
	onsetT := 0.0

	for _, ch := range channels {
		if !ch.IsParticipating() {
			continue
		}
		w, ok := wfByChan[ch.ID]
		if !ok {
			continue
		}
		onset, found := propagation.DetectOnset(w, threshold, minSlope)
		if !found {
			continue
		}
		conf := onset.Confidence
		if ch.IsFaulty() {
			conf *= 0.5 // 坏道只降置信，不丢证据
		}
		seg := &model.AnomalySegment{
			ID:           model.NewID("anom"),
			ExperimentID: expID,
			ChannelID:    ch.ID,
			SampleStart:  maxInt(0, onset.Idx-1),
			SampleEnd:    onset.Idx,
			DetectedAt:   onset.Time,
			Type:         model.AnomCandidate,
			Confidence:   conf,
			Severity:     model.Clamp(onset.Slope/(minSlope*10), 0, 1) * conf,
			Version:      1,
		}
		anomalies = append(anomalies, seg)
		onsets[ch.ID] = onset.Time
		if onsetCh == "" || onset.Time < onsetT {
			onsetCh = ch.ID
			onsetT = onset.Time
		}
	}

	if len(onsets) == 0 {
		return nil, fmt.Errorf("no quench onset detected on any participating channel")
	}

	// 异常段分类：起点通道 -> onset；其余在线通道 -> propagating；坏道保持 candidate（低置信证据）。
	chanStatus := map[string]string{}
	for _, c := range channels {
		chanStatus[c.ID] = c.Status
	}
	for _, an := range anomalies {
		if an.ChannelID == onsetCh {
			_ = an.Transition(model.AnomOnset)
		} else if chanStatus[an.ChannelID] == model.ChanOnline {
			_ = an.Transition(model.AnomPropagating)
		}
	}

	// 持久化异常段
	for _, an := range anomalies {
		if err := a.anom.Create(an); err != nil {
			return nil, err
		}
	}

	propRes := propagation.Propagate(onsets, channels, topo)

	onsetSeg := ""
	if c, err := a.chanS.Get(onsetCh); err == nil {
		onsetSeg = c.SegmentID
	}
	windowMs, breach := protection.ComputeWindow(onsetSeg, criticalSeg, propRes.PropagationSpeed, topo, marginMs, maxProtectMs)

	pkg := report.BuildPackage(expID, onsetCh, onsetT, propRes.PropagationSpeed, propRes.AffectedSegments, windowMs, breach)
	if err := a.ana.Create(pkg); err != nil {
		return nil, err
	}

	return &AnalyzeResult{
		Analysis:    pkg,
		Propagation: propRes,
		Anomalies:   anomalies,
		Calibrated:  true,
	}, nil
}

// PublishAnalysis 将分析包推进至「发布」：草稿→复核→发布，并把同试验既有的
// 已发布分析包标记为「替代」。
func (a *App) PublishAnalysis(anaID string) (*model.AnalysisPackage, error) {
	ana, err := a.ana.Get(anaID)
	if err != nil {
		return nil, err
	}
	if ana.Status == model.AnaDraft {
		if err := ana.Transition(model.AnaReview); err != nil {
			return nil, err
		}
		if err := a.ana.UpdateStatus(ana.ID, ana.Status, ana.Version); err != nil {
			return nil, err
		}
		// 重新读取，获取更新后的版本号，避免乐观锁陈旧。
		ana, err = a.ana.Get(anaID)
		if err != nil {
			return nil, err
		}
	}
	if ana.Status != model.AnaReview {
		return nil, fmt.Errorf("analysis %s cannot be published from state %s", anaID, ana.Status)
	}
	prev, err := a.ana.LatestPublished(ana.ExperimentID)
	if err != nil && err != model.ErrNotFound {
		return nil, err
	}
	if prev != nil && prev.ID != ana.ID {
		if err := a.ana.Supersede(prev.ID, ana.ID, prev.Version); err != nil {
			return nil, err
		}
	}
	if err := ana.Transition(model.AnaPublished); err != nil {
		return nil, err
	}
	if err := a.ana.UpdateStatus(ana.ID, ana.Status, ana.Version); err != nil {
		return nil, err
	}
	return a.ana.Get(ana.ID)
}

// UpdateExperimentTopology 更新试验的拓扑引用。
func (a *App) UpdateExperimentTopology(e *model.DischargeExperiment) error {
	if e == nil || !e.IsWritable() {
		return model.ErrSealed
	}
	return a.exp.Update(e)
}

// TransitionAnalysis 执行分析包状态机迁移（草稿/复核/替代）。
func (a *App) TransitionAnalysis(anaID, to string) (*model.AnalysisPackage, error) {
	ana, err := a.ana.Get(anaID)
	if err != nil {
		return nil, err
	}
	if err := ana.Transition(to); err != nil {
		return nil, err
	}
	if err := a.ana.UpdateStatus(ana.ID, ana.Status, ana.Version); err != nil {
		return nil, err
	}
	return a.ana.Get(anaID)
}

// GetAnomaly 读取单个异常段。
func (a *App) GetAnomaly(id string) (*model.AnomalySegment, error) {
	return a.anom.Get(id)
}

// TransitionExperiment 执行试验状态机迁移。
func (a *App) TransitionExperiment(expID, to string) (*model.DischargeExperiment, error) {
	e, err := a.exp.Get(expID)
	if err != nil {
		return nil, err
	}
	if err := e.Transition(to); err != nil {
		return nil, err
	}
	if err := a.exp.Update(e); err != nil {
		return nil, err
	}
	return e, nil
}

// TransitionAnomaly 执行异常段类型迁移。
func (a *App) TransitionAnomaly(anomID, to string) (*model.AnomalySegment, error) {
	an, err := a.anom.Get(anomID)
	if err != nil {
		return nil, err
	}
	if err := an.Transition(to); err != nil {
		return nil, err
	}
	if err := a.anom.UpdateType(an.ID, an.Type, an.Version); err != nil {
		return nil, err
	}
	return an, nil
}

// 只读查询方法 --------------------------------------------------------------

// GetExperiment 读取试验。
func (a *App) GetExperiment(expID string) (*model.DischargeExperiment, error) {
	return a.exp.Get(expID)
}

// ListExperiments 列出全部试验。
func (a *App) ListExperiments() ([]*model.DischargeExperiment, error) {
	return a.exp.List()
}

// GetTopology 读取拓扑。
func (a *App) GetTopology(id string) (*model.CoilTopology, error) {
	return a.topo.Get(id)
}

// ListChannels 列出试验的通道。
func (a *App) ListChannels(expID string) ([]*model.Channel, error) {
	return a.chanS.ListByExperiment(expID)
}

// ListWaveforms 列出试验的波形头部。
func (a *App) ListWaveforms(expID string) ([]*model.Waveform, error) {
	return a.wf.ListByExperiment(expID)
}

// ListAnalyses 列出试验的分析包。
func (a *App) ListAnalyses(expID string) ([]*model.AnalysisPackage, error) {
	return a.ana.ListByExperiment(expID)
}

// GetAnalysis 读取分析包。
func (a *App) GetAnalysis(anaID string) (*model.AnalysisPackage, error) {
	return a.ana.Get(anaID)
}

// ListAnomalies 列出试验的异常段。
func (a *App) ListAnomalies(expID string) ([]*model.AnomalySegment, error) {
	return a.anom.ListByExperiment(expID)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
