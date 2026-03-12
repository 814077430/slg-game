package database

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PlayerOperation 玩家操作类型
type PlayerOperationType int

const (
	PlayerOpCreate PlayerOperationType = iota
	PlayerOpUpdate
	PlayerOpDelete
)

// PlayerOperation 玩家操作
type PlayerOperation struct {
	Type     PlayerOperationType
	PlayerID uint64
	Data     map[string]interface{}
}

// MongoAsyncWriter MongoDB 异步写入器
type MongoAsyncWriter struct {
	client       *mongo.Client
	database     *mongo.Database
	opChan       chan *PlayerOperation
	stopChan     chan struct{}
	wg           sync.WaitGroup
	batchSize    int
	flushTimeout time.Duration
}

// NewMongoAsyncWriter 创建 MongoDB 异步写入器
func NewMongoAsyncWriter(uri, dbName string, batchSize int, flushTimeout time.Duration) (*MongoAsyncWriter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	writer := &MongoAsyncWriter{
		client:       client,
		database:     client.Database(dbName),
		opChan:       make(chan *PlayerOperation, 1000),
		stopChan:     make(chan struct{}),
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
	}

	// 启动写入线程
	writer.wg.Add(1)
	go writer.writeLoop()

	log.Printf("[MongoAsyncWriter] Connected to MongoDB: %s/%s", uri, dbName)
	return writer, nil
}

// writeLoop 写入循环
func (m *MongoAsyncWriter) writeLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.flushTimeout)
	defer ticker.Stop()

	batch := make([]*PlayerOperation, 0, m.batchSize)

	log.Printf("[MongoAsyncWriter] Write loop started (batch_size=%d, timeout=%v)", m.batchSize, m.flushTimeout)

	for {
		select {
		case op := <-m.opChan:
			batch = append(batch, op)

			// 达到批量大小，立即写入
			if len(batch) >= m.batchSize {
				m.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// 定时写入
			if len(batch) > 0 {
				m.flushBatch(batch)
				batch = batch[:0]
			}

		case <-m.stopChan:
			// 停止前写入剩余数据
			if len(batch) > 0 {
				m.flushBatch(batch)
			}
			log.Println("[MongoAsyncWriter] Write loop stopped")
			return
		}
	}
}

// flushBatch 批量写入
func (m *MongoAsyncWriter) flushBatch(batch []*PlayerOperation) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := m.database.Collection("players")

	// 分组处理
	creates := make([]interface{}, 0)
	updates := make([]mongo.WriteModel, 0)

	for _, op := range batch {
		switch op.Type {
		case PlayerOpCreate:
			creates = append(creates, op.Data)
		case PlayerOpUpdate:
			filter := bson.M{"player_id": op.PlayerID}
			update := bson.M{"$set": op.Data}
			updates = append(updates, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update))
		}
	}

	// 批量插入
	if len(creates) > 0 {
		_, err := collection.InsertMany(ctx, creates)
		if err != nil {
			log.Printf("[MongoAsyncWriter] Batch insert failed: %v", err)
		} else {
			log.Printf("[MongoAsyncWriter] Batch inserted %d players", len(creates))
		}
	}

	// 批量更新
	if len(updates) > 0 {
		_, err := collection.BulkWrite(ctx, updates)
		if err != nil {
			log.Printf("[MongoAsyncWriter] Bulk update failed: %v", err)
		} else {
			log.Printf("[MongoAsyncWriter] Bulk updated %d players", len(updates))
		}
	}
}

// AddPlayer 添加玩家（异步）
func (m *MongoAsyncWriter) AddPlayer(playerID uint64, data map[string]interface{}) {
	select {
	case m.opChan <- &PlayerOperation{
		Type:     PlayerOpCreate,
		PlayerID: playerID,
		Data:     data,
	}:
	default:
		log.Printf("[MongoAsyncWriter] Channel full, dropping player %d", playerID)
	}
}

// UpdatePlayer 更新玩家（异步）
func (m *MongoAsyncWriter) UpdatePlayer(playerID uint64, data map[string]interface{}) {
	select {
	case m.opChan <- &PlayerOperation{
		Type:     PlayerOpUpdate,
		PlayerID: playerID,
		Data:     data,
	}:
	default:
		log.Printf("[MongoAsyncWriter] Channel full, dropping update for player %d", playerID)
	}
}

// Stop 停止写入器
func (m *MongoAsyncWriter) Stop() {
	close(m.stopChan)
	m.wg.Wait()

	// 关闭 MongoDB 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.client.Disconnect(ctx); err != nil {
		log.Printf("[MongoAsyncWriter] Disconnect failed: %v", err)
	}

	log.Println("[MongoAsyncWriter] Stopped")
}
