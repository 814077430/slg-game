package game

import (
	"fmt"
	"time"

	"slg-game/database"
)

// UnitType 兵种类型
type UnitType string

const (
	UnitTypeInfantry   UnitType = "infantry"   // 步兵
	UnitTypeArcher     UnitType = "archer"     // 弓箭手
	UnitTypeCavalry    UnitType = "cavalry"    // 骑兵
	UnitTypeSiege      UnitType = "siege"      // 攻城器械
	UnitTypeHero       UnitType = "hero"       // 英雄
)

// UnitStats 兵种属性
type UnitStats struct {
	Attack    int32 `json:"attack"`    // 攻击力
	Defense   int32 `json:"defense"`   // 防御力
	Health    int32 `json:"health"`    // 生命值
	Speed     int32 `json:"speed"`     // 速度
	Carry     int32 `json:"carry"`     // 负重
	TrainingTime int32 `json:"training_time"` // 训练时间（秒）
	ResourceCost map[string]int32 `json:"resource_cost"` // 资源消耗
}

// ArmyUnit 军队单位
type ArmyUnit struct {
	Type      UnitType `json:"type"`
	Level     int32    `json:"level"`
	Count     int32    `json:"count"`
	Position  int32    `json:"position"` // 在军队中的位置
}

// Army 军队
type Army struct {
	ID        string      `json:"id"`
	PlayerID  uint64      `json:"player_id"`
	Units     []ArmyUnit  `json:"units"`
	X         int32       `json:"x"`
	Y         int32       `json:"y"`
	TargetX   int32       `json:"target_x,omitempty"`
	TargetY   int32       `json:"target_y,omitempty"`
	MovementStartTime *time.Time `json:"movement_start_time,omitempty"`
	ArrivalTime       *time.Time `json:"arrival_time,omitempty"`
	Status    string      `json:"status"` // idle, moving, attacking, returning
}

// BattleResult 战斗结果
type BattleResult struct {
	AttackerWins bool           `json:"attacker_wins"`
	AttackerLosses map[UnitType]int32 `json:"attacker_losses"`
	DefenderLosses map[UnitType]int32 `json:"defender_losses"`
	Loot           map[string]int32   `json:"loot"`
}

// ArmyManager 军队管理器
type ArmyManager struct {
	db *database.Database
}

func NewArmyManager(db *database.Database) *ArmyManager {
	return &ArmyManager{db: db}
}

// GetUnitStats 获取兵种属性
func (am *ArmyManager) GetUnitStats(unitType UnitType, level int32) (*UnitStats, error) {
	// 这里应该从配置或数据库中获取兵种数据
	// 为了简化，我们使用硬编码的基础数据
	baseStats := map[UnitType]UnitStats{
		UnitTypeInfantry: {
			Attack: 10, Defense: 15, Health: 100, Speed: 5, Carry: 10,
			TrainingTime: 60,
			ResourceCost: map[string]int32{"food": 50, "wood": 20},
		},
		UnitTypeArcher: {
			Attack: 15, Defense: 8, Health: 80, Speed: 6, Carry: 5,
			TrainingTime: 90,
			ResourceCost: map[string]int32{"food": 30, "wood": 60},
		},
		UnitTypeCavalry: {
			Attack: 20, Defense: 12, Health: 120, Speed: 10, Carry: 15,
			TrainingTime: 120,
			ResourceCost: map[string]int32{"food": 80, "gold": 40},
		},
		UnitTypeSiege: {
			Attack: 30, Defense: 5, Health: 150, Speed: 3, Carry: 50,
			TrainingTime: 180,
			ResourceCost: map[string]int32{"wood": 100, "stone": 50},
		},
		UnitTypeHero: {
			Attack: 50, Defense: 50, Health: 500, Speed: 8, Carry: 100,
			TrainingTime: 3600,
			ResourceCost: map[string]int32{"gold": 1000},
		},
	}

	if stats, exists := baseStats[unitType]; exists {
		// 根据等级调整属性
		multiplier := float32(level)
		return &UnitStats{
			Attack:    int32(float32(stats.Attack) * multiplier),
			Defense:   int32(float32(stats.Defense) * multiplier),
			Health:    int32(float32(stats.Health) * multiplier),
			Speed:     stats.Speed, // 速度通常不随等级提升
			Carry:     int32(float32(stats.Carry) * multiplier),
			TrainingTime: stats.TrainingTime,
			ResourceCost: stats.ResourceCost,
		}, nil
	}

	return nil, fmt.Errorf("unknown unit type: %s", unitType)
}

// CreateArmy 创建军队
func (am *ArmyManager) CreateArmy(playerID uint64, units []ArmyUnit, x, y int32) (*Army, error) {
	army := &Army{
		ID:       fmt.Sprintf("army_%d_%d", playerID, time.Now().Unix()),
		PlayerID: playerID,
		Units:    units,
		X:        x,
		Y:        y,
		Status:   "idle",
	}

	// 保存到数据库
	collection := am.db.GetCollection("armies")
	_, err := collection.InsertOne(nil, army)
	if err != nil {
		return nil, err
	}

	return army, nil
}

// MoveArmy 移动军队
func (am *ArmyManager) MoveArmy(armyID string, targetX, targetY int32) error {
	// 计算移动时间（基于距离和军队速度）
	// 这里简化处理，假设固定速度
	collection := am.db.GetCollection("armies")
	_, err := collection.UpdateOne(nil, 
		map[string]interface{}{"id": armyID},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"target_x": targetX,
				"target_y": targetY,
				"status": "moving",
				"movement_start_time": time.Now(),
				"arrival_time": time.Now().Add(time.Minute * 5), // 简化：5分钟到达
			},
		})
	return err
}

// CalculateBattle 计算战斗结果
func (am *ArmyManager) CalculateBattle(attacker *Army, defender *Army) *BattleResult {
	result := &BattleResult{
		AttackerLosses: make(map[UnitType]int32),
		DefenderLosses: make(map[UnitType]int32),
		Loot:          make(map[string]int32),
	}

	// 简化的战斗计算
	attackerPower := am.calculateArmyPower(attacker)
	defenderPower := am.calculateArmyPower(defender)

	if attackerPower > defenderPower {
		result.AttackerWins = true
		// 计算损失（简化）
		lossRatio := defenderPower / (attackerPower + defenderPower)
		for _, unit := range attacker.Units {
			result.AttackerLosses[unit.Type] = int32(float32(unit.Count) * lossRatio)
		}
		for _, unit := range defender.Units {
			result.DefenderLosses[unit.Type] = unit.Count // 全军覆没
		}
	} else {
		result.AttackerWins = false
		lossRatio := attackerPower / (attackerPower + defenderPower)
		for _, unit := range attacker.Units {
			result.AttackerLosses[unit.Type] = unit.Count // 全军覆没
		}
		for _, unit := range defender.Units {
			result.DefenderLosses[unit.Type] = int32(float32(unit.Count) * lossRatio)
		}
	}

	// 如果攻击方获胜，计算掠夺资源
	if result.AttackerWins {
		totalCarry := am.calculateArmyCarry(attacker)
		// 这里应该查询目标玩家的资源，简化处理
		result.Loot["food"] = min(totalCarry/3, 1000)
		result.Loot["wood"] = min(totalCarry/3, 1000)
		result.Loot["stone"] = min(totalCarry/3, 1000)
	}

	return result
}

func (am *ArmyManager) calculateArmyPower(army *Army) float32 {
	var totalPower float32
	for _, unit := range army.Units {
		stats, _ := am.GetUnitStats(unit.Type, unit.Level)
		if stats != nil {
			unitPower := float32(stats.Attack+stats.Defense) * float32(unit.Count)
			totalPower += unitPower
		}
	}
	return totalPower
}

func (am *ArmyManager) calculateArmyCarry(army *Army) int32 {
	var totalCarry int32
	for _, unit := range army.Units {
		stats, _ := am.GetUnitStats(unit.Type, unit.Level)
		if stats != nil {
			totalCarry += stats.Carry * unit.Count
		}
	}
	return totalCarry
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// GetPlayerArmies 获取玩家的所有军队
func (am *ArmyManager) GetPlayerArmies(playerID uint64) ([]*Army, error) {
	collection := am.db.GetCollection("armies")
	cursor, err := collection.Find(nil, map[string]interface{}{"player_id": playerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(nil)

	var armies []*Army
	if err = cursor.All(nil, &armies); err != nil {
		return nil, err
	}

	return armies, nil
}

// UpdateArmy 更新军队状态
func (am *ArmyManager) UpdateArmy(army *Army) error {
	collection := am.db.GetCollection("armies")
	_, err := collection.ReplaceOne(nil, map[string]interface{}{"id": army.ID}, army)
	return err
}