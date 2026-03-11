package chat

import (
	"log"
	"sync"
	"time"

	"slg-game/network"
)

// PlayerSession 玩家会话接口（避免循环导入）
type PlayerSession interface {
	IsLoggedIn() bool
	GetPlayerID() uint64
	GetUsername() string
	SendPacket(packet *network.Packet) error
}

// PlayerManager 玩家管理器接口
type PlayerManager interface {
	GetSession(playerID uint64) PlayerSession
	GetPlayerCount() int
}

const (
	MsgID_C2S_ChatRequest = 1010
	MsgID_S2C_ChatResponse = 2010
	MsgID_S2C_ChatBroadcast = 2011
)

// ChatMessage 聊天消息
type ChatMessage struct {
	PlayerID  uint64
	Username  string
	Content   string
	Timestamp int64
	Channel   string // "world" 全服 / "alliance" 联盟
}

// ChatManager 聊天管理器（独立线程）
type ChatManager struct {
	playerMgr   PlayerManager
	messageChan chan *ChatMessage
	clientChan  chan *ClientMessage
	stopChan    chan struct{}
	wg          sync.WaitGroup
	history     []*ChatMessage // 最近消息（内存，不持久化）
	maxHistory  int
}

// ClientMessage 客户端消息
type ClientMessage struct {
	Session PlayerSession
	Message *ChatMessage
}

// NewChatManager 创建聊天管理器
func NewChatManager(playerMgr PlayerManager) *ChatManager {
	return &ChatManager{
		playerMgr:   playerMgr,
		messageChan: make(chan *ChatMessage, 1000),
		clientChan:  make(chan *ClientMessage, 1000),
		stopChan:    make(chan struct{}),
		history:     make([]*ChatMessage, 0),
		maxHistory:  50, // 只保留最近 50 条
	}
}

// StartLoop 启动聊天循环（独立线程）
func (cm *ChatManager) StartLoop() {
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		log.Println("[Chat] Chat loop started")

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case msg := <-cm.clientChan:
				// 接收客户端消息
				cm.handleClientMessage(msg)
			case msg := <-cm.messageChan:
				// 广播消息
				cm.broadcastMessage(msg)
			case <-ticker.C:
				// 定期清理过期历史
				cm.cleanupHistory()
			case <-cm.stopChan:
				log.Println("[Chat] Chat loop stopping...")
				return
			}
		}
	}()
}

// StopLoop 停止聊天循环
func (cm *ChatManager) StopLoop() {
	close(cm.stopChan)
	cm.wg.Wait()
	log.Println("[Chat] Chat loop stopped")
}

// handleClientMessage 处理客户端消息
func (cm *ChatManager) handleClientMessage(clientMsg *ClientMessage) {
	msg := clientMsg.Message
	session := clientMsg.Session

	// 验证玩家是否在线
	if !session.IsLoggedIn() {
		log.Printf("[Chat] Player not logged in: %d", msg.PlayerID)
		return
	}

	// 添加到历史记录
	cm.addToHistory(msg)

	// 发送到广播队列
	cm.messageChan <- msg

	log.Printf("[Chat] [%s] %s: %s", msg.Channel, msg.Username, msg.Content)
}

// broadcastMessage 广播消息给所有在线玩家
func (cm *ChatManager) broadcastMessage(msg *ChatMessage) {
	// 获取所有在线玩家（通过接口）
	// 这里简化处理，实际需要通过接口获取
	log.Printf("[Chat] Broadcasting message from %s", msg.Username)
}

// SendChat 发送聊天消息
func (cm *ChatManager) SendChat(session PlayerSession, content, channel string) {
	playerID := session.GetPlayerID()
	username := session.GetUsername()

	msg := &ChatMessage{
		PlayerID:  playerID,
		Username:  username,
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
		Channel:   channel,
	}

	// 发送到客户端消息队列
	cm.clientChan <- &ClientMessage{
		Session: session,
		Message: msg,
	}
}

// GetHistory 获取聊天历史
func (cm *ChatManager) GetHistory() []*ChatMessage {
	return cm.history
}

// addToHistory 添加到历史记录
func (cm *ChatManager) addToHistory(msg *ChatMessage) {
	cm.history = append(cm.history, msg)

	// 限制历史记录数量
	if len(cm.history) > cm.maxHistory {
		cm.history = cm.history[1:]
	}
}

// cleanupHistory 清理过期历史
func (cm *ChatManager) cleanupHistory() {
	// 只保留最近 5 分钟的消息
	now := time.Now().UnixMilli()
	cutoff := now - 5*60*1000 // 5 分钟前

	newHistory := make([]*ChatMessage, 0)
	for _, msg := range cm.history {
		if msg.Timestamp > cutoff {
			newHistory = append(newHistory, msg)
		}
	}
	cm.history = newHistory
}

// GetOnlineCount 获取在线玩家数
func (cm *ChatManager) GetOnlineCount() int {
	return cm.playerMgr.GetPlayerCount()
}
