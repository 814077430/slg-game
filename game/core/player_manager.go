package core

import (
	"sync"
	"time"

	"slg-game/chat"
	"slg-game/database"
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

// PlayerManager 玩家管理器
type PlayerManager struct {
	players        map[uint64]*session.PlayerInfo     // 在线玩家（一直在内存）
	sessions       map[uint64]session.Session         // 在线会话
	offlinePlayers map[uint64]*OfflinePlayer          // 离线玩家（保留 10 分钟）
	usernameIndex  map[string]uint64                  // 用户名索引
	playerCache    map[uint64]*PlayerCache            // 玩家数据缓存（含密码哈希，保留 10 分钟）
	mongoWriter    *database.MongoAsyncWriter         // MongoDB 异步写入器
	mutex          sync.RWMutex
	cleanupTicker  *time.Ticker
	stopChan       chan struct{}
}

// PlayerCache 玩家缓存数据
type PlayerCache struct {
	Username     string
	PasswordHash string
}

// NewPlayerManager 创建玩家管理器
func NewPlayerManager(mongoWriter *database.MongoAsyncWriter) *PlayerManager {
	pm := &PlayerManager{
		players:        make(map[uint64]*session.PlayerInfo),
		sessions:       make(map[uint64]session.Session),
		offlinePlayers: make(map[uint64]*OfflinePlayer),
		usernameIndex:  make(map[string]uint64),
		playerCache:    make(map[uint64]*PlayerCache),
		mongoWriter:    mongoWriter,
		stopChan:       make(chan struct{}),
	}

	// 启动定期清理协程（每 2 分钟检查一次，清理离线 10 分钟的玩家）
	pm.cleanupTicker = time.NewTicker(2 * time.Minute)
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

// cleanupExpiredOfflinePlayers 清理过期（>10 分钟）的离线玩家数据
func (pm *PlayerManager) cleanupExpiredOfflinePlayers() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	now := time.Now()
	for playerID, offline := range pm.offlinePlayers {
		if now.Sub(offline.LastLogout) > 10*time.Minute {
			// 删除离线玩家记录
			delete(pm.offlinePlayers, playerID)
			// 删除玩家缓存
			delete(pm.playerCache, playerID)
			// 删除用户名索引
			delete(pm.usernameIndex, offline.Info.Username)
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
	
	// 添加用户名索引（用于登录查找）
	pm.usernameIndex[username] = playerID
}

// AddPlayerCache 添加玩家缓存（注册时使用）并异步写入 MongoDB
func (pm *PlayerManager) AddPlayerCache(playerID uint64, username, passwordHash string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	pm.usernameIndex[username] = playerID
	pm.playerCache[playerID] = &PlayerCache{
		Username:     username,
		PasswordHash: passwordHash,
	}

	// 异步写入 MongoDB
	if pm.mongoWriter != nil {
		go func() {
			pm.mongoWriter.UpdatePlayer(playerID, map[string]interface{}{
				"player_id":     playerID,
				"username":      username,
				"password_hash": passwordHash,
			})
		}()
	}
}

// UpdatePlayerData 更新玩家数据并异步写入 MongoDB
func (pm *PlayerManager) UpdatePlayerData(playerID uint64, data map[string]interface{}) {
	pm.mutex.Lock()
	if player, exists := pm.players[playerID]; exists {
		// 更新内存数据
		if x, ok := data["x"].(int32); ok {
			player.X = x
		}
		if y, ok := data["y"].(int32); ok {
			player.Y = y
		}
	}
	pm.mutex.Unlock()

	// 异步写入 MongoDB
	if pm.mongoWriter != nil {
		go func() {
			pm.mongoWriter.UpdatePlayer(playerID, data)
		}()
	}
}

// RemovePlayer 移除玩家（离线数据保留 10 分钟）
func (pm *PlayerManager) RemovePlayer(playerID uint64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 将玩家移到离线列表（数据保留 10 分钟）
	if player, exists := pm.players[playerID]; exists {
		pm.offlinePlayers[playerID] = &OfflinePlayer{
			Info:       player,
			LastLogout: time.Now(),
		}
		// 记录离线时间（用于清理）
		pm.playerCache[playerID] = &PlayerCache{}
		// 从在线列表移除
		delete(pm.players, playerID)
	}

	// 会话移除
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
	delete(pm.playerCache, playerID)
}

// GetPlayerCache 获取玩家缓存（登录时验证密码）
func (pm *PlayerManager) GetPlayerCache(playerID uint64) (*PlayerCache, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	cache, exists := pm.playerCache[playerID]
	return cache, exists
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
	return len(pm.players)
}

// UpdatePlayerPosition 更新玩家位置并异步写入 MongoDB
func (pm *PlayerManager) UpdatePlayerPosition(playerID uint64, x, y int32) {
	pm.mutex.Lock()
	if player, exists := pm.players[playerID]; exists {
		player.X = x
		player.Y = y
	}
	pm.mutex.Unlock()

	// 异步写入 MongoDB
	if pm.mongoWriter != nil {
		go func() {
			pm.mongoWriter.UpdatePlayer(playerID, map[string]interface{}{
				"x": x,
				"y": y,
			})
		}()
	}
}

// GetPlayerIDByUsername 通过用户名获取玩家 ID
func (pm *PlayerManager) GetPlayerIDByUsername(username string) (uint64, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	playerID, exists := pm.usernameIndex[username]
	return playerID, exists
}

// GetAllPlayers 获取所有在线玩家
func (pm *PlayerManager) GetAllPlayers() []interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	players := make([]interface{}, 0, len(pm.players))
	for _, player := range pm.players {
		players = append(players, player)
	}
	return players
}

// Stop 停止玩家管理器
func (pm *PlayerManager) Stop() {
	close(pm.stopChan)
	pm.cleanupTicker.Stop()
}
