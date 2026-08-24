package model

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID 生成带前缀的短随机标识，保证进程内唯一且可持久化。
func NewID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return prefix + "-x"
	}
	return prefix + "-" + hex.EncodeToString(b)
}
