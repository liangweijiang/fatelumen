package hash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// CanonicalJSONSHA256 hashes Go values using encoding/json's deterministic
// map-key ordering. Callers must normalize semantically unordered slices first.
func CanonicalJSONSHA256(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal canonical json: %w", err)
	}
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum), nil
}

// CalcChartHash 计算 chart_hash = sha256(normalized birth info)。
// 命中 charts 表即复用，避免重复计算。
func CalcChartHash(gender int8, calendarType int8, year, month, day, hour, minute int, isLeap bool, timezone string) string {
	raw := fmt.Sprintf("%d|%d|%d|%d|%d|%d|%d|%t|%s",
		gender, calendarType, year, month, day, hour, minute, isLeap, timezone)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

type ChartHashInput struct {
	Gender                         int8
	CalendarType                   int8
	Year, Month, Day, Hour, Minute int
	IsLeapMonth                    bool
	Latitude, Longitude            float64
	TimezoneID                     string
	HistoricalUTCOffset            int
	TimeCalculationMode            string
	DayBoundaryRule                string
	LocationVersion                string
	TimezoneVersion                string
	SolarAlgorithmVersion          string
	CalculatorVersion              string
}

func CalcChartHashV2(in ChartHashInput) string {
	raw := fmt.Sprintf("v2|%d|%d|%04d-%02d-%02dT%02d:%02d|%t|%.6f|%.6f|%s|%d|%s|%s|%s|%s|%s|%s",
		in.Gender, in.CalendarType, in.Year, in.Month, in.Day, in.Hour, in.Minute, in.IsLeapMonth,
		in.Latitude, in.Longitude, in.TimezoneID, in.HistoricalUTCOffset,
		in.TimeCalculationMode, in.DayBoundaryRule, in.LocationVersion, in.TimezoneVersion,
		in.SolarAlgorithmVersion, in.CalculatorVersion)
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
