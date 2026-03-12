package messenger

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// MessageHandler 消息处理器
type MessageHandler func(msg *Message)

// MessageBus 消息总线
type MessageBus struct {
	handlers     map[MessageType][]MessageHandler  // 按类型注册的处理者
	subscribers  map[string]chan *Message          // 订阅者队列
	queues       map[string]chan *Message          // 线程队列
	mutex        sync.RWMutex
	msgID        uint64
	stopChan     chan struct{}
}

// NewMessageBus 创建消息总线
func NewMessageBus() *MessageBus {
	mb := &MessageBus{
		handlers:    make(map[MessageType][]MessageHandler),
		subscribers: make(map[string]chan *Message),
		queues:      make(map[string]chan *Message),
		stopChan:    make(chan struct{}),
	}

	// 启动广播协程
	go mb.broadcastLoop()

	log.Println("[MessageBus] Message bus started")
	return mb
}

// RegisterHandler 注册消息处理器（按类型）
func (mb *MessageBus) RegisterHandler(msgType MessageType, handler MessageHandler) {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	mb.handlers[msgType] = append(mb.handlers[msgType], handler)
	log.Printf("[MessageBus] Registered handler for message type %d", msgType)
}

// RegisterSubscriber 注册订阅者（接收特定类型的消息）
func (mb *MessageBus) RegisterSubscriber(id string, msgTypes ...MessageType) chan *Message {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	queue := make(chan *Message, 1000)
	mb.subscribers[id] = queue

	// 为每个消息类型注册
	for _, msgType := range msgTypes {
		mb.handlers[msgType] = append(mb.handlers[msgType], func(msg *Message) {
			select {
			case queue <- msg:
			default:
				log.Printf("[MessageBus] Queue full for subscriber %s", id)
			}
		})
	}

	log.Printf("[MessageBus] Registered subscriber %s for types %v", id, msgTypes)
	return queue
}

// RegisterQueue 注册线程队列（点对点通信）
func (mb *MessageBus) RegisterQueue(id string) chan *Message {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	queue := make(chan *Message, 1000)
	mb.queues[id] = queue

	log.Printf("[MessageBus] Registered queue for thread %s", id)
	return queue
}

// Publish 发布消息（广播给所有订阅者）
func (mb *MessageBus) Publish(msgType MessageType, from string, data interface{}) {
	msg := &Message{
		Type:      msgType,
		Priority:  PriorityNormal,
		From:      from,
		Timestamp: time.Now(),
		ID:        atomic.AddUint64(&mb.msgID, 1),
		Data:      data,
	}

	mb.mutex.RLock()
	handlers := mb.handlers[msgType]
	mb.mutex.RUnlock()

	// 调用所有处理器
	for _, handler := range handlers {
		go handler(msg)
	}
}

// PublishWithPriority 发布消息（带优先级）
func (mb *MessageBus) PublishWithPriority(msgType MessageType, priority MessagePriority, from string, data interface{}) {
	msg := &Message{
		Type:      msgType,
		Priority:  priority,
		From:      from,
		Timestamp: time.Now(),
		ID:        atomic.AddUint64(&mb.msgID, 1),
		Data:      data,
	}

	mb.mutex.RLock()
	handlers := mb.handlers[msgType]
	mb.mutex.RUnlock()

	// 按优先级处理
	for _, handler := range handlers {
		go handler(msg)
	}
}

// Send 发送消息到特定线程（点对点）
func (mb *MessageBus) Send(msgType MessageType, from, to string, data interface{}) {
	mb.mutex.RLock()
	queue, exists := mb.queues[to]
	mb.mutex.RUnlock()

	if !exists {
		log.Printf("[MessageBus] Queue not found for thread %s", to)
		return
	}

	msg := &Message{
		Type:      msgType,
		Priority:  PriorityNormal,
		From:      from,
		To:        to,
		Timestamp: time.Now(),
		ID:        atomic.AddUint64(&mb.msgID, 1),
		Data:      data,
	}

	select {
	case queue <- msg:
	default:
		log.Printf("[MessageBus] Queue full for thread %s", to)
	}
}

// broadcastLoop 广播循环
func (mb *MessageBus) broadcastLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 定期清理不活跃的订阅者
			mb.cleanup()
		case <-mb.stopChan:
			log.Println("[MessageBus] Message bus stopped")
			return
		}
	}
}

// cleanup 清理不活跃的订阅者
func (mb *MessageBus) cleanup() {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	for id, queue := range mb.subscribers {
		if len(queue) > 900 { // 队列超过 90% 容量
			log.Printf("[MessageBus] Subscriber %s queue almost full (%d/1000)", id, len(queue))
		}
	}
}

// Stop 停止消息总线
func (mb *MessageBus) Stop() {
	close(mb.stopChan)

	// 关闭所有队列
	mb.mutex.Lock()
	for _, queue := range mb.subscribers {
		close(queue)
	}
	for _, queue := range mb.queues {
		close(queue)
	}
	mb.mutex.Unlock()

	log.Println("[MessageBus] Message bus shutdown complete")
}

// GetStats 获取统计信息
func (mb *MessageBus) GetStats() map[string]interface{} {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()

	return map[string]interface{}{
		"total_handlers":  len(mb.handlers),
		"subscribers":     len(mb.subscribers),
		"queues":          len(mb.queues),
		"total_messages":  atomic.LoadUint64(&mb.msgID),
	}
}
