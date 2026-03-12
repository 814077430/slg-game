package core

import (
	"sync"
	"time"

	"slg-game/chat"
	"slg-game/handler"
	"slg-game/session"
)

// 确保实现 chat.PlayerManager 和 handler.PlayerManager 接口
var _ chat.PlayerManager = (*PlayerManager)(nil)
var _ handler.PlayerManager = (*PlayerManager)(nil)

// OfflinePlayer 离线玩家数据
type OfflinePlayer struct {
	Info       *session.PlayerInfo
	LastLogout time.Time
}

// PlayerCache 玩家缓存数据（用于登录验证）
type PlayerCache struct {
	PlayerID     uint64
	Username     string
	PasswordHash string
}

// PlayerManager 玩家管理器
type PlayerManager struct {
	players        map[uint64]*session.PlayerInfo      // 在线玩家
	sessions       map[uint64]session.Session          // 在线会话
	offlinePlayers map[uint64]*OfflinePlayer           // 离线玩家（保留 1 小时）
	usernameIndex  map[string]uint64                   // 用户名→玩家 ID 索引
	playerCache    map[uint64]*PlayerCache             // 玩家数据缓存（含密码哈希）
	mutex          sync.RWMutex
	cleanupTicker  *time.Ticker
	stopChan       chan struct{}
}

// NewPlayerManager 创建玩家管理器
func NewPlayerManager() *PlayerManager {
	pm := &PlayerManager{
		players:        make(map[uint64]*session.PlayerInfo),
		sessions:       make(map[uint64]session.Session),
		offlinePlayers: make(map[uint64]*OfflinePlayer),
		usernameIndex:  make(map[string]uint64),   // 用户名索引
		playerCache:    make(map[uint64]*PlayerCache), // 玩家数据缓存
		stopChan:       make(chan struct{}),
	}

	// 启动定期清理协程（每 5 分钟清理过期离线数据）
	pm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go pm.cleanupLoop()

	return pm
}

// cleanupLoop 定期清理过期离线玩家数据
func (pm *PlayerManager) cleanupLoop() {
	defer pm.cleanupTicker.Stop()

	for {
		select {
		case <-pm.cleanupTicker.C:
			pm.cleanupExpiredOfflinePlayers()
		case <-pm.stopChan:
			return
		}
	}
}

// cleanupExpiredOfflinePlayers 清理过期（>1 小时）的离线玩家数据
func (pm *PlayerManager) cleanupExpiredOfflinePlayers() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	now := time.Now()
	for playerID, offline := range pm.offlinePlayers {
		if now.Sub(offline.LastLogout) > time.Hour {
			delete(pm.offlinePlayers, playerID)
		}
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

// RemovePlayer 移除玩家（离线数据保留 1 小时）
func (pm *PlayerManager) RemovePlayer(playerID uint64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 将玩家移到离线列表
	if player, exists := pm.players[playerID]; exists {
		pm.offlinePlayers[playerID] = &OfflinePlayer{
			Info:       player,
			LastLogout: time.Now(),
		}
		delete(pm.players, playerID)
	}

	delete(pm.sessions, playerID)
}

// GetOfflinePlayer 获取离线玩家数据
func (pm *PlayerManager) GetOfflinePlayer(playerID uint64) *session.PlayerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if offline, exists := pm.offlinePlayers[playerID]; exists {
		return offline.Info
	}
	return nil
}

// RemoveOfflinePlayer 移除离线玩家数据（登录后调用）
func (pm *PlayerManager) RemoveOfflinePlayer(playerID uint64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	delete(pm.offlinePlayers, playerID)
}

// AddPlayerCache 添加玩家缓存（注册时调用，包含密码哈希）
func (pm *PlayerManager) AddPlayerCache(playerID uint64, username, passwordHash string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.usernameIndex[username] = playerID
	pm.playerCache[playerID] = &PlayerCache{
		PlayerID:     playerID,
		Username:     username,
		PasswordHash: passwordHash,
	}
}

// GetPlayerCache 获取玩家缓存（登录时验证密码）
func (pm *PlayerManager) GetPlayerCache(playerID uint64) (*PlayerCache, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	cache, exists := pm.playerCache[playerID]
	return cache, exists
}

// GetPlayerIDByUsername 通过用户名获取玩家 ID（登录时先用）
func (pm *PlayerManager) GetPlayerIDByUsername(username string) (uint64, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	playerID, exists := pm.usernameIndex[username]
	return playerID, exists
}

// Stop 停止玩家管理器
func (pm *PlayerManager) Stop() {
	close(pm.stopChan)
	pm.cleanupTicker.Stop()
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
