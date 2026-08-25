package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestBug07_CalibrationAdvancesWaveformVersion(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/version.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := NewApp(db, DefaultConfig())
	if err := app.AddTopology(&model.CoilTopology{ID: "topo-version"}); err != nil { t.Fatal(err) }
	exp, err := app.CreateExperiment(ExperimentInput{Name: "version", MagnetSN: "mag", Operator: "op", TopologyID: "topo-version", SampleRate: 1000})
	if err != nil { t.Fatal(err) }
	ch := &model.Channel{Index: 1, SegmentID: "s1", Status: model.ChanOnline}
	if err := app.AddChannel(exp.ID, ch); err != nil { t.Fatal(err) }
	points := []model.Sample{{RawT: 0, V: 0}, {RawT: 0.001, V: 0.2}, {RawT: 0.002, V: 0}}
	if _, err := app.IngestWaveform(exp.ID, ch.ID, 1000, points); err != nil { t.Fatal(err) }
	if _, err := app.Calibrate(exp.ID, ch.ID); err != nil { t.Fatal(err) }
	wfs, err := app.ListWaveforms(exp.ID)
	if err != nil { t.Fatal(err) }
	if len(wfs) != 1 || wfs[0].Version <= 1 {
		t.Fatalf("waveform version after calibration = %+v, want version > 1", wfs)
	}
}
