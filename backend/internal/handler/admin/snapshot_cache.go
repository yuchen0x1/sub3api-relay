package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type snapshotCacheEntry struct {
	ETag      string
	Payload   any
	ExpiresAt time.Time
}

// snapshotCacheMaxEntries 限制单个缓存实例的条目数。
// key 由查询参数拼出（模型、粒度、时间范围等），恶意或好奇的请求能造出无限多的
// unique key，没有上限时 map 只增不减，形成内存泄漏。
const snapshotCacheMaxEntries = 1024

type snapshotCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]snapshotCacheEntry
	sf    singleflight.Group
}

type snapshotCacheLoadResult struct {
	Entry snapshotCacheEntry
	Hit   bool
}

func newSnapshotCache(ttl time.Duration) *snapshotCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &snapshotCache{
		ttl:   ttl,
		items: make(map[string]snapshotCacheEntry),
	}
}

func (c *snapshotCache) Get(key string) (snapshotCacheEntry, bool) {
	if c == nil || key == "" {
		return snapshotCacheEntry{}, false
	}
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return snapshotCacheEntry{}, false
	}
	if now.After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return snapshotCacheEntry{}, false
	}
	return entry, true
}

func (c *snapshotCache) Set(key string, payload any) snapshotCacheEntry {
	if c == nil {
		return snapshotCacheEntry{}
	}
	entry := snapshotCacheEntry{
		ETag:      buildETagFromAny(payload),
		Payload:   payload,
		ExpiresAt: time.Now().Add(c.ttl),
	}
	if key == "" {
		return entry
	}
	c.mu.Lock()
	c.evictLocked()
	c.items[key] = entry
	c.mu.Unlock()
	return entry
}

// evictLocked 在条目数达到上限时先清理过期项，仍超限则淘汰最早过期的条目。
// 调用方必须持有 c.mu 写锁。
func (c *snapshotCache) evictLocked() {
	if len(c.items) < snapshotCacheMaxEntries {
		return
	}
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
	if len(c.items) < snapshotCacheMaxEntries {
		return
	}
	oldestKey := ""
	var oldest time.Time
	for key, item := range c.items {
		if oldestKey == "" || item.ExpiresAt.Before(oldest) {
			oldestKey = key
			oldest = item.ExpiresAt
		}
	}
	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func (c *snapshotCache) GetOrLoad(key string, load func() (any, error)) (snapshotCacheEntry, bool, error) {
	if load == nil {
		return snapshotCacheEntry{}, false, nil
	}
	if entry, ok := c.Get(key); ok {
		return entry, true, nil
	}
	if c == nil || key == "" {
		payload, err := load()
		if err != nil {
			return snapshotCacheEntry{}, false, err
		}
		return c.Set(key, payload), false, nil
	}

	value, err, _ := c.sf.Do(key, func() (any, error) {
		if entry, ok := c.Get(key); ok {
			return snapshotCacheLoadResult{Entry: entry, Hit: true}, nil
		}
		payload, err := load()
		if err != nil {
			return nil, err
		}
		return snapshotCacheLoadResult{Entry: c.Set(key, payload), Hit: false}, nil
	})
	if err != nil {
		return snapshotCacheEntry{}, false, err
	}
	result, ok := value.(snapshotCacheLoadResult)
	if !ok {
		return snapshotCacheEntry{}, false, nil
	}
	return result.Entry, result.Hit, nil
}

func buildETagFromAny(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "\"" + hex.EncodeToString(sum[:]) + "\""
}

func parseBoolQueryWithDefault(raw string, def bool) bool {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return def
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
