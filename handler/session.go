package handler

import (
	"sync"
	"time"

	"slg-game/config"
	"slg-game/database"
	"slg-game/log"
	"slg-game/network"
	"slg-game/session"
)

// sessionImpl session.Session 实现
type sessionImpl struct {
	connection  *network.Connection
	db          database.DB
	config      *config.Config
	playerID    uint64
	username    string
	isLoggedIn  bool
	loginTime   time.Time
	lastActive  time.Time
	x           int32
	y           int32
	playerMgr   PlayerManager
	mutex       sync.RWMutex
}

func NewPlayerSession(conn *network.Connection, db database.DB, config *config.Config, playerMgr PlayerManager) session.Session {
	return &sessionImpl{
		connection: conn,
		db:         db,
		config:     config,
		lastActive: time.Now(),
		playerMgr:  playerMgr,
	}
}

func (ps *sessionImpl) GetPlayerID() uint64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.playerID
}

func (ps *sessionImpl) SetPlayerID(playerID uint64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.playerID = playerID
}

func (ps *sessionImpl) GetUsername() string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.username
}

func (ps *sessionImpl) SetUsername(username string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.username = username
}

func (ps *sessionImpl) IsLoggedIn() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.isLoggedIn
}

func (ps *sessionImpl) SetLoggedIn(loggedIn bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if loggedIn && !ps.isLoggedIn {
		ps.loginTime = time.Now()
		// 添加到玩家管理器
		if ps.playerMgr != nil {
			ps.playerMgr.AddPlayer(ps.playerID, ps.username, ps)
		}
	}
	ps.isLoggedIn = loggedIn
	ps.lastActive = time.Now()
}

func (ps *sessionImpl) SendPacket(packet session.Packet) error {
	pkt, ok := packet.(*network.Packet)
	if !ok {
		return nil
	}
	return ps.connection.SendPacket(pkt)
}

func (ps *sessionImpl) UpdateLastActive() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.lastActive = time.Now()
}

func (ps *sessionImpl) GetSessionDuration() time.Duration {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	if ps.loginTime.IsZero() {
		return 0
	}
	return time.Since(ps.loginTime)
}

// GetPosition 获取玩家位置
func (ps *sessionImpl) GetPosition() (int32, int32) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.x, ps.y
}

// SetPosition 设置玩家位置
func (ps *sessionImpl) SetPosition(x, y int32) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.x = x
	ps.y = y
	if ps.playerMgr != nil {
		ps.playerMgr.UpdatePlayerPosition(ps.playerID, x, y)
	}
}

// Cleanup 清理会话资源
func (ps *sessionImpl) Cleanup() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.isLoggedIn && ps.playerID > 0 {
		duration := time.Since(ps.loginTime)
		log.WithFields(map[string]interface{}{
			"player_id": ps.playerID,
			"username":  ps.username,
			"duration":  duration,
		}).Info("Cleaning up session")

		// 从玩家管理器移除
		if ps.playerMgr != nil {
			ps.playerMgr.RemovePlayer(ps.playerID)
		}
	}
}
