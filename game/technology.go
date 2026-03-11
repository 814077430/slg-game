package game

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"slg-game/database"
)

// TechnologyType 定义科技类型
type TechnologyType string

const (
	TechResourceProduction   TechnologyType = "resource_production"   // 资源生产
	TechBuildingSpeed        TechnologyType = "building_speed"        // 建筑速度
	TechArmyTraining         TechnologyType = "army_training"         // 军队训练
	TechArmyStrength         TechnologyType = "army_strength"         // 军队强度
	TechDefense              TechnologyType = "defense"               // 防御
	TechResearchSpeed        TechnologyType = "research_speed"        // 研究速度
)

// TechnologyLevel 科技等级信息
type TechnologyLevel struct {
	Level        int32  `bson:"level" json:"level"`
	Name         string `bson:"name" json:"name"`
	Description  string `bson:"description" json:"description"`
	ResourceCost map[string]int32 `bson:"resource_cost" json:"resource_cost"` // 升级所需资源
	TimeCost     int32  `bson:"time_cost" json:"time_cost"`           // 升级所需时间（秒）
	Effects      map[string]float64 `bson:"effects" json:"effects"`         // 科技效果
}

// Technology 科技信息
type Technology struct {
	Type         TechnologyType    `bson:"type" json:"type"`
	CurrentLevel int32             `bson:"current_level" json:"current_level"`
	Levels       []TechnologyLevel `bson:"levels" json:"levels"`
}

// ResearchQueueItem 研究队列项
type ResearchQueueItem struct {
	PlayerID       uint64         `bson:"player_id" json:"player_id"`
	TechnologyType TechnologyType `bson:"technology_type" json:"technology_type"`
	TargetLevel    int32          `bson:"target_level" json:"target_level"`
	StartTime      time.Time      `bson:"start_time" json:"start_time"`
	EndTime        time.Time      `bson:"end_time" json:"end_time"`
}

// TechnologyManager 科技管理器
type TechnologyManager struct {
	db *database.Database
}

// NewTechnologyManager 创建新的科技管理器
func NewTechnologyManager(db *database.Database) *TechnologyManager {
	return &TechnologyManager{
		db: db,
	}
}

// GetTechnologyConfig 获取科技配置
func (tm *TechnologyManager) GetTechnologyConfig(techType TechnologyType) (*Technology, error) {
	collection := tm.db.GetCollection("technology_config")
	
	var tech Technology
	err := collection.FindOne(context.Background(), bson.M{"type": techType}).Decode(&tech)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// 如果没有找到配置，创建默认配置
			tech = tm.createDefaultTechnology(techType)
			_, err = collection.InsertOne(context.Background(), tech)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	
	return &tech, nil
}

// createDefaultTechnology 创建默认科技配置
func (tm *TechnologyManager) createDefaultTechnology(techType TechnologyType) Technology {
	tech := Technology{
		Type:         techType,
		CurrentLevel: 0,
		Levels:       make([]TechnologyLevel, 0),
	}
	
	// 根据科技类型设置默认等级
	switch techType {
	case TechResourceProduction:
		tech.Levels = []TechnologyLevel{
			{Level: 1, Name: "基础采集", Description: "提高资源采集效率", ResourceCost: map[string]int32{"gold": 100, "wood": 50}, TimeCost: 60, Effects: map[string]float64{"resource_multiplier": 1.1}},
			{Level: 2, Name: "高效采集", Description: "大幅提高资源采集效率", ResourceCost: map[string]int32{"gold": 200, "wood": 100}, TimeCost: 120, Effects: map[string]float64{"resource_multiplier": 1.2}},
			{Level: 3, Name: "专业采集", Description: "极大提高资源采集效率", ResourceCost: map[string]int32{"gold": 400, "wood": 200}, TimeCost: 240, Effects: map[string]float64{"resource_multiplier": 1.3}},
		}
	case TechBuildingSpeed:
		tech.Levels = []TechnologyLevel{
			{Level: 1, Name: "快速建造", Description: "减少建筑建造时间", ResourceCost: map[string]int32{"gold": 150, "stone": 50}, TimeCost: 90, Effects: map[string]float64{"build_time_multiplier": 0.9}},
			{Level: 2, Name: "高效建造", Description: "大幅减少建筑建造时间", ResourceCost: map[string]int32{"gold": 300, "stone": 100}, TimeCost: 180, Effects: map[string]float64{"build_time_multiplier": 0.8}},
			{Level: 3, Name: "专家建造", Description: "极大减少建筑建造时间", ResourceCost: map[string]int32{"gold": 600, "stone": 200}, TimeCost: 360, Effects: map[string]float64{"build_time_multiplier": 0.7}},
		}
	case TechArmyTraining:
		tech.Levels = []TechnologyLevel{
			{Level: 1, Name: "基础训练", Description: "减少军队训练时间", ResourceCost: map[string]int32{"gold": 100, "food": 100}, TimeCost: 60, Effects: map[string]float64{"train_time_multiplier": 0.9}},
			{Level: 2, Name: "高效训练", Description: "大幅减少军队训练时间", ResourceCost: map[string]int32{"gold": 200, "food": 200}, TimeCost: 120, Effects: map[string]float64{"train_time_multiplier": 0.8}},
			{Level: 3, Name: "专业训练", Description: "极大减少军队训练时间", ResourceCost: map[string]int32{"gold": 400, "food": 400}, TimeCost: 240, Effects: map[string]float64{"train_time_multiplier": 0.7}},
		}
	case TechArmyStrength:
		tech.Levels = []TechnologyLevel{
			{Level: 1, Name: "基础强化", Description: "提高军队攻击力", ResourceCost: map[string]int32{"gold": 200, "iron": 100}, TimeCost: 120, Effects: map[string]float64{"attack_multiplier": 1.1}},
			{Level: 2, Name: "高级强化", Description: "大幅提高军队攻击力", ResourceCost: map[string]int32{"gold": 400, "iron": 200}, TimeCost: 240, Effects: map[string]float64{"attack_multiplier": 1.2}},
			{Level: 3, Name: "精英强化", Description: "极大提高军队攻击力", ResourceCost: map[string]int32{"gold": 800, "iron": 400}, TimeCost: 480, Effects: map[string]float64{"attack_multiplier": 1.3}},
		}
	case TechDefense:
		tech.Levels = []TechnologyLevel{
			{Level: 1, Name: "基础防御", Description: "提高防御力", ResourceCost: map[string]int32{"gold": 200, "stone": 100}, TimeCost: 120, Effects: map[string]float64{"defense_multiplier": 1.1}},
			{Level: 2, Name: "高级防御", Description: "大幅提高防御力", ResourceCost: map[string]int32{"gold": 400, "stone": 200}, TimeCost: 240, Effects: map[string]float64{"defense_multiplier": 1.2}},
			{Level: 3, Name: "精英防御", Description: "极大提高防御力", ResourceCost: map[string]int32{"gold": 800, "stone": 400}, TimeCost: 480, Effects: map[string]float64{"defense_multiplier": 1.3}},
		}
	case TechResearchSpeed:
		tech.Levels = []TechnologyLevel{
			{Level: 1, Name: "快速研究", Description: "减少科技研究时间", ResourceCost: map[string]int32{"gold": 150, "wood": 150}, TimeCost: 90, Effects: map[string]float64{"research_time_multiplier": 0.9}},
			{Level: 2, Name: "高效研究", Description: "大幅减少科技研究时间", ResourceCost: map[string]int32{"gold": 300, "wood": 300}, TimeCost: 180, Effects: map[string]float64{"research_time_multiplier": 0.8}},
			{Level: 3, Name: "专家研究", Description: "极大减少科技研究时间", ResourceCost: map[string]int32{"gold": 600, "wood": 600}, TimeCost: 360, Effects: map[string]float64{"research_time_multiplier": 0.7}},
		}
	}
	
	return tech
}

// StartResearch 开始研究科技
func (tm *TechnologyManager) StartResearch(playerID uint64, techType TechnologyType, targetLevel int32) error {
	// 获取玩家当前科技信息
	playerTech, err := tm.getPlayerTechnology(playerID, techType)
	if err != nil {
		return err
	}
	
	// 检查目标等级是否有效
	if targetLevel <= playerTech.CurrentLevel || targetLevel > int32(len(playerTech.Levels)) {
		return ErrInvalidTechnologyLevel
	}
	
	// 获取目标等级的科技信息
	targetLevelInfo := playerTech.Levels[targetLevel-1]
	
	// 检查玩家是否有足够的资源
	playerCollection := tm.db.GetCollection("players")
	var player database.Player
	err = playerCollection.FindOne(context.Background(), bson.M{"player_id": playerID}).Decode(&player)
	if err != nil {
		return err
	}
	
	// 简化：跳过资源检查，直接添加到研究队列
	endTime := time.Now().Add(time.Duration(targetLevelInfo.TimeCost) * time.Second)
	
	// 保存到研究队列
	researchCollection := tm.db.GetCollection("research_queue")
	_, err = researchCollection.InsertOne(context.Background(), bson.M{
		"player_id":       playerID,
		"technology_type": techType,
		"target_level":    targetLevel,
		"start_time":      time.Now(),
		"end_time":        endTime,
	})
	
	return err
}

// CompleteResearch 完成科技研究
func (tm *TechnologyManager) CompleteResearch(playerID uint64, techType TechnologyType, level int32) error {
	// 更新玩家科技等级
	techCollection := tm.db.GetCollection("player_technology")
	_, err := techCollection.UpdateOne(context.Background(),
		bson.M{"player_id": playerID, "technology_type": techType},
		bson.M{"$set": bson.M{"current_level": level}},
		options.Update().SetUpsert(true))
	
	if err != nil {
		return err
	}
	
	// 从研究队列中移除
	researchCollection := tm.db.GetCollection("research_queue")
	_, err = researchCollection.DeleteOne(context.Background(),
		bson.M{"player_id": playerID, "technology_type": techType, "target_level": level})
	
	return err
}

// getPlayerTechnology 获取玩家科技信息
func (tm *TechnologyManager) getPlayerTechnology(playerID uint64, techType TechnologyType) (*Technology, error) {
	// 获取科技配置
	config, err := tm.GetTechnologyConfig(techType)
	if err != nil {
		return nil, err
	}
	
	// 获取玩家当前等级
	techCollection := tm.db.GetCollection("player_technology")
	var playerTech struct {
		CurrentLevel int32 `bson:"current_level"`
	}
	
	err = techCollection.FindOne(context.Background(),
		bson.M{"player_id": playerID, "technology_type": techType}).Decode(&playerTech)
	
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}
	
	if err == mongo.ErrNoDocuments {
		playerTech.CurrentLevel = 0
	}
	
	config.CurrentLevel = playerTech.CurrentLevel
	return config, nil
}

// GetPlayerTechnologyLevel 获取玩家科技等级
func (tm *TechnologyManager) GetPlayerTechnologyLevel(playerID uint64, techType TechnologyType) (int32, error) {
	techCollection := tm.db.GetCollection("player_technology")
	var playerTech struct {
		CurrentLevel int32 `bson:"current_level"`
	}
	
	err := techCollection.FindOne(context.Background(),
		bson.M{"player_id": playerID, "technology_type": techType}).Decode(&playerTech)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}
	
	return playerTech.CurrentLevel, nil
}

// GetTechnologyEffect 获取科技效果
func (tm *TechnologyManager) GetTechnologyEffect(playerID uint64, techType TechnologyType) (map[string]float64, error) {
	level, err := tm.GetPlayerTechnologyLevel(playerID, techType)
	if err != nil {
		return nil, err
	}
	
	if level == 0 {
		return make(map[string]float64), nil
	}
	
	config, err := tm.GetTechnologyConfig(techType)
	if err != nil {
		return nil, err
	}
	
	if int(level) > len(config.Levels) {
		level = int32(len(config.Levels))
	}
	
	return config.Levels[level-1].Effects, nil
}

// ProcessResearchQueue 处理研究队列（定时任务调用）
func (tm *TechnologyManager) ProcessResearchQueue() {
	researchCollection := tm.db.GetCollection("research_queue")
	
	// 查找已完成的研究
	cursor, err := researchCollection.Find(context.Background(),
		bson.M{"end_time": bson.M{"$lte": time.Now()}})
	if err != nil {
		log.Printf("Error finding completed research: %v", err)
		return
	}
	defer cursor.Close(context.Background())
	
	var completedResearch []ResearchQueueItem
	if err = cursor.All(context.Background(), &completedResearch); err != nil {
		log.Printf("Error decoding completed research: %v", err)
		return
	}
	
	// 完成每个研究
	for _, research := range completedResearch {
		err := tm.CompleteResearch(research.PlayerID, research.TechnologyType, research.TargetLevel)
		if err != nil {
			log.Printf("Error completing research for player %d: %v", research.PlayerID, err)
			continue
		}
		
		log.Printf("Completed research for player %d: %s level %d", 
			research.PlayerID, research.TechnologyType, research.TargetLevel)
	}
}

var (
	ErrInvalidTechnologyLevel = &TechnologyError{"invalid technology level"}
	ErrInsufficientResources  = &TechnologyError{"insufficient resources"}
)

type TechnologyError struct {
	Message string
}

func (e *TechnologyError) Error() string {
	return e.Message
}