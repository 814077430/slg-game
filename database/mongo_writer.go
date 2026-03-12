package database

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// WriteOperation 写操作
type WriteOperation struct {
	Collection string
	Filter     map[string]interface{}
	Update     map[string]interface{}
	Doc        map[string]interface{} // 用于 Insert
	OpType     WriteOpType
}

// WriteOpType 写操作类型
type WriteOpType int

const (
	WriteOpUpdate WriteOpType = iota
	WriteOpInsert
)

// MongoWriter MongoDB 异步写入器（独立线程）
type MongoWriter struct {
	client     *mongo.Client
	db         *mongo.Database
	queue      chan *WriteOperation
	stopChan   chan struct{}
	wg         sync.WaitGroup
	batchSize  int           // 批量大小
	batchInterval time.Duration // 批量间隔
}

// NewMongoWriter 创建 MongoDB 异步写入器
func NewMongoWriter(client *mongo.Client, dbName string, batchSize int, batchInterval time.Duration) *MongoWriter {
	mw := &MongoWriter{
		client:        client,
		db:            client.Database(dbName),
		queue:         make(chan *WriteOperation, 10000),
		stopChan:      make(chan struct{}),
		batchSize:     batchSize,
		batchInterval: batchInterval,
	}

	// 启动独立写入线程
	mw.wg.Add(1)
	go mw.writeLoop()

	return mw
}

// writeLoop 写入循环（独立线程）
func (mw *MongoWriter) writeLoop() {
	defer mw.wg.Done()

	ticker := time.NewTicker(mw.batchInterval)
	defer ticker.Stop()

	batch := make([]*WriteOperation, 0, mw.batchSize)

	log.Println("[MongoWriter] MongoDB writer thread started")

	for {
		select {
		case op := <-mw.queue:
			batch = append(batch, op)

			// 达到批量大小，立即写入
			if len(batch) >= mw.batchSize {
				mw.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// 定时写入
			if len(batch) > 0 {
				mw.flushBatch(batch)
				batch = batch[:0]
			}

		case <-mw.stopChan:
			// 停止前写入剩余数据
			if len(batch) > 0 {
				mw.flushBatch(batch)
			}
			log.Println("[MongoWriter] MongoDB writer thread stopped")
			return
		}
	}
}

// flushBatch 批量写入数据库
func (mw *MongoWriter) flushBatch(batch []*WriteOperation) {
	if len(batch) == 0 {
		return
	}

	ctx := context.Background()

	// 按 Collection 分组
	collectionOps := make(map[string][]*WriteOperation)
	for _, op := range batch {
		collectionOps[op.Collection] = append(collectionOps[op.Collection], op)
	}

	// 批量写入每个 Collection
	for collName, ops := range collectionOps {
		collection := mw.db.Collection(collName)

		// 分组更新和插入
		var updates []mongo.WriteModel
		var inserts []interface{}

		for _, op := range ops {
			if op.OpType == WriteOpUpdate {
				update := bson.M{"$set": op.Update}
				model := mongo.NewUpdateOneModel().
					SetFilter(op.Filter).
					SetUpdate(update)
				updates = append(updates, model)
			} else if op.OpType == WriteOpInsert {
				inserts = append(inserts, op.Doc)
			}
		}

		// 批量更新
		if len(updates) > 0 {
			_, err := collection.BulkWrite(ctx, updates)
			if err != nil {
				log.Printf("[MongoWriter] Bulk update failed: %v", err)
			}
		}

		// 批量插入
		if len(inserts) > 0 {
			_, err := collection.InsertMany(ctx, inserts)
			if err != nil {
				log.Printf("[MongoWriter] Bulk insert failed: %v", err)
			}
		}
	}

	log.Printf("[MongoWriter] Flushed %d operations", len(batch))
}

// UpdateOne 异步更新（不阻塞）
func (mw *MongoWriter) UpdateOne(collection string, filter, update map[string]interface{}) {
	select {
	case mw.queue <- &WriteOperation{
		Collection: collection,
		Filter:     filter,
		Update:     update,
		OpType:     WriteOpUpdate,
	}:
		// 成功加入队列
	default:
		// 队列已满，降级为同步写入
		log.Printf("[MongoWriter] Queue full, falling back to sync write")
		mw.syncUpdateOne(collection, filter, update)
	}
}

// InsertOne 异步插入（不阻塞）
func (mw *MongoWriter) InsertOne(collection string, doc map[string]interface{}) {
	select {
	case mw.queue <- &WriteOperation{
		Collection: collection,
		Doc:        doc,
		OpType:     WriteOpInsert,
	}:
		// 成功加入队列
	default:
		// 队列已满，降级为同步写入
		log.Printf("[MongoWriter] Queue full, falling back to sync insert")
		mw.syncInsertOne(collection, doc)
	}
}

// syncUpdateOne 同步更新（降级用）
func (mw *MongoWriter) syncUpdateOne(collection string, filter, update map[string]interface{}) {
	ctx := context.Background()
	coll := mw.db.Collection(collection)
	updateDoc := bson.M{"$set": update}
	coll.UpdateOne(ctx, filter, updateDoc)
}

// syncInsertOne 同步插入（降级用）
func (mw *MongoWriter) syncInsertOne(collection string, doc map[string]interface{}) {
	ctx := context.Background()
	coll := mw.db.Collection(collection)
	coll.InsertOne(ctx, doc)
}

// Stop 停止写入器
func (mw *MongoWriter) Stop() {
	close(mw.stopChan)
	mw.wg.Wait()
}

// QueueSize 获取队列大小
func (mw *MongoWriter) QueueSize() int {
	return len(mw.queue)
}
