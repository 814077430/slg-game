package resource

import (
	"log"
	"sync"
	"time"

	"slg-game/database"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceGold  ResourceType = "gold"
	ResourceWood  ResourceType = "wood"
	ResourceFood  ResourceType = "food"
	ResourceStone ResourceType = "stone"
	ResourceIron  ResourceType = "iron"
)

// ResourceManager 资源管理器
type ResourceManager struct {
	db database.DB
}

// NewResourceManager 创建资源管理器
func NewResourceManager(db database.DB) *ResourceManager {
	return &ResourceManager{db: db}
}

// GetPlayerResources 获取玩家资源
func (rm *ResourceManager) GetPlayerResources(playerID uint64) (map[string]int64, error) {
	collection := rm.db.GetCollection("players")
	player, err := collection.FindOne(map[string]interface{}{"player_id": playerID})
	if err != nil {
		return nil, err
	}

	return map[string]int64{
		"gold":  player["gold"].(int64),
		"wood":  player["wood"].(int64),
		"food":  player["food"].(int64),
		"stone": 0,
		"iron":  0,
	}, nil
}

// CanAfford 检查玩家是否能负担资源消耗
func (rm *ResourceManager) CanAfford(playerID uint64, costs map[string]int64) (bool, error) {
	resources, err := rm.GetPlayerResources(playerID)
	if err != nil {
		return false, err
	}

	for resource, cost := range costs {
		if resources[resource] < cost {
			return false, nil
		}
	}
	return true, nil
}

// DeductResources 扣除玩家资源
func (rm *ResourceManager) DeductResources(playerID uint64, costs map[string]int64) error {
	collection := rm.db.GetCollection("players")

	player, err := collection.FindOne(map[string]interface{}{"player_id": playerID})
	if err != nil {
		return err
	}

	update := make(map[string]interface{})
	for resource, cost := range costs {
		switch resource {
		case "gold":
			update["gold"] = player["gold"].(int64) - cost
		case "wood":
			update["wood"] = player["wood"].(int64) - cost
		case "food":
			update["food"] = player["food"].(int64) - cost
		}
	}

	return collection.UpdateOne(map[string]interface{}{"player_id": playerID}, update)
}

// AddResources 增加玩家资源
func (rm *ResourceManager) AddResources(playerID uint64, gains map[string]int64) error {
	collection := rm.db.GetCollection("players")

	player, err := collection.FindOne(map[string]interface{}{"player_id": playerID})
	if err != nil {
		return err
	}

	update := make(map[string]interface{})
	for resource, gain := range gains {
		switch resource {
		case "gold":
			update["gold"] = player["gold"].(int64) + gain
		case "wood":
			update["wood"] = player["wood"].(int64) + gain
		case "food":
			update["food"] = player["food"].(int64) + gain
		}
	}

	return collection.UpdateOne(map[string]interface{}{"player_id": playerID}, update)
}

// ResourceCollector 资源收集器
type ResourceCollector struct {
	rm       *ResourceManager
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewResourceCollector 创建资源收集器
func NewResourceCollector(rm *ResourceManager, interval time.Duration) *ResourceCollector {
	return &ResourceCollector{
		rm:       rm,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start 启动资源收集
func (rc *ResourceCollector) Start() {
	rc.wg.Add(1)
	go func() {
		defer rc.wg.Done()
		ticker := time.NewTicker(rc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rc.collect()
			case <-rc.stopChan:
				return
			}
		}
	}()
}

// Stop 停止资源收集
func (rc *ResourceCollector) Stop() {
	close(rc.stopChan)
	rc.wg.Wait()
}

// collect 收集逻辑
func (rc *ResourceCollector) collect() {
	collection := rc.rm.db.GetCollection("players")
	players := collection.GetAll()

	for _, player := range players {
		playerID := player["player_id"].(uint64)
		
		// 基础资源产量
		gains := map[string]int64{
			"gold": 10,
			"wood": 10,
			"food": 10,
		}
		
		err := rc.rm.AddResources(playerID, gains)
		if err != nil {
			log.Printf("Failed to add resources for player %d: %v", playerID, err)
		}
	}

	log.Printf("Resource collection completed for %d players", len(players))
}
