package game

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"slg-game/database"
)

// ResourceManager 资源管理器
type ResourceManager struct {
	db *database.Database
}

// NewResourceManager 创建资源管理器
func NewResourceManager(db *database.Database) *ResourceManager {
	return &ResourceManager{db: db}
}

// GetPlayerResources 获取玩家资源
func (rm *ResourceManager) GetPlayerResources(playerID uint64) (map[string]int64, error) {
	collection := rm.db.GetCollection("players")
	var player database.Player
	err := collection.FindOne(context.Background(), bson.M{"player_id": playerID}).Decode(&player)
	if err != nil {
		return nil, err
	}

	return map[string]int64{
		"gold":  player.Gold,
		"wood":  player.Wood,
		"food":  player.Food,
		"stone": 0, // 暂未实现
		"iron":  0, // 暂未实现
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

	update := bson.M{}
	for resource, cost := range costs {
		switch resource {
		case "gold":
			update["gold"] = bson.M{"$inc": -cost}
		case "wood":
			update["wood"] = bson.M{"$inc": -cost}
		case "food":
			update["food"] = bson.M{"$inc": -cost}
		}
	}

	if len(update) == 0 {
		return nil
	}

	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"player_id": playerID},
		update,
	)
	return err
}

// AddResources 增加玩家资源
func (rm *ResourceManager) AddResources(playerID uint64, gains map[string]int64) error {
	collection := rm.db.GetCollection("players")

	update := bson.M{}
	for resource, gain := range gains {
		switch resource {
		case "gold":
			update["gold"] = bson.M{"$inc": gain}
		case "wood":
			update["wood"] = bson.M{"$inc": gain}
		case "food":
			update["food"] = bson.M{"$inc": gain}
		}
	}

	if len(update) == 0 {
		return nil
	}

	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"player_id": playerID},
		update,
	)
	return err
}

// ResourceCollector 资源收集器（定时收集）
type ResourceCollector struct {
	rm        *ResourceManager
	interval  time.Duration
	stopChan  chan struct{}
	wg        sync.WaitGroup
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
	// 获取所有在线玩家
	collection := rc.rm.db.GetCollection("players")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Failed to get players for resource collection: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var players []database.Player
	if err := cursor.All(context.Background(), &players); err != nil {
		log.Printf("Failed to decode players: %v", err)
		return
	}

	// 为每个玩家添加资源（基于建筑产量）
	for _, player := range players {
		// 基础资源产量（后续根据建筑计算）
		gains := map[string]int64{
			"gold": 10,
			"wood": 10,
			"food": 10,
		}
		err := rc.rm.AddResources(player.PlayerID, gains)
		if err != nil {
			log.Printf("Failed to add resources for player %d: %v", player.PlayerID, err)
		}
	}

	log.Printf("Resource collection completed for %d players", len(players))
}
