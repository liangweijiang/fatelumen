package hash

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// CalcChartHash 计算 chart_hash = sha256(normalized birth info)。
// 命中 charts 表即复用，避免重复计算。
func CalcChartHash(gender int8, calendarType int8, year, month, day, hour, minute int, isLeap bool, timezone string) string {
	raw := fmt.Sprintf("%d|%d|%d|%d|%d|%d|%d|%t|%s",
		gender, calendarType, year, month, day, hour, minute, isLeap, timezone)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

// HashPassword 用 bcrypt 对明文密码做哈希（默认 cost）。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword 校验明文密码与哈希是否匹配。
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
