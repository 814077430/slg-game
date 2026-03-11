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
	db          *database.MemoryDB
	config      *config.Config
	playerID    uint64
	username    string
	isLoggedIn  bool
	loginTime   time.Time
	lastActive  time.Time
	mutex       sync.RWMutex
}

func NewPlayerSession(conn *network.Connection, db *database.MemoryDB, config *config.Config) *PlayerSession {
	return &PlayerSession{
		connection: conn,
		db:         db,
		config:     config,
		lastActive: time.Now(),
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
	}
}
