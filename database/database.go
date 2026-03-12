package database

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"slg-game/cache"
)

// DB 数据库接口
type DB interface {
	GetCollection(name string) Collection
	Disconnect() error
}

// Collection 集合接口
type Collection interface {
	FindOne(filter map[string]interface{}) (map[string]interface{}, error)
	InsertOne(doc map[string]interface{}) error
	UpdateOne(filter, update map[string]interface{}) error
	CountDocuments(filter map[string]interface{}) (int64, error)
	GetAll() []map[string]interface{}
}

// MemoryDB 内存数据库实现
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

// MongoDatabase MongoDB 实现
type MongoDatabase struct {
	client     *mongo.Client
	db         *mongo.Database
	collections map[string]*MongoCollection
	cache      *cache.PlayerCache
	mutex      sync.RWMutex
}

// MongoCollection MongoDB 集合实现
type MongoCollection struct {
	collection *mongo.Collection
	cache      *cache.PlayerCache
}

// CachedDatabase 带缓存的数据库（包装 MongoDatabase）
type CachedDatabase struct {
	mongo *MongoDatabase
	cache *cache.PlayerCache
}

// NewMemoryDB 创建内存数据库
func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		collections: make(map[string]*MemoryCollection),
	}
}

func (m *MemoryDB) GetCollection(name string) Collection {
	m.mutex.RLock()
	collection, exists := m.collections[name]
	m.mutex.RUnlock()

	if exists {
		return collection
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

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

func (m *MemoryDB) Disconnect() error {
	return nil
}

// InitMongoDB 初始化 MongoDB 连接
func InitMongoDB(uri, dbName string) (*MongoDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err = client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	return &MongoDatabase{
		client:      client,
		db:          client.Database(dbName),
		collections: make(map[string]*MongoCollection),
	}, nil
}

// InitMongoDBWithCache 初始化 MongoDB 连接并启用 Redis 缓存
func InitMongoDBWithCache(mongoURI, dbName, redisAddr, redisPassword string, redisDB int) (*CachedDatabase, error) {
	// 初始化 MongoDB
	mongoDB, err := InitMongoDB(mongoURI, dbName)
	if err != nil {
		return nil, err
	}

	// 初始化 Redis 缓存
	playerCache := cache.NewPlayerCache(redisAddr, redisPassword, redisDB)

	// 测试 Redis 连接
	if err := playerCache.Ping(); err != nil {
		// Redis 不可用，回退到纯 MongoDB
		return nil, err
	}

	return &CachedDatabase{
		mongo: mongoDB,
		cache: playerCache,
	}, nil
}

func (m *MongoDatabase) GetCollection(name string) Collection {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if collection, exists := m.collections[name]; exists {
		return collection
	}

	collection := &MongoCollection{
		collection: m.db.Collection(name),
		cache:      nil, // 不使用缓存
	}
	m.collections[name] = collection
	return collection
}

func (m *MongoDatabase) Disconnect() error {
	if m.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

// GetCollection 获取集合（带缓存）
func (c *CachedDatabase) GetCollection(name string) Collection {
	if name == "players" {
		// 玩家集合使用缓存
		return &MongoCollection{
			collection: c.mongo.db.Collection(name),
			cache:      c.cache,
		}
	}
	// 其他集合不使用缓存
	return c.mongo.GetCollection(name)
}

func (c *CachedDatabase) Disconnect() error {
	if c.cache != nil {
		c.cache.Close()
	}
	return c.mongo.Disconnect()
}

// MemoryCollection 方法实现
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
			result := make(map[string]interface{})
			for k, v := range doc {
				result[k] = v
			}
			return result, nil
		}
	}
	return nil, nil
}

func (c *MemoryCollection) InsertOne(doc map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.idGen++
	doc["_id"] = c.idGen
	doc["id"] = c.idGen
	c.data = append(c.data, doc)
	return nil
}

func (c *MemoryCollection) UpdateOne(filter, update map[string]interface{}) error {
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
				c.data[i][k] = v
			}
			return nil
		}
	}
	return nil
}

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

// MongoCollection 方法实现
func (c *MongoCollection) FindOne(filter map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.collection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *MongoCollection) InsertOne(doc map[string]interface{}) error {
	_, err := c.collection.InsertOne(context.Background(), doc)
	return err
}

func (c *MongoCollection) UpdateOne(filter, update map[string]interface{}) error {
	_, err := c.collection.UpdateOne(context.Background(), filter, bson.M{"$set": update})
	return err
}

func (c *MongoCollection) CountDocuments(filter map[string]interface{}) (int64, error) {
	return c.collection.CountDocuments(context.Background(), filter)
}

func (c *MongoCollection) GetAll() []map[string]interface{} {
	cursor, err := c.collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil
	}
	var results []map[string]interface{}
	if err := cursor.All(context.Background(), &results); err != nil {
		return nil
	}
	return results
}
