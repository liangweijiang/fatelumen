package cache

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type memoryEntry struct {
	val       string
	expiresAt time.Time
}

func (e memoryEntry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

// MemoryCache 进程内内存缓存，带 TTL 过期清理。无 Redis 时兜底使用。
type MemoryCache struct {
	mu      sync.Mutex
	items   map[string]memoryEntry
	closeCh chan struct{}
	closeOnce sync.Once
}

// NewMemoryCache 创建内存缓存实例，启动后台过期清理协程。
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		items:   make(map[string]memoryEntry),
		closeCh: make(chan struct{}),
	}
	go mc.cleanupLoop()
	return mc
}

// Incr 原子递增 key 对应的计数器。key 不存在时返回 1。
func (m *MemoryCache) Incr(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.items[key]
	now := time.Now()
	if exists && !entry.expired(now) {
		v, err := strconv.ParseInt(entry.val, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cache incr: value not integer for key %s", key)
		}
		v++
		entry.val = strconv.FormatInt(v, 10)
		m.items[key] = entry
		return v, nil
	}

	m.items[key] = memoryEntry{val: "1"}
	return 1, nil
}

// Get 获取 key 对应的值，过期或不存在返回空字符串。
func (m *MemoryCache) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.items[key]
	if !exists || entry.expired(time.Now()) {
		return "", nil
	}
	return entry.val, nil
}

// Set 写入 key-val，设置 TTL。ttl=0 表示永不过期。
func (m *MemoryCache) Set(_ context.Context, key, val string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := memoryEntry{val: val}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	m.items[key] = entry
	return nil
}

// Close 停止后台清理协程。
func (m *MemoryCache) Close() {
	m.closeOnce.Do(func() {
		close(m.closeCh)
	})
}

func (m *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.closeCh:
			return
		case now := <-ticker.C:
			m.mu.Lock()
			for k, v := range m.items {
				if v.expired(now) {
					delete(m.items, k)
				}
			}
			m.mu.Unlock()
		}
	}
}
