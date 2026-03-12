package chat

import (
	"log"
	"strings"
	"sync"
	"time"

	"slg-game/errors"
	"slg-game/messenger"
	"slg-game/network"
	pb "slg-game/protocol"
	"slg-game/protocol"
	sessionPkg "slg-game/session"
)

// PlayerManager 玩家管理器接口
type PlayerManager interface {
	GetSession(playerID uint64) sessionPkg.Session
	GetPlayerCount() int
	GetAllPlayers() []interface{}
}

// 聊天协议消息 ID
const (
	MsgID_C2S_ChatRequest   = 4010
	MsgID_S2C_ChatResponse  = 4011
	MsgID_S2C_ChatBroadcast = 4012
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
	Session sessionPkg.Session
	Message *ChatMessage
}

// NewChatManager 创建聊天管理器
func NewChatManager(playerMgr PlayerManager, messageBus *messenger.MessageBus) *ChatManager {
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
	// 获取所有在线玩家并广播
	players := cm.playerMgr.GetAllPlayers()
	log.Printf("[Chat] Broadcasting message from %s to %d players", msg.Username, len(players))
	
	// TODO: 实现广播逻辑，需要遍历玩家并发送消息
	// 这需要 world 模块提供视野内玩家列表
}

// SendChat 发送聊天消息
func (cm *ChatManager) SendChat(session sessionPkg.Session, content, channel string) {
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
	if len(cm.history) > cm.maxHistory {
		cm.history = cm.history[1:]
	}
}

// cleanupHistory 清理过期历史
func (cm *ChatManager) cleanupHistory() {
	// 简单实现：保留最近 50 条
	if len(cm.history) > cm.maxHistory {
		cm.history = cm.history[len(cm.history)-cm.maxHistory:]
	}
}

// ChatHandler 聊天协议处理器
type ChatHandler struct {
	chatMgr *ChatManager
}

// NewChatHandler 创建聊天协议处理器
func NewChatHandler(chatMgr *ChatManager) *ChatHandler {
	return &ChatHandler{
		chatMgr: chatMgr,
	}
}

// Handle 处理聊天协议消息
func (h *ChatHandler) Handle(sess sessionPkg.Session, packet *network.Packet) *network.Packet {
	switch packet.MsgID {
	case MsgID_C2S_ChatRequest:
		return h.handleChatRequest(sess, packet.Data)
	default:
		return nil
	}
}

// handleChatRequest 处理聊天消息
func (h *ChatHandler) handleChatRequest(sess sessionPkg.Session, data []byte) *network.Packet {
	if !sess.IsLoggedIn() {
		return createChatErrorResponse(errors.ErrNotLoggedInErr)
	}

	var content, channel string

	// 尝试解析为 Protobuf
	request := &pb.C2S_ChatRequest{}
	if err := protocol.Unmarshal(data, request); err == nil && request.Content != "" {
		content = request.Content
		channel = request.Channel
	} else {
		// 回退到文本解析（格式：channel content）
		text := strings.TrimSpace(string(data))
		parts := strings.SplitN(text, " ", 2)
		if len(parts) >= 2 {
			channel = parts[0]
			content = parts[1]
		} else if len(parts) == 1 {
			channel = "world"
			content = parts[0]
		}
	}

	// 验证消息内容
	if len(content) == 0 || len(content) > 500 {
		return createChatErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Message length must be 1-500 characters"))
	}

	// 发送到聊天管理器
	if channel == "" {
		channel = "world"
	}

	h.chatMgr.SendChat(sess, content, channel)

	// 返回成功响应
	response := &pb.S2C_ChatResponse{
		Success:   true,
		Message:   "Message sent",
		Timestamp: time.Now().UnixMilli(),
	}

	responseData, err := protocol.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal chat response: %v", err)
		return createChatErrorResponse(errors.ErrInternalErr)
	}

	return &network.Packet{
		MsgID: MsgID_S2C_ChatResponse,
		Data:  responseData,
	}
}

// createChatErrorResponse 创建聊天错误响应
func createChatErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &pb.S2C_ChatResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := protocol.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_ChatResponse,
			Data:  data,
		}
	}
	return nil
}
