package game

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"slg-game/config"
	"slg-game/database"
	"slg-game/network"
)

type PlayerSession struct {
	connection  *network.Connection
	db          *database.Database
	config      *config.Config
	playerID    uint64
	username    string
	isLoggedIn  bool
	loginTime   time.Time
	lastActive  time.Time
	mutex       sync.RWMutex
}

func NewPlayerSession(conn *network.Connection, db *database.Database, config *config.Config) *PlayerSession {
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

// Cleanup 清理会话资源 - 保存玩家数据、从在线列表移除等
func (ps *PlayerSession) Cleanup() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.isLoggedIn && ps.playerID > 0 {
		log.Printf("Cleaning up session for player: %s (ID: %d), duration: %v",
			ps.username, ps.playerID, time.Since(ps.loginTime))

		// 保存玩家数据到数据库
		ps.savePlayerData()
	}
}

// savePlayerData 保存玩家数据到数据库
func (ps *PlayerSession) savePlayerData() {
	if ps.playerID == 0 {
		return
	}

	collection := ps.db.GetCollection("players")

	// 更新最后登录时间和会话信息
	update := bson.M{
		"$set": bson.M{
			"last_login": time.Now(),
		},
	}

	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"player_id": ps.playerID},
		update,
	)

	if err != nil {
		log.Printf("Failed to save player data for %s: %v", ps.username, err)
	} else {
		log.Printf("Player data saved: %s (ID: %d)", ps.username, ps.playerID)
	}
}
