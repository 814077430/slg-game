package city

import (
	"fmt"
	"time"

	"slg-game/database"
)

// BuildingType 定义建筑类型
type BuildingType string

const (
	BuildingTypeTownHall    BuildingType = "town_hall"
	BuildingTypeBarracks    BuildingType = "barracks"
	BuildingTypeFarm        BuildingType = "farm"
	BuildingTypeLumberMill  BuildingType = "lumber_mill"
	BuildingTypeMine        BuildingType = "mine"
)

// BuildingLevelConfig 建筑等级配置
type BuildingLevelConfig struct {
	Level        int32             `json:"level"`
	BuildTime    int64             `json:"build_time"`
	ResourceCost map[string]int32  `json:"resource_cost"`
	Production   map[string]int32  `json:"production"`
	Capacity     map[string]int32  `json:"capacity"`
	Stats        map[string]int32  `json:"stats"`
}

// Building 建筑实例
type Building struct {
	ID            string            `json:"id"`
	PlayerID      uint64            `json:"player_id"`
	Type          BuildingType      `json:"type"`
	Level         int32             `json:"level"`
	X             int32             `json:"x"`
	Y             int32             `json:"y"`
	Construction  *Construction     `json:"construction,omitempty"`
	LastCollected time.Time         `json:"last_collected"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Construction 正在建造的建筑
type Construction struct {
	TargetLevel int32     `json:"target_level"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// BuildingManager 建筑管理器
type BuildingManager struct {
	db database.DB
}

func NewBuildingManager(db database.DB) *BuildingManager {
	return &BuildingManager{db: db}
}

// GetBuildingConfig 获取建筑配置
func (bm *BuildingManager) GetBuildingConfig(buildingType BuildingType, level int32) (*BuildingLevelConfig, error) {
	config := bm.getHardcodedConfig(buildingType, level)
	if config == nil {
		return nil, fmt.Errorf("building config not found for type %s level %d", buildingType, level)
	}
	return config, nil
}

// getHardcodedConfig 获取硬编码的建筑配置
func (bm *BuildingManager) getHardcodedConfig(buildingType BuildingType, level int32) *BuildingLevelConfig {
	baseConfigs := map[BuildingType]map[int32]*BuildingLevelConfig{
		BuildingTypeTownHall: {
			1: {Level: 1, BuildTime: 60, ResourceCost: map[string]int32{"gold": 100, "wood": 50}},
			2: {Level: 2, BuildTime: 120, ResourceCost: map[string]int32{"gold": 200, "wood": 100}},
		},
		BuildingTypeFarm: {
			1: {Level: 1, BuildTime: 30, ResourceCost: map[string]int32{"wood": 50}, Production: map[string]int32{"food": 10}},
			2: {Level: 2, BuildTime: 60, ResourceCost: map[string]int32{"wood": 100}, Production: map[string]int32{"food": 20}},
		},
		BuildingTypeLumberMill: {
			1: {Level: 1, BuildTime: 30, ResourceCost: map[string]int32{"wood": 50}, Production: map[string]int32{"wood": 10}},
		},
		BuildingTypeMine: {
			1: {Level: 1, BuildTime: 30, ResourceCost: map[string]int32{"wood": 50}, Production: map[string]int32{"gold": 10}},
		},
	}

	if configs, exists := baseConfigs[buildingType]; exists {
		if config, exists := configs[level]; exists {
			return config
		}
	}
	return nil
}

// CreateBuilding 创建新建筑
func (bm *BuildingManager) CreateBuilding(playerID uint64, buildingType BuildingType, x, y int32) (*Building, error) {
	building := &Building{
		PlayerID:  playerID,
		Type:      buildingType,
		Level:     0,
		X:         x,
		Y:         y,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := bm.startConstruction(building, 1)
	if err != nil {
		return nil, err
	}

	return building, nil
}

// startConstruction 开始建筑建造
func (bm *BuildingManager) startConstruction(building *Building, targetLevel int32) error {
	config, err := bm.GetBuildingConfig(building.Type, targetLevel)
	if err != nil {
		return err
	}

	building.Construction = &Construction{
		TargetLevel: targetLevel,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Duration(config.BuildTime) * time.Second),
	}

	return nil
}

// GetBuildingProduction 获取建筑的资源产量
func (bm *BuildingManager) GetBuildingProduction(building *Building) map[string]int32 {
	if building.Level == 0 {
		return nil
	}

	config, err := bm.GetBuildingConfig(building.Type, building.Level)
	if err != nil {
		return nil
	}

	return config.Production
}
