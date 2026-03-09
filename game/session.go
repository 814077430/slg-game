package game

import (
	"sync"

	"slg-game/config"
	"slg-game/database"
	"slg-game/network"
)

type PlayerSession struct {
	connection *network.Connection
	db         *database.Database
	config     *config.Config
	playerID   uint64
	username   string
	isLoggedIn bool
	mutex      sync.RWMutex
}

func NewPlayerSession(conn *network.Connection, db *database.Database, config *config.Config) *PlayerSession {
	return &PlayerSession{
		connection: conn,
		db:         db,
		config:     config,
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
	ps.isLoggedIn = loggedIn
}

func (ps *PlayerSession) SendPacket(packet *network.Packet) error {
	return ps.connection.SendPacket(packet)
}

func (ps *PlayerSession) Cleanup() {
	// 清理会话资源
	// 例如：保存玩家数据、从在线列表移除等
}