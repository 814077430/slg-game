package database

import (
	"fmt"
	"sync"
)

// MemoryDB 简易内存数据库（用于测试）
type MemoryDB struct {
	data  map[string][]map[string]interface{}
	mutex sync.RWMutex
}

// NewMemoryDB 创建内存数据库
func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		data: make(map[string][]map[string]interface{}),
	}
}

func (m *MemoryDB) GetCollection(name string) *MemoryCollection {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if _, exists := m.data[name]; !exists {
		m.data[name] = make([]map[string]interface{}, 0)
	}
	
	return &MemoryCollection{
		name:  name,
		data:  m.data[name],
		mutex: &m.mutex,
	}
}

func (m *MemoryDB) Client() interface{} {
	return nil
}

func (m *MemoryDB) Disconnect() error {
	return nil
}

// MemoryCollection 内存集合
type MemoryCollection struct {
	name  string
	data  []map[string]interface{}
	mutex *sync.RWMutex
}

// InsertOne 插入文档
func (c *MemoryCollection) InsertOne(doc map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	id := uint64(len(c.data) + 1)
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
			return doc, nil
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
			for k, v := range update {
				doc[k] = v
			}
			c.data[i] = doc
			return nil
		}
	}
	
	return fmt.Errorf("document not found")
}

// CountDocuments 统计文档数
func (c *MemoryCollection) CountDocuments(filter map[string]interface{}) (int, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	count := 0
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
			results = append(results, doc)
		}
	}
	
	return results
}
