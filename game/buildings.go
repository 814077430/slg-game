package game

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"slg-game/database"
)

// BuildingType 定义建筑类型
type BuildingType string

const (
	BuildingTypeTownHall    BuildingType = "town_hall"     // 城堡/市政厅
	BuildingTypeBarracks   BuildingType = "barracks"      // 兵营
	BuildingTypeStable     BuildingType = "stable"        // 马厩
	BuildingTypeArchery    BuildingType = "archery_range" // 射箭场
	BuildingTypeFarm       BuildingType = "farm"          // 农场
	BuildingTypeLumberMill BuildingType = "lumber_mill"   // 伐木场
	BuildingTypeMine       BuildingType = "mine"          // 矿场
	BuildingTypeWall       BuildingType = "wall"          // 城墙
	BuildingTypeWatchTower BuildingType = "watch_tower"   // 瞭望塔
	BuildingTypeAcademy    BuildingType = "academy"       // 学院
)

// BuildingLevelConfig 建筑等级配置
type BuildingLevelConfig struct {
	Level         int32             `bson:"level" json:"level"`
	BuildTime     int64             `bson:"build_time" json:"build_time"` // 建造时间（秒）
	ResourceCost  map[string]int32  `bson:"resource_cost" json:"resource_cost"`
	Production    map[string]int32  `bson:"production" json:"production"` // 每小时产量
	Capacity      map[string]int32  `bson:"capacity" json:"capacity"`     // 容量（仅存储建筑）
	Stats         map[string]int32  `bson:"stats" json:"stats"`           // 属性加成
	Requirements  map[string]int32  `bson:"requirements" json:"requirements"` // 前置要求
}

// Building 建筑实例
type Building struct {
	ID            string         `bson:"_id,omitempty" json:"id"`
	PlayerID      uint64         `bson:"player_id" json:"player_id"`
	Type          BuildingType   `bson:"type" json:"type"`
	Level         int32          `bson:"level" json:"level"`
	X             int32          `bson:"x" json:"x"`
	Y             int32          `bson:"y" json:"y"`
	Construction  *Construction  `bson:"construction,omitempty" json:"construction,omitempty"`
	LastCollected time.Time      `bson:"last_collected" json:"last_collected"`
	CreatedAt     time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `bson:"updated_at" json:"updated_at"`
}

// Construction 正在建造的建筑
type Construction struct {
	TargetLevel int32     `bson:"target_level" json:"target_level"`
	StartTime   time.Time `bson:"start_time" json:"start_time"`
	EndTime     time.Time `bson:"end_time" json:"end_time"`
}

// BuildingManager 建筑管理器
type BuildingManager struct {
	db *database.Database
}

func NewBuildingManager(db *database.Database) *BuildingManager {
	return &BuildingManager{
		db: db,
	}
}

// GetBuildingConfig 获取建筑配置
func (bm *BuildingManager) GetBuildingConfig(buildingType BuildingType, level int32) (*BuildingLevelConfig, error) {
	// 这里应该从配置文件或数据库中加载建筑配置
	// 为了简化，我们使用硬编码的配置
	config := bm.getHardcodedConfig(buildingType, level)
	if config == nil {
		return nil, fmt.Errorf("building config not found for type %s level %d", buildingType, level)
	}
	return config, nil
}

// getHardcodedConfig 获取硬编码的建筑配置（实际项目中应该从配置文件加载）
func (bm *BuildingManager) getHardcodedConfig(buildingType BuildingType, level int32) *BuildingLevelConfig {
	baseConfigs := map[BuildingType]map[int32]*BuildingLevelConfig{
		BuildingTypeTownHall: {
			1: {Level: 1, BuildTime: 60, ResourceCost: map[string]int32{"gold": 100, "wood": 50}, Stats: map[string]int32{"max_buildings": 10}},
			2: {Level: 2, BuildTime: 120, ResourceCost: map[string]int32{"gold": 200, "wood": 100}, Stats: map[string]int32{"max_buildings": 15}},
			3: {Level: 3, BuildTime: 300, ResourceCost: map[string]int32{"gold": 400, "wood": 200}, Stats: map[string]int32{"max_buildings": 20}},
		},
		BuildingTypeFarm: {
			1: {Level: 1, BuildTime: 30, ResourceCost: map[string]int32{"wood": 50}, Production: map[string]int32{"food": 10}, Capacity: map[string]int32{"food": 100}},
			2: {Level: 2, BuildTime: 60, ResourceCost: map[string]int32{"wood": 100}, Production: map[string]int32{"food": 20}, Capacity: map[string]int32{"food": 200}},
			3: {Level: 3, BuildTime: 120, ResourceCost: map[string]int32{"wood": 200}, Production: map[string]int32{"food": 40}, Capacity: map[string]int32{"food": 400}},
		},
		BuildingTypeLumberMill: {
			1: {Level: 1, BuildTime: 30, ResourceCost: map[string]int32{"wood": 50}, Production: map[string]int32{"wood": 10}, Capacity: map[string]int32{"wood": 100}},
			2: {Level: 2, BuildTime: 60, ResourceCost: map[string]int32{"wood": 100}, Production: map[string]int32{"wood": 20}, Capacity: map[string]int32{"wood": 200}},
			3: {Level: 3, BuildTime: 120, ResourceCost: map[string]int32{"wood": 200}, Production: map[string]int32{"wood": 40}, Capacity: map[string]int32{"wood": 400}},
		},
		BuildingTypeMine: {
			1: {Level: 1, BuildTime: 30, ResourceCost: map[string]int32{"wood": 50}, Production: map[string]int32{"gold": 10}, Capacity: map[string]int32{"gold": 100}},
			2: {Level: 2, BuildTime: 60, ResourceCost: map[string]int32{"wood": 100}, Production: map[string]int32{"gold": 20}, Capacity: map[string]int32{"gold": 200}},
			3: {Level: 3, BuildTime: 120, ResourceCost: map[string]int32{"wood": 200}, Production: map[string]int32{"gold": 40}, Capacity: map[string]int32{"gold": 400}},
		},
		BuildingTypeBarracks: {
			1: {Level: 1, BuildTime: 120, ResourceCost: map[string]int32{"gold": 200, "wood": 100}, Stats: map[string]int32{"can_train_infantry": 1}},
			2: {Level: 2, BuildTime: 240, ResourceCost: map[string]int32{"gold": 400, "wood": 200}, Stats: map[string]int32{"can_train_infantry": 1, "training_speed": 2}},
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
	// 检查位置是否已被占用
	exists, err := bm.isPositionOccupied(playerID, x, y)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("position (%d, %d) is already occupied", x, y)
	}

	building := &Building{
		PlayerID:  playerID,
		Type:      buildingType,
		Level:     0, // 初始为0级，需要建造到1级
		X:         x,
		Y:         y,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 开始建造到1级
	err = bm.startConstruction(building, 1)
	if err != nil {
		return nil, err
	}

	// 保存到数据库
	collection := bm.db.GetCollection("buildings")
	_, err = collection.InsertOne(context.Background(), building)
	if err != nil {
		return nil, err
	}

	return building, nil
}

// UpgradeBuilding 升级建筑
func (bm *BuildingManager) UpgradeBuilding(buildingID string, targetLevel int32) error {
	collection := bm.db.GetCollection("buildings")
	filter := bson.M{"_id": buildingID}

	// 获取当前建筑信息
	var building Building
	err := collection.FindOne(context.Background(), filter).Decode(&building)
	if err != nil {
		return err
	}

	// 检查是否已经在建造中
	if building.Construction != nil {
		return fmt.Errorf("building is already under construction")
	}

	// 检查目标等级是否有效
	if targetLevel <= building.Level {
		return fmt.Errorf("invalid target level")
	}

	// 开始升级建造
	err = bm.startConstruction(&building, targetLevel)
	if err != nil {
		return err
	}

	// 更新数据库
	update := bson.M{
		"$set": bson.M{
			"construction": building.Construction,
			"updated_at":   time.Now(),
		},
	}
	_, err = collection.UpdateOne(context.Background(), filter, update)
	return err
}

// CompleteConstruction 完成建筑建造
func (bm *BuildingManager) CompleteConstruction(buildingID string) error {
	collection := bm.db.GetCollection("buildings")
	filter := bson.M{"_id": buildingID}

	var building Building
	err := collection.FindOne(context.Background(), filter).Decode(&building)
	if err != nil {
		return err
	}

	if building.Construction == nil {
		return fmt.Errorf("no construction in progress")
	}

	// 更新建筑等级
	building.Level = building.Construction.TargetLevel
	building.Construction = nil
	building.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"level":      building.Level,
			"construction": nil,
			"updated_at": time.Now(),
		},
	}
	_, err = collection.UpdateOne(context.Background(), filter, update)
	return err
}

// CancelConstruction 取消建筑建造
func (bm *BuildingManager) CancelConstruction(buildingID string) error {
	collection := bm.db.GetCollection("buildings")
	filter := bson.M{"_id": buildingID}

	update := bson.M{
		"$set": bson.M{
			"construction": nil,
			"updated_at":   time.Now(),
		},
	}
	_, err := collection.UpdateOne(context.Background(), filter, update)
	return err
}

// GetPlayerBuildings 获取玩家所有建筑
func (bm *BuildingManager) GetPlayerBuildings(playerID uint64) ([]*Building, error) {
	collection := bm.db.GetCollection("buildings")
	filter := bson.M{"player_id": playerID}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var buildings []*Building
	if err = cursor.All(context.Background(), &buildings); err != nil {
		return nil, err
	}

	return buildings, nil
}

// startConstruction 开始建筑建造
func (bm *BuildingManager) startConstruction(building *Building, targetLevel int32) error {
	config, err := bm.GetBuildingConfig(building.Type, targetLevel)
	if err != nil {
		return err
	}

	// 检查资源是否足够
	// 这里应该调用资源管理器检查并扣除资源
	// 为了简化，我们假设资源足够

	building.Construction = &Construction{
		TargetLevel: targetLevel,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Duration(config.BuildTime) * time.Second),
	}

	return nil
}

// isPositionOccupied 检查位置是否被占用
func (bm *BuildingManager) isPositionOccupied(playerID uint64, x, y int32) (bool, error) {
	collection := bm.db.GetCollection("buildings")
	filter := bson.M{
		"player_id": playerID,
		"x":         x,
		"y":         y,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetBuildingProduction 获取建筑的资源产量
func (bm *BuildingManager) GetBuildingProduction(building *Building) map[string]int32 {
	if building.Level == 0 {
		return nil
	}

	config, err := bm.GetBuildingConfig(building.Type, building.Level)
	if err != nil {
		log.Printf("Failed to get building config: %v", err)
		return nil
	}

	return config.Production
}

// GetBuildingCapacity 获取建筑的资源容量
func (bm *BuildingManager) GetBuildingCapacity(building *Building) map[string]int32 {
	if building.Level == 0 {
		return nil
	}

	config, err := bm.GetBuildingConfig(building.Type, building.Level)
	if err != nil {
		log.Printf("Failed to get building config: %v", err)
		return nil
	}

	return config.Capacity
}