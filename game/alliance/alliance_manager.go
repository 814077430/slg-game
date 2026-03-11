package alliance

import (
	"fmt"
	"time"

	"slg-game/database"
)

// AllianceManager 联盟管理器（简化版）
type AllianceManager struct {
	db *database.MemoryDB
}

// NewAllianceManager 创建联盟管理器
func NewAllianceManager(db *database.MemoryDB) *AllianceManager {
	return &AllianceManager{db: db}
}

// CreateAlliance 创建联盟（简化版）
func (am *AllianceManager) CreateAlliance(leaderID uint64, name, description string) error {
	collection := am.db.GetCollection("alliances")
	
	// 检查是否已在联盟中
	playerCollection := am.db.GetCollection("players")
	player, err := playerCollection.FindOne(map[string]interface{}{"player_id": leaderID})
	if err != nil {
		return err
	}
	
	if player["alliance_id"] != nil && player["alliance_id"].(uint64) > 0 {
		return fmt.Errorf("already in alliance")
	}
	
	// 创建联盟
	alliance := map[string]interface{}{
		"name":         name,
		"description":  description,
		"creator_id":   leaderID,
		"created_at":   time.Now(),
		"member_count": 1,
		"max_members":  50,
		"level":        1,
		"members": []interface{}{
			map[string]interface{}{
				"player_id": leaderID,
				"role":      "leader",
				"joined_at": time.Now(),
			},
		},
	}
	
	return collection.InsertOne(alliance)
}

// JoinAlliance 加入联盟
func (am *AllianceManager) JoinAlliance(playerID uint64, allianceID uint64) error {
	return fmt.Errorf("not implemented")
}

// LeaveAlliance 离开联盟
func (am *AllianceManager) LeaveAlliance(playerID uint64) error {
	return fmt.Errorf("not implemented")
}

// GetAlliance 获取联盟信息
func (am *AllianceManager) GetAlliance(allianceID uint64) (map[string]interface{}, error) {
	collection := am.db.GetCollection("alliances")
	return collection.FindOne(map[string]interface{}{"alliance_id": allianceID})
}
