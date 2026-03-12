package database

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// BatchWriter 批量写入器
type BatchWriter struct {
	collection *mongo.Collection
	queue      chan *writeOp
	ticker     *time.Ticker
	stopChan   chan struct{}
	wg         sync.WaitGroup
	maxSize    int           // 最大批量大小
	maxWait    time.Duration // 最大等待时间
}

type writeOp struct {
	filter map[string]interface{}
	update map[string]interface{}
	result chan error
}

// NewBatchWriter 创建批量写入器
func NewBatchWriter(collection *mongo.Collection, maxSize int, maxWait time.Duration) *BatchWriter {
	bw := &BatchWriter{
		collection: collection,
		queue:      make(chan *writeOp, 10000),
		ticker:     time.NewTicker(maxWait),
		stopChan:   make(chan struct{}),
		maxSize:    maxSize,
		maxWait:    maxWait,
	}
	
	// 启动批量写入协程
	bw.wg.Add(1)
	go bw.batchLoop()
	
	return bw
}

// UpdateOne 添加更新操作到队列
func (bw *BatchWriter) UpdateOne(filter, update map[string]interface{}) error {
	op := &writeOp{
		filter: filter,
		update: update,
		result: make(chan error, 1),
	}
	
	bw.queue <- op
	return <-op.result
}

// batchLoop 批量写入循环
func (bw *BatchWriter) batchLoop() {
	defer bw.wg.Done()
	
	var batch []*writeOp
	ctx := context.Background()
	
	for {
		select {
		case op := <-bw.queue:
			batch = append(batch, op)
			
			// 达到批量大小，立即写入
			if len(batch) >= bw.maxSize {
				bw.flushBatch(ctx, batch)
				batch = nil
			}
			
		case <-bw.ticker.C:
			// 定时写入
			if len(batch) > 0 {
				bw.flushBatch(ctx, batch)
				batch = nil
			}
			
		case <-bw.stopChan:
			// 停止前写入剩余数据
			if len(batch) > 0 {
				bw.flushBatch(ctx, batch)
			}
			return
		}
	}
}

// flushBatch 批量写入数据库
func (bw *BatchWriter) flushBatch(ctx context.Context, batch []*writeOp) {
	if len(batch) == 0 {
		return
	}
	
	// 构建批量操作
	models := make([]mongo.WriteModel, 0, len(batch))
	for _, op := range batch {
		model := mongo.NewUpdateOneModel().
			SetFilter(op.filter).
			SetUpdate(bson.M{"$set": op.update})
		models = append(models, model)
	}
	
	// 批量执行
	_, err := bw.collection.BulkWrite(ctx, models)
	
	// 通知所有操作完成
	for _, op := range batch {
		op.result <- err
	}
}

// Stop 停止批量写入器
func (bw *BatchWriter) Stop() {
	close(bw.stopChan)
	bw.ticker.Stop()
	bw.wg.Wait()
}

// Stats 获取统计信息
func (bw *BatchWriter) Stats() (queueSize int, isRunning bool) {
	select {
	case <-bw.stopChan:
		return len(bw.queue), false
	default:
		return len(bw.queue), true
	}
}
