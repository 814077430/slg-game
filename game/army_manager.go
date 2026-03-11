package game

import (
	"context"
	"log"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"slg-game/database"
)

// ArmyManager 军队管理器
type ArmyManager struct {
	db *database.Database
}

// NewArmyManager 创建军队管理器
func NewArmyManager(db *database.Database) *ArmyManager {
	return &ArmyManager{db: db}
}

// TroopInfo 军队信息
type TroopInfo struct {
	Type  string `bson:"type" json:"type"`
	Count int32  `bson:"count" json:"count"`
	Power int32  `bson:"power" json:"power"` // 战斗力
}

// BattleResult 战斗结果
type BattleResult struct {
	AttackerID      uint64            `bson:"attacker_id" json:"attacker_id"`
	DefenderID      uint64            `bson:"defender_id" json:"defender_id"`
	AttackerWon     bool              `bson:"attacker_won" json:"attacker_won"`
	AttackerLosses  map[string]int32  `bson:"attacker_losses" json:"attacker_losses"`
	DefenderLosses  map[string]int32  `bson:"defender_losses" json:"defender_losses"`
	LootedResources map[string]int64  `bson:"looted_resources" json:"looted_resources"`
	BattleTime      time.Time         `bson:"battle_time" json:"battle_time"`
}

// CalculatePower 计算军队总战力
func (am *ArmyManager) CalculatePower(troops []database.Troop) int32 {
	totalPower := int32(0)
	for _, troop := range troops {
		// 基础战力 + 等级加成
		basePower := am.getTroopBasePower(troop.Type)
		totalPower += basePower * int32(troop.Count) * (1 + int32(troop.Count)/100)
	}
	return totalPower
}

// getTroopBasePower 获取兵种基础战力
func (am *ArmyManager) getTroopBasePower(troopType string) int32 {
	powerMap := map[string]int32{
		"infantry": 10,
		"archer":   8,
		"cavalry":  15,
		"siege":    20,
	}
	if power, ok := powerMap[troopType]; ok {
		return power
	}
	return 10 // 默认战力
}

// Attack 发起攻击
func (am *ArmyManager) Attack(attackerID, defenderID uint64, attackerTroops []database.Troop) (*BattleResult, error) {
	// 获取防御方军队
	defenderTroops, err := am.GetPlayerTroops(defenderID)
	if err != nil {
		return nil, err
	}

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
		AttackerWon:     attackerWon,
		AttackerLosses:  attackerLosses,
		DefenderLosses:  defenderLosses,
		LootedResources: lootedResources,
		BattleTime:      time.Now(),
	}

	// 保存战斗记录
	err = am.saveBattleLog(result)
	if err != nil {
		log.Printf("Failed to save battle log: %v", err)
	}

	// 更新双方军队
	am.updateTroops(attackerID, attackerLosses)
	am.updateTroops(defenderID, defenderLosses)

	// 如果攻击方胜利，转移掠夺资源
	if attackerWon && len(lootedResources) > 0 {
		am.transferLoot(attackerID, defenderID, lootedResources)
	}

	log.Printf("Battle completed: attacker=%d, defender=%d, winner=%s",
		attackerID, defenderID, map[bool]string{true: "attacker", false: "defender"}[attackerWon])

	return result, nil
}

// calculateLosses 计算损失
func (am *ArmyManager) calculateLosses(troops []database.Troop, isWinner bool) map[string]int32 {
	losses := make(map[string]int32)
	lossRate := int32(10) // 胜利方 10% 损失
	if !isWinner {
		lossRate = 50 // 失败方 50% 损失
	}

	for _, troop := range troops {
		loss := int32(troop.Count * int64(lossRate) / 100)
		if loss > 0 {
			losses[troop.Type] = loss
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
	var player database.Player
	err := collection.FindOne(context.Background(), bson.M{"player_id": defenderID}).Decode(&player)
	if err != nil {
		return map[string]int64{}
	}

	// 掠夺比例（30%）
	lootRatio := 0.3
	return map[string]int64{
		"gold": int64(float64(player.Gold) * lootRatio),
		"wood": int64(float64(player.Wood) * lootRatio),
		"food": int64(float64(player.Food) * lootRatio),
	}
}

// saveBattleLog 保存战斗记录
func (am *ArmyManager) saveBattleLog(result *BattleResult) error {
	collection := am.db.GetCollection("battle_logs")
	_, err := collection.InsertOne(context.Background(), result)
	return err
}

// updateTroops 更新军队数量
func (am *ArmyManager) updateTroops(playerID uint64, losses map[string]int32) error {
	if len(losses) == 0 {
		return nil
	}

	collection := am.db.GetCollection("players")

	// 更新军队数量
	for troopType, loss := range losses {
		_, err := collection.UpdateOne(
			context.Background(),
			bson.M{
				"player_id": playerID,
				"troops.type": troopType,
			},
			bson.M{
				"$inc": bson.M{"troops.$.count": -loss},
			},
		)
		if err != nil {
			log.Printf("Failed to update troops: %v", err)
		}
	}
	return nil
}

// transferLoot 转移掠夺资源
func (am *ArmyManager) transferLoot(attackerID, defenderID uint64, loot map[string]int64) error {
	collection := am.db.GetCollection("players")

	// 扣除防御方资源
	for resource, amount := range loot {
		switch resource {
		case "gold":
			collection.UpdateOne(context.Background(), bson.M{"player_id": defenderID}, bson.M{"$inc": bson.M{"gold": -amount}})
			collection.UpdateOne(context.Background(), bson.M{"player_id": attackerID}, bson.M{"$inc": bson.M{"gold": amount}})
		case "wood":
			collection.UpdateOne(context.Background(), bson.M{"player_id": defenderID}, bson.M{"$inc": bson.M{"wood": -amount}})
			collection.UpdateOne(context.Background(), bson.M{"player_id": attackerID}, bson.M{"$inc": bson.M{"wood": amount}})
		case "food":
			collection.UpdateOne(context.Background(), bson.M{"player_id": defenderID}, bson.M{"$inc": bson.M{"food": -amount}})
			collection.UpdateOne(context.Background(), bson.M{"player_id": attackerID}, bson.M{"$inc": bson.M{"food": amount}})
		}
	}
	return nil
}

// GetPlayerTroops 获取玩家军队
func (am *ArmyManager) GetPlayerTroops(playerID uint64) ([]database.Troop, error) {
	collection := am.db.GetCollection("players")
	var player database.Player
	err := collection.FindOne(context.Background(), bson.M{"player_id": playerID}).Decode(&player)
	if err != nil {
		return nil, err
	}
	return player.Troops, nil
}
