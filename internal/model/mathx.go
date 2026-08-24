package model

import "math"

// Clamp 将 x 限制在 [lo, hi]。
func Clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// Abs 返回绝对值。
func Abs(x float64) float64 { return math.Abs(x) }

// ApproxEqual 判断两浮点数是否在 eps 容差内相等。
func ApproxEqual(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

// Median 返回切片中位数（原地排序副本）。
func Median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := make([]float64, len(xs))
	copy(cp, xs)
	// 简单插入排序，避免引入 sort 依赖之外的复杂度
	for i := 1; i < len(cp); i++ {
		v := cp[i]
		j := i - 1
		for j >= 0 && cp[j] > v {
			cp[j+1] = cp[j]
			j--
		}
		cp[j+1] = v
	}
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}
