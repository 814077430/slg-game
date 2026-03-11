package game

import (
	"math/rand"
	"time"

	"slg-game/database"
)

// ArmyManager 军队管理器（简化版）
type ArmyManager struct {
	db *database.MemoryDB
}

// NewArmyManager 创建军队管理器
func NewArmyManager(db *database.MemoryDB) *ArmyManager {
	return &ArmyManager{db: db}
}

// TroopInfo 军队信息
type TroopInfo struct {
	Type  string `json:"type"`
	Count int32  `json:"count"`
	Power int32  `json:"power"`
}

// BattleResult 战斗结果
type BattleResult struct {
	AttackerID      uint64           `json:"attacker_id"`
	DefenderID      uint64           `json:"defender_id"`
	AttackerWon     bool             `json:"attacker_won"`
	AttackerLosses  map[string]int32 `json:"attacker_losses"`
	DefenderLosses  map[string]int32 `json:"defender_losses"`
	LootedResources map[string]int64 `json:"looted_resources"`
	BattleTime      time.Time        `json:"battle_time"`
}

// CalculatePower 计算军队总战力
func (am *ArmyManager) CalculatePower(troops []interface{}) int32 {
	totalPower := int32(0)
	for _, troop := range troops {
		if t, ok := troop.(map[string]interface{}); ok {
			count := t["count"].(int32)
			basePower := int32(10)
			totalPower += basePower * count
		}
	}
	return totalPower
}

// Attack 发起攻击（简化版）
func (am *ArmyManager) Attack(attackerID, defenderID uint64, attackerTroops []interface{}) (*BattleResult, error) {
	// 获取防御方军队
	defenderTroops := []interface{}{}
	
	// 计算双方战力
	attackerPower := am.CalculatePower(attackerTroops)
	defenderPower := am.CalculatePower(defenderTroops)
	
	// 添加随机因素
	randomFactor := 0.8 + rand.Float32()*0.4
	attackerPower = int32(float32(attackerPower) * randomFactor)
	
	// 判定胜负
	attackerWon := attackerPower > defenderPower
	
	// 创建战斗结果
	result := &BattleResult{
		AttackerID:      attackerID,
		DefenderID:      defenderID,
		AttackerWon:     attackerWon,
		AttackerLosses:  map[string]int32{},
		DefenderLosses:  map[string]int32{},
		LootedResources: map[string]int64{},
		BattleTime:      time.Now(),
	}
	
	return result, nil
}

// GetPlayerTroops 获取玩家军队
func (am *ArmyManager) GetPlayerTroops(playerID uint64) ([]interface{}, error) {
	collection := am.db.GetCollection("players")
	player, err := collection.FindOne(map[string]interface{}{"player_id": playerID})
	if err != nil {
		return []interface{}{}, err
	}
	
	if troops, ok := player["troops"].([]interface{}); ok {
		return troops, nil
	}
	
	return []interface{}{}, nil
}
