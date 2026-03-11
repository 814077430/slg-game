package core

import (
	"sync"
)

// PlayerInfo 玩家信息
type PlayerInfo struct {
	ID       uint64
	Username string
	X        int32
	Y        int32
	Online   bool
}

// GetPlayerInfo 获取玩家信息（用于 chat 包）
func (pm *PlayerManager) GetPlayerInfo(playerID uint64) *PlayerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.players[playerID]
}

// PlayerManager 玩家管理器
type PlayerManager struct {
	players    map[uint64]*PlayerInfo
	sessions   map[uint64]*PlayerSession
	mutex      sync.RWMutex
	viewRange  int32 // 视野范围（格）
}

// NewPlayerManager 创建玩家管理器
func NewPlayerManager(viewRange int32) *PlayerManager {
	return &PlayerManager{
		players:   make(map[uint64]*PlayerInfo),
		sessions:  make(map[uint64]*PlayerSession),
		viewRange: viewRange,
	}
}

// AddPlayer 添加玩家
func (pm *PlayerManager) AddPlayer(playerID uint64, username string, session *PlayerSession) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.players[playerID] = &PlayerInfo{
		ID:       playerID,
		Username: username,
		X:        0,
		Y:        0,
		Online:   true,
	}
	pm.sessions[playerID] = session
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
func (pm *PlayerManager) GetPlayer(playerID uint64) *PlayerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.players[playerID]
}

// GetPlayersInVision 获取视野内的玩家
func (pm *PlayerManager) GetPlayersInVision(playerID uint64) []*PlayerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	target, exists := pm.players[playerID]
	if !exists {
		return nil
	}

	var visible []*PlayerInfo
	for id, player := range pm.players {
		if id == playerID {
			continue // 排除自己
		}
		if !player.Online {
			continue // 排除离线玩家
		}

		// 计算距离（曼哈顿距离）
		dx := abs32(player.X - target.X)
		dy := abs32(player.Y - target.Y)

		// 在视野范围内
		if dx <= pm.viewRange && dy <= pm.viewRange {
			visible = append(visible, player)
		}
	}

	return visible
}

// GetSession 获取玩家会话（返回 interface{} 避免循环导入）
func (pm *PlayerManager) GetSession(playerID uint64) interface{} {
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

// abs32 int32 绝对值
func abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
