// Package cache 提供进程内 TTL 缓存，语义与上游 SaiAdmin 6.x 的 think-cache 保持一致
// （对应 CACHE_MODE=file）。接口按 tag 组织，便于后续替换为 Redis 实现。
package cache

import (
	"sync"
	"time"
)

type item struct {
	val     interface{}
	expires time.Time // 零值表示永不过期
	tags    []string
}

// Cache 进程内缓存
type Cache struct {
	mu    sync.RWMutex
	items map[string]item
}

var C = &Cache{items: make(map[string]item)}

// Set 写入并设置 TTL（秒）。ttl<=0 表示长期有效。
func (c *Cache) Set(key string, val interface{}, ttl int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it := item{val: val}
	if ttl > 0 {
		it.expires = time.Now().Add(time.Duration(ttl) * time.Second)
	}
	c.items[key] = it
}

// SetTagged 写入并关联标签，便于按 tag 批量清理
func (c *Cache) SetTagged(key string, val interface{}, ttl int, tags []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it := item{val: val, tags: tags}
	if ttl > 0 {
		it.expires = time.Now().Add(time.Duration(ttl) * time.Second)
	}
	c.items[key] = it
}

// Get 读取，未命中或已过期返回 nil
func (c *Cache) Get(key string) interface{} {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	if !it.expires.IsZero() && time.Now().After(it.expires) {
		c.Delete(key)
		return nil
	}
	return it.val
}

// GetString 读取字符串值
func (c *Cache) GetString(key string) (string, bool) {
	v := c.Get(key)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// Delete 删除单个键
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// ClearTag 清理带有指定标签的所有键
func (c *Cache) ClearTag(tag string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, it := range c.items {
		for _, t := range it.tags {
			if t == tag {
				delete(c.items, k)
				break
			}
		}
	}
}

// ClearTags 清理带有任一指定标签的所有键
func (c *Cache) ClearTags(tags []string) {
	for _, t := range tags {
		c.ClearTag(t)
	}
}

// Clear 清空全部缓存
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]item)
}

// Keys 返回当前未过期的键列表（供缓存管理页展示）
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	out := make([]string, 0, len(c.items))
	for k, it := range c.items {
		if !it.expires.IsZero() && now.After(it.expires) {
			continue
		}
		out = append(out, k)
	}
	return out
}

// Len 未过期键数量
func (c *Cache) Len() int { return len(c.Keys()) }
