package infrastructure

import (
	"katydid-common-account/pkg/validator/v6/core"
	"sync"
	"sync/atomic"
)

// ============================================================================
// 简单缓存实现 - 无淘汰策略
// ============================================================================

// simpleCache 简单缓存实现
type simpleCache struct {
	data   sync.Map
	hits   atomic.Int64
	misses atomic.Int64
}

// NewSimpleCache 创建简单缓存
func NewSimpleCache() core.CacheManager {
	return &simpleCache{}
}

// Get 获取缓存
func (c *simpleCache) Get(key any) (any, bool) {
	val, ok := c.data.Load(key)
	if ok {
		c.hits.Add(1)
	} else {
		c.misses.Add(1)
	}
	return val, ok
}

// Set 设置缓存
func (c *simpleCache) Set(key, value any) {
	c.data.Store(key, value)
}

// Delete 删除缓存
func (c *simpleCache) Delete(key any) {
	c.data.Delete(key)
}

// Clear 清空缓存
func (c *simpleCache) Clear() {
	c.data.Range(func(key, value any) bool {
		c.data.Delete(key)
		return true
	})
}

// Stats 获取统计信息
func (c *simpleCache) Stats() core.CacheStats {
	size := 0
	c.data.Range(func(_, _ any) bool {
		size++
		return true
	})

	return core.CacheStats{
		Hits:   c.hits.Load(),
		Misses: c.misses.Load(),
		Size:   size,
	}
}

// ============================================================================
// LRU 缓存实现
// ============================================================================

// lruCache LRU 缓存实现
type lruCache struct {
	mu       sync.RWMutex
	data     map[any]*lruNode
	head     *lruNode
	tail     *lruNode
	maxSize  int
	hits     atomic.Int64
	misses   atomic.Int64
}

// lruNode LRU 链表节点
type lruNode struct {
	key   any
	value any
	prev  *lruNode
	next  *lruNode
}

// NewLRUCache 创建 LRU 缓存
func NewLRUCache(maxSize int) core.CacheManager {
	if maxSize <= 0 {
		maxSize = 1000
	}

	cache := &lruCache{
		data:    make(map[any]*lruNode),
		maxSize: maxSize,
	}

	// 初始化哨兵节点
	cache.head = &lruNode{}
	cache.tail = &lruNode{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	return cache
}

// Get 获取缓存
func (c *lruCache) Get(key any) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.data[key]; ok {
		// 移动到头部
		c.moveToHead(node)
		c.hits.Add(1)
		return node.value, true
	}

	c.misses.Add(1)
	return nil, false
}

// Set 设置缓存
func (c *lruCache) Set(key, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.data[key]; ok {
		// 更新值并移动到头部
		node.value = value
		c.moveToHead(node)
		return
	}

	// 创建新节点
	node := &lruNode{
		key:   key,
		value: value,
	}

	c.data[key] = node
	c.addToHead(node)

	// 检查是否超过容量
	if len(c.data) > c.maxSize {
		// 删除尾部节点
		removed := c.removeTail()
		delete(c.data, removed.key)
	}
}

// Delete 删除缓存
func (c *lruCache) Delete(key any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.data[key]; ok {
		c.removeNode(node)
		delete(c.data, key)
	}
}

// Clear 清空缓存
func (c *lruCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[any]*lruNode)
	c.head.next = c.tail
	c.tail.prev = c.head
}

// Stats 获取统计信息
func (c *lruCache) Stats() core.CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return core.CacheStats{
		Hits:   c.hits.Load(),
		Misses: c.misses.Load(),
		Size:   len(c.data),
	}
}

// moveToHead 移动节点到头部
func (c *lruCache) moveToHead(node *lruNode) {
	c.removeNode(node)
	c.addToHead(node)
}

// addToHead 添加节点到头部
func (c *lruCache) addToHead(node *lruNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

// removeNode 删除节点
func (c *lruCache) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// removeTail 删除尾部节点
func (c *lruCache) removeTail() *lruNode {
	node := c.tail.prev
	c.removeNode(node)
	return node
}

// ============================================================================
// 无缓存实现 - 用于禁用缓存
// ============================================================================

// noCache 无缓存实现
type noCache struct{}

// NewNoCache 创建无缓存
func NewNoCache() core.CacheManager {
	return &noCache{}
}

// Get 获取缓存（总是返回 false）
func (c *noCache) Get(key any) (any, bool) {
	return nil, false
}

// Set 设置缓存（什么都不做）
func (c *noCache) Set(key, value any) {
	// 不缓存
}

// Delete 删除缓存（什么都不做）
func (c *noCache) Delete(key any) {
	// 不缓存
}

// Clear 清空缓存（什么都不做）
func (c *noCache) Clear() {
	// 不缓存
}

// Stats 获取统计信息
func (c *noCache) Stats() core.CacheStats {
	return core.CacheStats{
		Hits:   0,
		Misses: 0,
		Size:   0,
	}
}
