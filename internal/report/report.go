// Package report 将传播分析结果组装为可发布的分析包。
package report

import (
	"fmt"
	"strings"

	"task211-quenchprop/internal/model"
)

// BuildPackage 依据传播分析结果与保护窗口构造一份分析包（初始为草稿状态）。
func BuildPackage(expID, onsetCh string, onsetTime, propSpeed float64, affected []string, windowMs float64, breach bool) *model.AnalysisPackage {
	summary := fmt.Sprintf(
		"淬灭起点通道=%s，起点校正时间=%.4fs，传播速度=%.3f m/s，受影响分段=%d，保护窗口=%.1f ms（%s）。",
		onsetCh, onsetTime, propSpeed, len(affected), windowMs,
		map[bool]string{true: "超出保护能力", false: "保护窗口内"}[breach],
	)
	return &model.AnalysisPackage{
		ID:                model.NewID("ana"),
		ExperimentID:      expID,
		Status:            model.AnaDraft,
		OnsetChannelID:    onsetCh,
		OnsetTime:         onsetTime,
		PropagationSpeed:  propSpeed,
		AffectedSegments:  affected,
		ProtectionWindowMs: windowMs,
		Breach:            breach,
		Summary:           summary,
	}
}

// AffectedSummary 生成受影响分段的简短可读摘要。
func AffectedSummary(segs []string) string {
	if len(segs) == 0 {
		return "(无)"
	}
	return strings.Join(segs, ", ")
}
