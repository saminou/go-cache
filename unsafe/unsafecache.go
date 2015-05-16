// This package provide a simple memory k/v cache and LRU cache.
// It based on the k/v cache implementation in go-cache:
// https://github.com/pmylund/go-cache
// and LRU cache implementation in groupcache:
// https://github.com/golang/groupcache/tree/master/lru
package unsafecache

import (
	"container/list"
	"errors"
	"fmt"
	"time"
)

type EXPLRUCache interface {
	Get(interface{}) (interface{}, bool)
	DumpKeys() []interface{}
}

// Cache is a goroutine-safe K/V cache.
type Cache struct {
	items             map[interface{}]*Item
	defaultExpiration time.Duration
}

type Item struct {
	Object     interface{}
	Expiration *time.Time
}

// Returns true if the item has expired.
func (item *Item) Expired() bool {
	if item.Expiration == nil {
		return false
	}
	return item.Expiration.Before(time.Now())
}

// New create a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than 1, the items in the cache
// never expire (by default), and must be deleted manually. If the cleanup
// interval is less than one, expired items are not deleted from the cache
// before calling DeleteExpired.
func New(defaultExpiration, cleanInterval time.Duration) *Cache {
	c := &Cache{
		items:             map[interface{}]*Item{},
		defaultExpiration: defaultExpiration,
	}
	if cleanInterval > 0 {
		go func() {
			for {
				time.Sleep(cleanInterval)
				c.DeleteExpired()
			}
		}()
	}
	return c
}

func (c *Cache) DumpKeys() (keys []interface{}) {
	for k, _ := range c.items {
		if k != nil {
			keys = append(keys, k)
		}
	}
	return
}

// Get return an item or nil, and a bool indicating whether
// the key was found.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	item, ok := c.items[key]
	if !ok || item.Expired() {
		return nil, false
	}
	return item.Object, true
}

// Set add a new key or replace an exist key. If the dur is 0, we will
// use the defaultExpiration.
func (c *Cache) Set(key interface{}, val interface{}, dur time.Duration) {
	var t *time.Time
	if dur == 0 {
		dur = c.defaultExpiration
	}
	if dur > 0 {
		tmp := time.Now().Add(dur)
		t = &tmp
	}
	c.items[key] = &Item{
		Object:     val,
		Expiration: t,
	}
}

// Delete a key-value pair if the key is existed.
func (c *Cache) Delete(key interface{}) {
	delete(c.items, key)
}

// Delete all cache.
func (c *Cache) Flush() {
	c.items = map[interface{}]*Item{}
}

// Add a number to a key-value pair.
func (c *Cache) Increment(key interface{}, x int64) error {
	val, ok := c.items[key]
	if !ok || val.Expired() {
		return fmt.Errorf("Item %s not found", key)
	}
	switch val.Object.(type) {
	case int:
		val.Object = val.Object.(int) + int(x)
	case int8:
		val.Object = val.Object.(int8) + int8(x)
	case int16:
		val.Object = val.Object.(int16) + int16(x)
	case int32:
		val.Object = val.Object.(int32) + int32(x)
	case int64:
		val.Object = val.Object.(int64) + x
	case uint:
		val.Object = val.Object.(uint) + uint(x)
	case uint8:
		val.Object = val.Object.(uint8) + uint8(x)
	case uint16:
		val.Object = val.Object.(uint16) + uint16(x)
	case uint32:
		val.Object = val.Object.(uint32) + uint32(x)
	case uint64:
		val.Object = val.Object.(uint64) + uint64(x)
	case uintptr:
		val.Object = val.Object.(uintptr) + uintptr(x)
	default:
		return fmt.Errorf("The value type error")
	}
	return nil
}

// Sub a number to a key-value pair.
func (c *Cache) Decrement(key interface{}, x int64) error {
	val, ok := c.items[key]
	if !ok || val.Expired() {
		return fmt.Errorf("Item %s not found", key)
	}
	switch val.Object.(type) {
	case int:
		val.Object = val.Object.(int) - int(x)
	case int8:
		val.Object = val.Object.(int8) - int8(x)
	case int16:
		val.Object = val.Object.(int16) - int16(x)
	case int32:
		val.Object = val.Object.(int32) - int32(x)
	case int64:
		val.Object = val.Object.(int64) - x
	case uint:
		val.Object = val.Object.(uint) - uint(x)
	case uint8:
		val.Object = val.Object.(uint8) - uint8(x)
	case uint16:
		val.Object = val.Object.(uint16) - uint16(x)
	case uint32:
		val.Object = val.Object.(uint32) - uint32(x)
	case uint64:
		val.Object = val.Object.(uint64) - uint64(x)
	case uintptr:
		val.Object = val.Object.(uintptr) - uintptr(x)
	default:
		return fmt.Errorf("The value type error")
	}
	return nil
}

// Return the number of item in cache.
func (c *Cache) ItemCount() int {
	counts := len(c.items)
	return counts
}

// Delete all expired items.
func (c *Cache) DeleteExpired() {
	for k, v := range c.items {
		if v.Expired() {
			delete(c.items, k)
		}
	}
}

// The LRUCache is a goroutine-safe cache.
type LRUCache struct {
	maxEntries int
	items      map[interface{}]*list.Element
	cacheList  *list.List
}

type entry struct {
	key   interface{}
	value interface{}
}

// NewLRU create a LRUCache with max size. The size is 0 means no limit.
func NewLRU(size int) (*LRUCache, error) {
	if size < 0 {
		return nil, errors.New("The size of LRU Cache must no less than 0")
	}
	lru := &LRUCache{
		maxEntries: size,
		items:      make(map[interface{}]*list.Element, size),
		cacheList:  list.New(),
	}
	return lru, nil
}

func (c *LRUCache) DumpKeys() (keys []interface{}) {
	for k, _ := range c.items {
		if k != nil {
			keys = append(keys, k)
		}
	}
	return
}

// Add a new key-value pair to the LRUCache.
func (c *LRUCache) Add(key interface{}, value interface{}) {
	if ent, hit := c.items[key]; hit {
		c.cacheList.MoveToFront(ent)
		ent.Value.(*entry).value = value
		return
	}
	ent := &entry{
		key:   key,
		value: value,
	}
	entry := c.cacheList.PushFront(ent)
	c.items[key] = entry

	if c.maxEntries > 0 && c.cacheList.Len() > c.maxEntries {
		c.removeOldestElement()
	}
}

// Get a value from the LRUCache. And a bool indicating
// whether found or not.
func (c *LRUCache) Get(key interface{}) (interface{}, bool) {
	if ent, hit := c.items[key]; hit {
		c.cacheList.MoveToFront(ent)
		return ent.Value.(*entry).value, true
	}
	return nil, false
}

// Remove a key-value pair in LRUCache. If the key is not existed,
// nothing will happen.
func (c *LRUCache) Remove(key interface{}) {
	if ent, hit := c.items[key]; hit {
		c.removeElement(ent)
	}
}

// Return the number of key-value pair in LRUCache.
func (c *LRUCache) Len() int {
	length := c.cacheList.Len()
	return length
}

// Delete all entry in the LRUCache. But the max size will hold.
func (c *LRUCache) Clear() {
	c.cacheList = list.New()
	c.items = make(map[interface{}]*list.Element, c.maxEntries)
}

// Resize the max limit.
func (c *LRUCache) SetMaxEntries(max int) error {
	if max < 0 {
		return errors.New("The max limit of entryies must no less than 0")
	}
	c.maxEntries = max
	return nil
}

func (c *LRUCache) removeElement(e *list.Element) {
	c.cacheList.Remove(e)
	ent := e.Value.(*entry)
	delete(c.items, ent.key)
}

func (c *LRUCache) removeOldestElement() {
	ent := c.cacheList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}
