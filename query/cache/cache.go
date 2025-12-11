// Package cache provides query result caching functionality.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// CacheEntry represents a cached query result
type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	AccessTime time.Time
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

// Cache provides query result caching
type Cache interface {
	// Get retrieves a value from the cache
	Get(key string) (interface{}, bool)
	// Set stores a value in the cache with optional TTL
	Set(key string, value interface{}, ttl time.Duration)
	// Invalidate removes a specific key from the cache
	Invalidate(key string)
	// InvalidatePattern removes all keys matching a pattern (e.g., "table:*")
	InvalidatePattern(pattern string)
	// Clear removes all entries from the cache
	Clear()
	// GetStats returns cache statistics
	GetStats() Stats
}

// Stats represents cache statistics
type Stats struct {
	Hits       int64
	Misses     int64
	Size       int
	MaxSize    int
	Evictions  int64
	HitRate    float64
}

// LRUCache implements an LRU cache with TTL support
type LRUCache struct {
	mu          sync.RWMutex
	data        map[string]*cacheNode
	maxSize     int
	defaultTTL  time.Duration
	head        *cacheNode
	tail        *cacheNode
	stats       Stats
	evictions   int64
}

// cacheNode represents a node in the doubly-linked list for LRU
type cacheNode struct {
	key        string
	value      interface{}
	expiresAt  time.Time
	accessTime time.Time
	prev       *cacheNode
	next       *cacheNode
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(maxSize int, defaultTTL time.Duration) *LRUCache {
	cache := &LRUCache{
		data:       make(map[string]*cacheNode),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		stats:      Stats{MaxSize: maxSize},
	}
	return cache
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.data[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}

	// Check if expired
	if !node.expiresAt.IsZero() && time.Now().After(node.expiresAt) {
		c.removeNode(node)
		c.stats.Misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.moveToFront(node)
	node.accessTime = time.Now()

	c.stats.Hits++
	c.updateHitRate()
	return node.value, true
}

// Set stores a value in the cache
func (c *LRUCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use default TTL if not specified
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Check if key already exists
	if node, exists := c.data[key]; exists {
		// Update existing node
		node.value = value
		node.expiresAt = expiresAt
		node.accessTime = time.Now()
		c.moveToFront(node)
		return
	}

	// Create new node
	node := &cacheNode{
		key:        key,
		value:      value,
		expiresAt:  expiresAt,
		accessTime: time.Now(),
	}

	// Check if we need to evict
	if len(c.data) >= c.maxSize {
		c.evictLRU()
		c.evictions++
		c.stats.Evictions = c.evictions
	}

	// Add to front
	c.addToFront(node)
	c.data[key] = node
	c.stats.Size = len(c.data)
}

// Invalidate removes a specific key from the cache
func (c *LRUCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.data[key]; ok {
		c.removeNode(node)
		c.stats.Size = len(c.data)
	}
}

// InvalidatePattern removes all keys matching a pattern
// Pattern format: "prefix:*" or "*:suffix" or "*:middle:*"
func (c *LRUCache) InvalidatePattern(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toRemove []*cacheNode

	for key, node := range c.data {
		if matchesPattern(key, pattern) {
			toRemove = append(toRemove, node)
		}
	}

	for _, node := range toRemove {
		c.removeNode(node)
	}

	c.stats.Size = len(c.data)
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*cacheNode)
	c.head = nil
	c.tail = nil
	c.stats.Size = 0
	c.stats.Hits = 0
	c.stats.Misses = 0
	c.stats.Evictions = 0
	c.evictions = 0
	c.updateHitRate()
}

// GetStats returns cache statistics
func (c *LRUCache) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = len(c.data)
	stats.Evictions = c.evictions
	c.updateHitRate()
	stats.HitRate = c.stats.HitRate
	return stats
}

// addToFront adds a node to the front of the list
func (c *LRUCache) addToFront(node *cacheNode) {
	if c.head == nil {
		c.head = node
		c.tail = node
		return
	}

	node.next = c.head
	c.head.prev = node
	c.head = node
}

// moveToFront moves a node to the front of the list
func (c *LRUCache) moveToFront(node *cacheNode) {
	if node == c.head {
		return
	}

	c.removeNode(node)
	c.addToFront(node)
}

// removeNode removes a node from the list
func (c *LRUCache) removeNode(node *cacheNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}

	delete(c.data, node.key)
}

// evictLRU evicts the least recently used node
func (c *LRUCache) evictLRU() {
	if c.tail == nil {
		return
	}

	c.removeNode(c.tail)
}

// updateHitRate updates the hit rate statistic
func (c *LRUCache) updateHitRate() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.Hits) / float64(total) * 100
	}
}

// matchesPattern checks if a key matches a pattern
func matchesPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple pattern matching: supports "prefix:*", "*:suffix", "*:middle:*"
	parts := splitPattern(pattern)
	keyParts := splitKey(key)

	if len(parts) != len(keyParts) {
		return false
	}

	for i, part := range parts {
		if part != "*" && part != keyParts[i] {
			return false
		}
	}

	return true
}

// splitPattern splits a pattern into parts
func splitPattern(pattern string) []string {
	return strings.Split(pattern, ":")
}

// splitKey splits a cache key into parts (assuming format "table:sql:hash")
func splitKey(key string) []string {
	return strings.Split(key, ":")
}

// GenerateCacheKey generates a cache key from SQL query and arguments
func GenerateCacheKey(sql string, args []interface{}) string {
	// Create a hash of SQL + args
	hasher := sha256.New()
	hasher.Write([]byte(sql))
	
	// Add args to hash
	for _, arg := range args {
		hasher.Write([]byte(fmt.Sprintf("%v", arg)))
	}
	
	hash := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("query:%s:%s", sql[:min(len(sql), 50)], hash[:16])
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

