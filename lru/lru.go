package lru

import (
	"container/list"
)

type Size int64

const (
	B  Size = iota + 1
	KB      = 1 << (10 * iota)
	MB
	GB
)

type LRU struct {
	maxBytes  int64                          // 最大容量，0 表示无限制
	OnEvicted func(key string, value Value) // 被淘汰时的回调方法
	ll        *list.List                    // 双向链表用来实现 LRU
	cache     map[string]*list.Element      // 快速访问到 list 的节点
	curBytes  int64                          // 当前占用容量
}

type Value interface {
	Len() int64
}

type entry struct {
	key   string
	value Value
}

func New(maxBytes int64, onEvicted func(string, Value)) *LRU {
	return &LRU{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *LRU) Get(key string) (value Value, exist bool) {
	if v, ok := c.cache[key]; ok {
		c.ll.MoveToFront(v)
		return v.Value.(*entry).value, true
	}
	return nil, false
}

func (c *LRU) Add(key string, value Value) {
	// key 已经存在，则更新 value
	if v, ok := c.cache[key]; ok {
		c.ll.MoveToFront(v)
		oldVal := v.Value.(*entry).value
		// 更新后，新的 value 大小可能大于（小于）旧的 value，相应的更新 curBytes 的值
		c.curBytes += value.Len() - oldVal.Len()
		v.Value.(*entry).value = value
		return
	}

	// key 不存在
	c.ll.PushFront(&entry{
		key:   key,
		value: value,
	})
	c.cache[key] = c.ll.Front()
	c.curBytes += int64(len(key)) + value.Len()

	for c.maxBytes != 0 && c.curBytes > c.maxBytes {
		c.RemoveOldest()
	}
}

func (c *LRU) RemoveOldest() {
	l := c.ll.Back()
	if l != nil {
		delete(c.cache, l.Value.(*entry).key)
		c.ll.Remove(l)
		lv := l.Value.(*entry)
		c.curBytes -= int64(len(lv.key)) + lv.value.Len()

		if c.OnEvicted != nil {
			c.OnEvicted(lv.key, lv.value)
		}
	}
}

func (c *LRU) Len() int {
	return c.ll.Len()
}
