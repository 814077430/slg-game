package messenger

import (
	"sync"
	"testing"
	"time"
)

func TestMessageBus(t *testing.T) {
	// 创建消息总线
	bus := NewMessageBus()
	defer bus.Stop()

	// 测试计数器
	var receivedCount int
	var mu sync.Mutex

	// 注册处理器
	bus.RegisterHandler(MsgPlayerMove, func(msg *Message) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
		t.Logf("Received move message: %+v", msg.Data)
	})

	// 发布消息
	for i := 0; i < 5; i++ {
		bus.Publish(MsgPlayerMove, "test", &PlayerMoveData{
			PlayerID: uint64(i),
			X:        int32(i * 10),
			Y:        int32(i * 10),
		})
	}

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if receivedCount != 5 {
		t.Errorf("Expected 5 messages, got %d", receivedCount)
	}
	mu.Unlock()

	t.Logf("Message bus test passed: %d messages received", receivedCount)
}

func TestMessageBusPointToPoint(t *testing.T) {
	bus := NewMessageBus()
	defer bus.Stop()

	// 注册线程队列
	queue := bus.RegisterQueue("world")

	// 发送点对点消息
	bus.Send(MsgPlayerMove, "game", "world", &PlayerMoveData{
		PlayerID: 123,
		X:        100,
		Y:        200,
	})

	// 接收消息
	select {
	case msg := <-queue:
		if data, ok := msg.Data.(*PlayerMoveData); ok {
			if data.PlayerID != 123 {
				t.Errorf("Expected playerID 123, got %d", data.PlayerID)
			}
			t.Logf("Point-to-point message received: %+v", data)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestMessageBusPriority(t *testing.T) {
	bus := NewMessageBus()
	defer bus.Stop()

	var priorityOrder []MessagePriority
	var mu sync.Mutex

	bus.RegisterHandler(MsgPlayerLogin, func(msg *Message) {
		mu.Lock()
		priorityOrder = append(priorityOrder, msg.Priority)
		mu.Unlock()
	})

	// 发送不同优先级的消息
	bus.PublishWithPriority(MsgPlayerLogin, PriorityLow, "test", nil)
	bus.PublishWithPriority(MsgPlayerLogin, PriorityUrgent, "test", nil)
	bus.PublishWithPriority(MsgPlayerLogin, PriorityNormal, "test", nil)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(priorityOrder) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(priorityOrder))
	}
	mu.Unlock()

	t.Logf("Priority test passed: %d messages received", len(priorityOrder))
}

func TestMessageBusStats(t *testing.T) {
	bus := NewMessageBus()
	defer bus.Stop()

	// 注册一些处理器
	bus.RegisterHandler(MsgPlayerMove, func(msg *Message) {})
	bus.RegisterHandler(MsgChatMessage, func(msg *Message) {})

	// 注册订阅者
	bus.RegisterSubscriber("subscriber1", MsgPlayerMove)
	bus.RegisterSubscriber("subscriber2", MsgChatMessage)

	// 注册队列
	bus.RegisterQueue("queue1")

	stats := bus.GetStats()

	t.Logf("Message bus stats: %+v", stats)

	if stats["total_handlers"].(int) != 2 {
		t.Errorf("Expected 2 handlers, got %d", stats["total_handlers"])
	}
	if stats["subscribers"].(int) != 2 {
		t.Errorf("Expected 2 subscribers, got %d", stats["subscribers"])
	}
	if stats["queues"].(int) != 1 {
		t.Errorf("Expected 1 queue, got %d", stats["queues"])
	}

	t.Log("Stats test passed")
}
