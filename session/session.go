package session

// Packet 数据包接口（避免循环导入）
type Packet interface {
	GetMsgID() uint32
	GetData() []byte
}

// Session 玩家会话接口（定义在独立包中避免循环导入）
type Session interface {
	IsLoggedIn() bool
	GetPlayerID() uint64
	SetPlayerID(uint64)
	GetUsername() string
	SetUsername(string)
	SetLoggedIn(bool)
	SetPosition(x, y int32)
	SendPacket(packet Packet) error
	Cleanup()
}

// PlayerInfo 玩家信息
type PlayerInfo struct {
	ID       uint64
	Username string
	X        int32
	Y        int32
	Online   bool
}
