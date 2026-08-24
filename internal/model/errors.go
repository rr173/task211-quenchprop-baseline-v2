// Package model 定义低温超导磁体淬灭传播分析服务的领域实体、状态机与错误。
package model

import "errors"

// 领域错误集合。
var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict: version mismatch")
	ErrInvalidState     = errors.New("invalid state transition")
	ErrInsufficientData = errors.New("insufficient data")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrBadInput         = errors.New("bad input")
	ErrSealed           = errors.New("experiment sealed: write rejected")
	ErrUnknownChannel   = errors.New("unknown or disabled channel")
	ErrRateMismatch     = errors.New("sample rate mismatch with existing waveform")
	ErrTimeAxisBroken   = errors.New("time axis not monotonic or broken")
	ErrDuplicate        = errors.New("duplicate fingerprint")
)
