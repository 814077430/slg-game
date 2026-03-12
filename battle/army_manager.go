package battle

import (
	"math/rand"
	"time"

	"slg-game/database"
	"slg-game/messenger"
)

// ArmyManager 军队管理器
type ArmyManager struct {
	db database.DB
	battleMgr *BattleManager
}

// NewArmyManager 创建军队管理器
func NewArmyManager(db database.DB, messageBus *messenger.MessageBus) *ArmyManager {
	am := &ArmyManager{
		db: db,
	}
	// 创建并启动战斗管理器（独立线程）
	am.battleMgr = NewBattleManager(db, messageBus)
	am.battleMgr.StartLoop()
	return am
}

// GetBattleManager 获取战斗管理器
func (am *ArmyManager) GetBattleManager() *BattleManager {
	return am.battleMgr
}

// Stop 停止所有子模块
func (am *ArmyManager) Stop() {
	if am.battleMgr != nil {
		am.battleMgr.StopLoop()
	}
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
	BattleTime      time.Time        `json:"battle_time"`
	Result          string           `json:"result"`
	AttackerLosses  map[string]int32 `json:"attacker_losses"`
	DefenderLosses  map[string]int32 `json:"defender_losses"`
	LootedResources map[string]int64 `json:"looted_resources"`
}

// CalculatePower 计算军队总战力
func (am *ArmyManager) CalculatePower(troops []interface{}) int32 {
	totalPower := int32(0)
	for _, troop := range troops {
		if t, ok := troop.(map[string]interface{}); ok {
			count, _ := t["count"].(int32)
			basePower := int32(10)
			totalPower += basePower * count
		}
	}
	return totalPower
}

// Attack 发起攻击（添加到战斗队列）
func (am *ArmyManager) Attack(attackerID, defenderID uint64, attackerTroops []interface{}) (*BattleResult, error) {
	// 获取防御方军队
	defenderTroops := []interface{}{}
	
	// 计算双方战力
	attackerPower := am.CalculatePower(attackerTroops)
	defenderPower := am.CalculatePower(defenderTroops)
	
	// 添加随机因素（±20%）
	randomFactor := 0.8 + rand.Float32()*0.4
	attackerPower = int32(float32(attackerPower) * randomFactor)
	
	// 判定胜负
	attackerWon := attackerPower > defenderPower
	
	// 计算损失
	attackerLosses := am.calculateLosses(attackerTroops, attackerWon)
	defenderLosses := am.calculateLosses(defenderTroops, !attackerWon)
	
	// 计算掠夺资源
	lootedResources := am.calculateLoot(attackerWon, defenderID)
	
	// 创建战斗结果
	result := &BattleResult{
		AttackerID:      attackerID,
		DefenderID:      defenderID,
		BattleTime:      time.Now(),
		Result:          map[bool]string{true: "attacker_win", false: "defender_win"}[attackerWon],
		AttackerLosses:  attackerLosses,
		DefenderLosses:  defenderLosses,
		LootedResources: lootedResources,
	}
	
	// 保存战斗记录
	am.saveBattleLog(result)
	
	// 更新双方军队
	am.updateTroops(attackerID, attackerLosses)
	am.updateTroops(defenderID, defenderLosses)
	
	// 如果攻击方胜利，转移掠夺资源
	if attackerWon && len(lootedResources) > 0 {
		am.transferLoot(attackerID, defenderID, lootedResources)
	}
	
	return result, nil
}

// calculateLosses 计算损失
func (am *ArmyManager) calculateLosses(troops []interface{}, isWinner bool) map[string]int32 {
	losses := make(map[string]int32)
	lossRate := int32(10) // 胜利方 10% 损失
	if !isWinner {
		lossRate = 50 // 失败方 50% 损失
	}
	
	for _, troop := range troops {
		if t, ok := troop.(map[string]interface{}); ok {
			troopType, _ := t["type"].(string)
			count, _ := t["count"].(int32)
			loss := count * lossRate / 100
			if loss > 0 {
				losses[troopType] = loss
			}
		}
	}
	return losses
}

// calculateLoot 计算掠夺资源
func (am *ArmyManager) calculateLoot(attackerWon bool, defenderID uint64) map[string]int64 {
	if !attackerWon {
		return map[string]int64{}
	}
	
	// 获取防御方资源
	collection := am.db.GetCollection("players")
	player, err := collection.FindOne(map[string]interface{}{"player_id": defenderID})
	if err != nil {
		return map[string]int64{}
	}
	
	// 掠夺比例（30%）
	lootRatio := 0.3
	return map[string]int64{
		"gold": int64(float64(player["gold"].(int64)) * lootRatio),
		"wood": int64(float64(player["wood"].(int64)) * lootRatio),
		"food": int64(float64(player["food"].(int64)) * lootRatio),
	}
}

// saveBattleLog 保存战斗记录
func (am *ArmyManager) saveBattleLog(result *BattleResult) {
	collection := am.db.GetCollection("battle_logs")
	collection.InsertOne(map[string]interface{}{
		"attacker_id":      result.AttackerID,
		"defender_id":      result.DefenderID,
		"battle_time":      result.BattleTime,
		"result":           result.Result,
		"attacker_losses":  result.AttackerLosses,
		"defender_losses":  result.DefenderLosses,
		"looted_resources": result.LootedResources,
	})
}

// updateTroops 更新军队数量
func (am *ArmyManager) updateTroops(playerID uint64, losses map[string]int32) {
	// TODO: 更新玩家军队数量
}

// transferLoot 转移掠夺资源
func (am *ArmyManager) transferLoot(attackerID, defenderID uint64, loot map[string]int64) {
	// 扣除防御方资源，增加攻击方资源
	// TODO: 实现资源转移逻辑
	_ = attackerID
	_ = defenderID
	_ = loot
}

// GetPlayerTroops 获取玩家军队
func (am *ArmyManager) GetPlayerTroops(playerID uint64) ([]interface{}, error) {
	collection := am.db.GetCollection("players")
	player, err := collection.FindOne(map[string]interface{}{"player_id": playerID})
	if err != nil {
		return []interface{}{}, nil
	}
	
	if troops, ok := player["troops"].([]interface{}); ok {
		return troops, nil
	}
	
	return []interface{}{}, nil
}
