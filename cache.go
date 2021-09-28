package cache

import (
	"go-LRU/lru"
	"sync"
)

// cache 是并发安全的 lru
type cache struct {
	mu   sync.RWMutex
	lru  *lru.LRU
	size int64 // 缓存大小
}

func (c *cache) get(key string) (value *ByteView, exist bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lru == nil {
		return &ByteView{}, false
	}

	v, exist := c.lru.Get(key)
	if exist {
		return v.(*ByteView), true
	}

	return nil, false
}

func (c *cache) Add(key string, value *ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.size, nil)
	}

	c.lru.Add(key, value)
}
