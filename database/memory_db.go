package database

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// MemoryDB 内存数据库（测试/演示用）
type MemoryDB struct {
	collections map[string]*MemoryCollection
	mutex       sync.RWMutex
}

// MemoryCollection 内存集合
type MemoryCollection struct {
	name  string
	data  []map[string]interface{}
	mutex sync.RWMutex
	idGen uint64
}

// NewMemoryDB 创建内存数据库
func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		collections: make(map[string]*MemoryCollection),
	}
}

// GetCollection 获取集合
func (m *MemoryDB) GetCollection(name string) *MemoryCollection {
	m.mutex.RLock()
	collection, exists := m.collections[name]
	m.mutex.RUnlock()

	if exists {
		return collection
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 双重检查
	if collection, exists = m.collections[name]; exists {
		return collection
	}

	collection = &MemoryCollection{
		name:  name,
		data:  make([]map[string]interface{}, 0),
		idGen: 0,
	}
	m.collections[name] = collection
	return collection
}

// Client 返回 nil（内存模式没有 MongoDB 客户端）
func (m *MemoryDB) Client() interface{} {
	return nil
}

// Disconnect 无操作
func (m *MemoryDB) Disconnect() error {
	return nil
}

// InsertOne 插入文档
func (c *MemoryCollection) InsertOne(doc map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 生成自增 ID
	id := atomic.AddUint64(&c.idGen, 1)
	doc["_id"] = id
	doc["id"] = id

	c.data = append(c.data, doc)
	return nil
}

// FindOne 查找单个文档
func (c *MemoryCollection) FindOne(filter map[string]interface{}) (map[string]interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, doc := range c.data {
		match := true
		for k, v := range filter {
			if doc[k] != v {
				match = false
				break
			}
		}
		if match {
			// 返回副本
			result := make(map[string]interface{})
			for k, v := range doc {
				result[k] = v
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("document not found")
}

// UpdateOne 更新文档
func (c *MemoryCollection) UpdateOne(filter map[string]interface{}, update map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for i, doc := range c.data {
		match := true
		for k, v := range filter {
			if doc[k] != v {
				match = false
				break
			}
		}

		if match {
			// 应用更新
			for k, v := range update {
				c.data[i][k] = v
			}
			return nil
		}
	}

	return fmt.Errorf("document not found")
}

// CountDocuments 统计文档数
func (c *MemoryCollection) CountDocuments(filter map[string]interface{}) (int64, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	count := int64(0)
	for _, doc := range c.data {
		match := true
		for k, v := range filter {
			if doc[k] != v {
				match = false
				break
			}
		}
		if match {
			count++
		}
	}

	return count, nil
}

// FindAll 查找所有匹配的文档
func (c *MemoryCollection) FindAll(filter map[string]interface{}) []map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var results []map[string]interface{}
	for _, doc := range c.data {
		match := true
		for k, v := range filter {
			if doc[k] != v {
				match = false
				break
			}
		}
		if match {
			// 返回副本
			result := make(map[string]interface{})
			for k, v := range doc {
				result[k] = v
			}
			results = append(results, result)
		}
	}

	return results
}

// DeleteOne 删除文档
func (c *MemoryCollection) DeleteOne(filter map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for i, doc := range c.data {
		match := true
		for k, v := range filter {
			if doc[k] != v {
				match = false
				break
			}
		}
		if match {
			// 删除元素
			c.data = append(c.data[:i], c.data[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("document not found")
}

// GetAll 获取所有文档
func (c *MemoryCollection) GetAll() []map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	results := make([]map[string]interface{}, len(c.data))
	for i, doc := range c.data {
		result := make(map[string]interface{})
		for k, v := range doc {
			result[k] = v
		}
		results[i] = result
	}
	return results
}

// Clear 清空集合
func (c *MemoryCollection) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make([]map[string]interface{}, 0)
	c.idGen = 0
}

// Count 返回文档总数
func (c *MemoryCollection) Count() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return int64(len(c.data))
}
