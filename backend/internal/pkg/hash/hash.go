package hash

import (
	"crypto/sha256"
	"fmt"
)

// CalcChartHash 计算 chart_hash = sha256(normalized birth info)。
// 命中 charts 表即复用，避免重复计算。
func CalcChartHash(gender int8, calendarType int8, year, month, day, hour, minute int, isLeap bool, timezone string) string {
	raw := fmt.Sprintf("%d|%d|%d|%d|%d|%d|%d|%t|%s",
		gender, calendarType, year, month, day, hour, minute, isLeap, timezone)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}
