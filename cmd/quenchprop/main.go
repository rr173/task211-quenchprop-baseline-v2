// Command quenchprop 为低温超导磁体淬灭传播分析服务入口。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"task211-quenchprop/internal/httpapi"
	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/service"
	"task211-quenchprop/internal/store"
)

func main() {
	dbPath := flag.String("db", "./quenchprop.db", "SQLite 数据库路径")
	addr := flag.String("addr", ":8080", "HTTP 监听地址")
	smoke := flag.Bool("smoke-test", false, "运行自测后退出")
	flag.Parse()

	if *smoke {
		os.Exit(runSmokeTest())
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	app := service.NewApp(db, service.DefaultConfig())
	mux := httpapi.NewRouter(app)

	log.Printf("quenchprop listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// runSmokeTest 运行端到端自测，返回退出码（0 成功，1 失败）。
func runSmokeTest() int {
	tmp, err := os.MkdirTemp("", "quenchprop-smoke-")
	if err != nil {
		log.Printf("mkdirtemp: %v", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	dbPath := filepath.Join(tmp, "smoke.db")

	db, err := store.Open(dbPath)
	if err != nil {
		log.Printf("open db: %v", err)
		return 1
	}
	defer db.Close()

	cfg := service.DefaultConfig()
	app := service.NewApp(db, cfg)

	fail := func(format string, args ...interface{}) int {
		log.Printf("SMOKE FAIL: "+format, args...)
		return 1
	}

	// 1. 拓扑：线性链 seg-1 → seg-2 → seg-3 → seg-crit，每段间距 1.0 米。
	topo := &model.CoilTopology{
		ID:   "topo-main",
		Name: "主线圈拓扑",
		Segments: []model.TopoSegment{
			{ID: "seg-1", Label: "S1", Position: 0},
			{ID: "seg-2", Label: "S2", Position: 1.0},
			{ID: "seg-3", Label: "S3", Position: 2.0},
			{ID: "seg-crit", Label: "CRIT", Position: 3.0},
		},
		Edges: []model.TopoEdge{
			{ID: "e12", FromID: "seg-1", ToID: "seg-2", Distance: 1.0},
			{ID: "e23", FromID: "seg-2", ToID: "seg-3", Distance: 1.0},
			{ID: "e3c", FromID: "seg-3", ToID: "seg-crit", Distance: 1.0},
		},
	}
	if err := app.AddTopology(topo); err != nil {
		return fail("add topology: %v", err)
	}

	// 2. 试验。
	exp, err := app.CreateExperiment(service.ExperimentInput{
		Name: "主磁体淬灭传播试验", MagnetSN: "MAG-001", Operator: "op-1",
		TopologyID: topo.ID, SampleRate: 1000,
	})
	if err != nil {
		return fail("create experiment: %v", err)
	}

	// 3. 通道：seg-1/2/3/crit 各一在线通道，seg-2 额外一条坏道。
	// delayNs 为各通道真实的采集延迟（硬件偏斜），由校准环节估计。
	type chSpec struct {
		key     string
		idx     int
		seg     string
		status  string
		onsetT  float64
		delayNs float64
	}
	chSpecs := []chSpec{
		{"a", 1, "seg-1", model.ChanOnline, 0.10, 0},
		{"b", 2, "seg-2", model.ChanOnline, 0.30, 5e6},
		{"bad", 3, "seg-2", model.ChanBad, 0.30, 3e6},
		{"c", 4, "seg-3", model.ChanOnline, 0.50, 10e6},
		{"crit", 5, "seg-crit", model.ChanOnline, 0.70, 15e6},
	}
	chByKey := map[string]*model.Channel{}
	for _, cs := range chSpecs {
		c := &model.Channel{
			Index: cs.idx, SegmentID: cs.seg, Status: cs.status, Kind: "voltage",
			Label: fmt.Sprintf("ch-%s", cs.key), Position: float64(cs.idx), DelayNs: 0,
		}
		if err := app.AddChannel(exp.ID, c); err != nil {
			return fail("add channel %s: %v", cs.key, err)
		}
		chByKey[cs.key] = c
	}

	// 4. 生成波形：含同步校准脉冲（1ms 处、幅度 0.2、最陡边），
	// 淬灭从 seg-1(0.1s) 起依次传播到 seg-2(0.3s)、seg-3(0.5s)、crit(0.7s)。
	for _, cs := range chSpecs {
		c := chByKey[cs.key]
		res, err := app.IngestWaveform(exp.ID, c.ID, 1000, genWaveform(cs.onsetT, cs.delayNs, 1000, 1.0))
		if err != nil {
			return fail("ingest waveform %s: %v", cs.key, err)
		}
		if res.Duplicated {
			return fail("unexpected duplicate on first ingest of %s", cs.key)
		}
	}

	// 5. 幂等：重复摄入 seg-1 应命中重复。
	dupRes, err := app.IngestWaveform(exp.ID, chByKey["a"].ID, 1000, genWaveform(0.10, 0, 1000, 1.0))
	if err != nil {
		return fail("re-ingest seg-1: %v", err)
	}
	if !dupRes.Duplicated {
		return fail("expected duplicate on re-ingest of seg-1")
	}

	// 6. 校准（以 seg-1 通道为基准），并校验估计出的延迟接近真实值。
	delays, err := app.Calibrate(exp.ID, chByKey["a"].ID)
	if err != nil {
		return fail("calibrate: %v", err)
	}
	for _, cs := range chSpecs {
		want := cs.delayNs
		got := delays[chByKey[cs.key].ID]
		if model.Abs(got-want) > 0.2e6 { // 200us 容差
			return fail("calibrated delay for %s = %.0fns, want %.0fns", cs.key, got, want)
		}
	}

	// 7. 分析。
	res, err := app.Analyze(exp.ID, service.AnalyzeOptions{
		Threshold: 0.5, MinSlope: 0.05, CriticalSegID: "seg-crit",
		MarginMs: 50, MaxProtectMs: 500,
	})
	if err != nil {
		return fail("analyze: %v", err)
	}
	if res.Analysis.OnsetChannelID != chByKey["a"].ID {
		return fail("onset channel = %s, want %s", res.Analysis.OnsetChannelID, chByKey["a"].ID)
	}
	if len(res.Analysis.AffectedSegments) != 4 {
		return fail("affected segments = %d, want 4", len(res.Analysis.AffectedSegments))
	}
	if res.Propagation.PropagationSpeed <= 0 {
		return fail("propagation speed not positive: %v", res.Propagation.PropagationSpeed)
	}
	if !res.Analysis.Breach {
		return fail("expected breach (3m @ ~5m/s => 650ms > 500ms)")
	}
	if len(res.Anomalies) != 5 {
		return fail("anomalies = %d, want 5", len(res.Anomalies))
	}

	// 8. 发布分析包（草稿→复核→发布）。
	pub, err := app.PublishAnalysis(res.Analysis.ID)
	if err != nil {
		return fail("publish analysis: %v", err)
	}
	if pub.Status != model.AnaPublished {
		return fail("analysis status = %s, want published", pub.Status)
	}

	// 9. 封存试验，封存后拒写。
	if _, err := app.TransitionExperiment(exp.ID, model.ExpStatusConfirmed); err != nil {
		return fail("transition to confirmed: %v", err)
	}
	if _, err := app.TransitionExperiment(exp.ID, model.ExpStatusSealed); err != nil {
		return fail("transition to sealed: %v", err)
	}
	if _, err := app.IngestWaveform(exp.ID, chByKey["a"].ID, 1000, genWaveform(0.10, 0, 1000, 1.0)); err != model.ErrSealed {
		return fail("expected ErrSealed on write to sealed experiment, got %v", err)
	}

	log.Printf("SMOKE PASS: onset=%s speed=%.3f m/s affected=%d breach=%v anomalies=%d",
		res.Analysis.OnsetChannelID, res.Propagation.PropagationSpeed,
		len(res.Analysis.AffectedSegments), res.Analysis.Breach, len(res.Anomalies))
	return 0
}

// genWaveform 生成一条波形：
//   - 在真实时间 1ms 处注入一个幅度 0.2 的校准脉冲（最陡上升边，低于淬灭阈值 0.5，用于延迟校准）；
//   - 淬灭在真实时间 onsetT 处以 50/s 斜率上升（上限 5）；
//   - delayNs 为该通道真实的采集延迟（叠加到原始记录时间上）。
func genWaveform(onsetT, delayNs, rate, dur float64) []model.Sample {
	n := int(rate * dur)
	pts := make([]model.Sample, n)
	spikeIdx := int(0.001 * rate) // 校准脉冲在真实时间 1ms
	for i := 0; i < n; i++ {
		trueT := float64(i) / rate
		rawT := trueT + delayNs/1e9
		v := 0.0
		switch {
		case i == spikeIdx:
			v = 0.2 // 校准脉冲
		case trueT >= onsetT:
			v = (trueT - onsetT) * 50
			if v > 5 {
				v = 5
			}
		}
		pts[i] = model.Sample{RawT: rawT, V: v}
	}
	return pts
}
