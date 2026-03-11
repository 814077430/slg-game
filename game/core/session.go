package core

import (
	"sync"
	"time"

	"slg-game/config"
	"slg-game/database"
	"slg-game/log"
	"slg-game/network"
)

type PlayerSession struct {
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
	playerMgr   *PlayerManager
	mutex       sync.RWMutex
}

func NewPlayerSession(conn *network.Connection, db database.DB, config *config.Config, playerMgr *PlayerManager) *PlayerSession {
	return &PlayerSession{
		connection: conn,
		db:         db,
		config:     config,
		lastActive: time.Now(),
		playerMgr:  playerMgr,
	}
}

func (ps *PlayerSession) GetPlayerID() uint64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.playerID
}

func (ps *PlayerSession) SetPlayerID(playerID uint64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.playerID = playerID
}

func (ps *PlayerSession) GetUsername() string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.username
}

func (ps *PlayerSession) SetUsername(username string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.username = username
}

func (ps *PlayerSession) IsLoggedIn() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.isLoggedIn
}

func (ps *PlayerSession) SetLoggedIn(loggedIn bool) {
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

func (ps *PlayerSession) SendPacket(packet *network.Packet) error {
	return ps.connection.SendPacket(packet)
}

func (ps *PlayerSession) UpdateLastActive() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.lastActive = time.Now()
}

func (ps *PlayerSession) GetSessionDuration() time.Duration {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	if ps.loginTime.IsZero() {
		return 0
	}
	return time.Since(ps.loginTime)
}

// GetPosition 获取玩家位置
func (ps *PlayerSession) GetPosition() (int32, int32) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.x, ps.y
}

// SetPosition 设置玩家位置
func (ps *PlayerSession) SetPosition(x, y int32) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.x = x
	ps.y = y
	if ps.playerMgr != nil {
		ps.playerMgr.UpdatePlayerPosition(ps.playerID, x, y)
	}
}

// Cleanup 清理会话资源
func (ps *PlayerSession) Cleanup() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.isLoggedIn && ps.playerID > 0 {
		duration := time.Since(ps.loginTime)
		log.WithFields(map[string]interface{}{
			"player_id": ps.playerID,
			"username":  ps.username,
			"duration":  duration.String(),
		}).Info("Cleaning up session")

		// 从玩家管理器移除
		if ps.playerMgr != nil {
			ps.playerMgr.RemovePlayer(ps.playerID)
		}
	}
}
