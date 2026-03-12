package messenger

import "time"

// MessageType 消息类型
type MessageType int

const (
	MsgUnknown MessageType = iota
	MsgPlayerMove          // 玩家移动
	MsgPlayerLogin         // 玩家登录
	MsgPlayerLogout        // 玩家登出
	MsgPlayerRegister      // 玩家注册
	MsgBattleStart         // 战斗开始
	MsgBattleEnd           // 战斗结束
	MsgChatMessage         // 聊天消息
	MsgWorldTick           // 世界 Tick
	MsgDataPersist         // 数据持久化
)

// MessagePriority 消息优先级
type MessagePriority int

const (
	PriorityLow MessagePriority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// Message 消息结构
type Message struct {
	Type      MessageType       // 消息类型
	Priority  MessagePriority   // 优先级
	From      string            // 发送者
	To        string            // 接收者（空表示广播）
	Data      interface{}       // 消息数据
	Timestamp time.Time         // 时间戳
	ID        uint64            // 消息 ID
}

// PlayerMoveData 玩家移动数据
type PlayerMoveData struct {
	PlayerID uint64
	X        int32
	Y        int32
}

// PlayerLoginData 玩家登录数据
type PlayerLoginData struct {
	PlayerID uint64
	Username string
}

// BattleStartData 战斗开始数据
type BattleStartData struct {
	BattleID    uint64
	AttackerID  uint64
	DefenderID  uint64
}

// BattleEndData 战斗结束数据
type BattleEndData struct {
	BattleID   uint64
	WinnerID   uint64
	LoserID    uint64
}

// ChatMessageData 聊天消息数据
type ChatMessageData struct {
	PlayerID  uint64
	Username  string
	Content   string
	Channel   string
}

// DataPersistData 数据持久化数据
type DataPersistData struct {
	PlayerID uint64
	Data     map[string]interface{}
}
