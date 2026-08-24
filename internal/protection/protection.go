// Package protection 计算淬灭保护窗口，并判定是否超出磁体保护能力。
package protection

import "task211-quenchprop/internal/model"

// ComputeWindow 计算从淬灭起点传播到关键分段所需的保护窗口。
// onsetSegID：起点所在拓扑分段；criticalSegID：关键（最危险）分段；
// propSpeed：传播速度（米/秒）；topo：线圈拓扑；marginMs：安全裕度（毫秒）；
// maxProtectionMs：磁体保护系统最大可容忍窗口（毫秒），超出即 breach。
// 返回窗口（毫秒）与是否超限。
func ComputeWindow(onsetSegID, criticalSegID string, propSpeed float64, topo *model.CoilTopology, marginMs, maxProtectionMs float64) (windowMs float64, breach bool) {
	if propSpeed <= 0 {
		// 速度未知，无法给出正窗口，按超限处理以触发人工复核。
		return 0, true
	}
	d := topo.PathDistance(onsetSegID, criticalSegID)
	if d < 0 {
		// 关键分段不可达，无法评估，按超限处理。
		return 0, true
	}
	seconds := d / propSpeed
	windowMs = seconds*1000 + marginMs
	breach = windowMs > maxProtectionMs
	return windowMs, breach
}
