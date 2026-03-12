package core

import (
	"sync"

	"slg-game/chat"
	"slg-game/handler"
	"slg-game/session"
)

// 确保实现 chat.PlayerManager 和 handler.PlayerManager 接口
var _ chat.PlayerManager = (*PlayerManager)(nil)
var _ handler.PlayerManager = (*PlayerManager)(nil)

// PlayerManager 玩家管理器
type PlayerManager struct {
	players   map[uint64]*session.PlayerInfo
	sessions  map[uint64]session.Session
	mutex     sync.RWMutex
}

// NewPlayerManager 创建玩家管理器
func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players:  make(map[uint64]*session.PlayerInfo),
		sessions: make(map[uint64]session.Session),
	}
}

// AddPlayer 添加玩家
func (pm *PlayerManager) AddPlayer(playerID uint64, username string, sess session.Session) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.players[playerID] = &session.PlayerInfo{
		ID:       playerID,
		Username: username,
		X:        0,
		Y:        0,
		Online:   true,
	}
	pm.sessions[playerID] = sess
}

// RemovePlayer 移除玩家
func (pm *PlayerManager) RemovePlayer(playerID uint64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if player, exists := pm.players[playerID]; exists {
		player.Online = false
	}
	delete(pm.sessions, playerID)
}

// UpdatePlayerPosition 更新玩家位置
func (pm *PlayerManager) UpdatePlayerPosition(playerID uint64, x, y int32) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if player, exists := pm.players[playerID]; exists {
		player.X = x
		player.Y = y
	}
}

// GetPlayer 获取玩家信息
func (pm *PlayerManager) GetPlayer(playerID uint64) *session.PlayerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.players[playerID]
}

// GetSession 获取玩家会话
func (pm *PlayerManager) GetSession(playerID uint64) session.Session {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.sessions[playerID]
}

// GetPlayerCount 获取在线玩家数量
func (pm *PlayerManager) GetPlayerCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	count := 0
	for _, player := range pm.players {
		if player.Online {
			count++
		}
	}
	return count
}

// GetAllPlayers 获取所有在线玩家
func (pm *PlayerManager) GetAllPlayers() []interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var players []interface{}
	for _, player := range pm.players {
		if player.Online {
			players = append(players, player)
		}
	}
	return players
}
