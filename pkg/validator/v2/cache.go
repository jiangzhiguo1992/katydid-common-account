package v2

import (
	"sync"
)

// ============================================================================
// 缓存管理器实现 - 单一职责：规则缓存管理
// ============================================================================

// cacheKey 缓存键
type cacheKey struct {
	typeName string
	scene    Scene
}

// defaultCacheManager 默认缓存管理器
type defaultCacheManager struct {
	cache map[cacheKey]map[string]string
	mu    sync.RWMutex
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() CacheManager {
	return &defaultCacheManager{
		cache: make(map[cacheKey]map[string]string),
	}
}

// Get 获取缓存
func (c *defaultCacheManager) Get(key string, scene Scene) (map[string]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cKey := cacheKey{typeName: key, scene: scene}
	rules, ok := c.cache[cKey]
	return rules, ok
}

// Set 设置缓存
func (c *defaultCacheManager) Set(key string, scene Scene, rules map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cKey := cacheKey{typeName: key, scene: scene}
	// 深拷贝规则，避免外部修改
	copiedRules := make(map[string]string, len(rules))
	for k, v := range rules {
		copiedRules[k] = v
	}
	c.cache[cKey] = copiedRules
}

// Clear 清空缓存
func (c *defaultCacheManager) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[cacheKey]map[string]string)
}

// ============================================================================
// LRU 缓存管理器 - 带容量限制的缓存
// ============================================================================

// lruNode LRU节点
type lruNode struct {
	key   cacheKey
	value map[string]string
	prev  *lruNode
	next  *lruNode
}

// LRUCacheManager LRU缓存管理器
type LRUCacheManager struct {
	capacity int
	cache    map[cacheKey]*lruNode
	head     *lruNode // 最近使用
	tail     *lruNode // 最少使用
	mu       sync.RWMutex
}

// NewLRUCacheManager 创建LRU缓存管理器
func NewLRUCacheManager(capacity int) CacheManager {
	if capacity <= 0 {
		capacity = 100 // 默认容量
	}

	lru := &LRUCacheManager{
		capacity: capacity,
		cache:    make(map[cacheKey]*lruNode),
		head:     &lruNode{},
		tail:     &lruNode{},
	}
	lru.head.next = lru.tail
	lru.tail.prev = lru.head

	return lru
}

// Get 获取缓存
func (c *LRUCacheManager) Get(key string, scene Scene) (map[string]string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cKey := cacheKey{typeName: key, scene: scene}
	if node, ok := c.cache[cKey]; ok {
		c.moveToHead(node)
		return node.value, true
	}
	return nil, false
}

// Set 设置缓存
func (c *LRUCacheManager) Set(key string, scene Scene, rules map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cKey := cacheKey{typeName: key, scene: scene}

	// 深拷贝规则
	copiedRules := make(map[string]string, len(rules))
	for k, v := range rules {
		copiedRules[k] = v
	}

	if node, ok := c.cache[cKey]; ok {
		// 更新现有节点
		node.value = copiedRules
		c.moveToHead(node)
	} else {
		// 创建新节点
		node := &lruNode{
			key:   cKey,
			value: copiedRules,
		}
		c.cache[cKey] = node
		c.addToHead(node)

		// 检查容量
		if len(c.cache) > c.capacity {
			removed := c.removeTail()
			delete(c.cache, removed.key)
		}
	}
}

// Clear 清空缓存
func (c *LRUCacheManager) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[cacheKey]*lruNode)
	c.head.next = c.tail
	c.tail.prev = c.head
}

// moveToHead 移动节点到头部
func (c *LRUCacheManager) moveToHead(node *lruNode) {
	c.removeNode(node)
	c.addToHead(node)
}

// removeNode 移除节点
func (c *LRUCacheManager) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// addToHead 添加节点到头部
func (c *LRUCacheManager) addToHead(node *lruNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

// removeTail 移除尾部节点
func (c *LRUCacheManager) removeTail() *lruNode {
	node := c.tail.prev
	c.removeNode(node)
	return node
}
