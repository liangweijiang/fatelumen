package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"fatelumen/backend/internal/cache"
	"fatelumen/backend/internal/pkg/logger"
)

const DefaultMaxDailyQuota = 3

// ErrQuotaExceeded 每日免费额度已用完。handler 映射为 code=4290。
var ErrQuotaExceeded = fmt.Errorf("daily free quota exceeded")

// QuotaService 每日免费额度服务。依赖 Cache 接口（P5）。
type QuotaService struct {
	cache    cache.Cache
	maxDaily int
	log      *logger.Logger
}

// NewQuotaService 创建额度服务。maxDaily 为 0 时使用默认值 3。
func NewQuotaService(c cache.Cache, maxDaily int, log *logger.Logger) *QuotaService {
	if maxDaily <= 0 {
		maxDaily = DefaultMaxDailyQuota
	}
	return &QuotaService{
		cache:    c,
		maxDaily: maxDaily,
		log:      log,
	}
}

// CheckAndConsume 检查并消耗一次免费额度。超额返回 ErrQuotaExceeded。
func (s *QuotaService) CheckAndConsume(ctx context.Context, userID uint64) error {
	now := time.Now().UTC()
	key := fmt.Sprintf("quota:%d:%s", userID, now.Format("2006-01-02"))

	count, err := s.cache.Incr(ctx, key)
	if err != nil {
		return fmt.Errorf("quota incr: %w", err)
	}

	if count == 1 {
		ttl := ttlToEndOfUTCDay(now)
		if err := s.cache.Set(ctx, key, strconv.FormatInt(count, 10), ttl); err != nil {
			s.log.Warn("quota set ttl failed", "key", key, "err", err)
		}
	}

	if count > int64(s.maxDaily) {
		return ErrQuotaExceeded
	}
	return nil
}

// ttlToEndOfUTCDay 计算到 UTC 当天结束的时间。
func ttlToEndOfUTCDay(now time.Time) time.Duration {
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
	return endOfDay.Sub(now) + time.Second
}
