package world

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"slg-game/database"
)

// World represents the game world with all tiles and players
type World struct {
	db            database.DB
	tiles         map[WorldCoord]*WorldTile
	players       map[uint64]map[string]interface{}
	mutex         sync.RWMutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
	tickInterval  time.Duration
	currentTick   uint64
}

// WorldCoord represents a coordinate in the world
type WorldCoord struct {
	X int32
	Y int32
}

// WorldTile represents a single tile in the world
type WorldTile struct {
	Coord      WorldCoord       `json:"coord"`
	TileType   string           `json:"tile_type"`
	OwnerID    uint64           `json:"owner_id"`
	BuildingID string           `json:"building_id"`
	Resource   map[string]int32 `json:"resource"`
}

// NewWorld creates a new world instance
func NewWorld(db database.DB) *World {
	world := &World{
		db:           db,
		tiles:        make(map[WorldCoord]*WorldTile),
		players:      make(map[uint64]map[string]interface{}),
		stopChan:     make(chan struct{}),
		tickInterval: 1000 * time.Millisecond, // 1 秒
		currentTick:  0,
	}

	log.Println("[World] World initialized")

	return world
}

// StartLoop 启动世界独立循环（独立 Goroutine）
func (w *World) StartLoop() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		log.Printf("[World] World loop started with tick interval: %v", w.tickInterval)
		
		ticker := time.NewTicker(w.tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.tick()
			case <-w.stopChan:
				log.Println("[World] World loop stopping...")
				return
			}
		}
	}()
}

// StopLoop 停止世界循环
func (w *World) StopLoop() {
	close(w.stopChan)
	w.wg.Wait()
	log.Println("[World] World loop stopped")
}

// tick 执行一个世界 tick
func (w *World) tick() {
	w.mutex.Lock()
	w.currentTick++
	tick := w.currentTick
	w.mutex.Unlock()

	// 每 10 个 tick 记录一次状态
	if tick%10 == 0 {
		log.Printf("[World] Tick: %d", tick)
	}

	// 处理资源生成
	w.processResourceGeneration()

	// 处理世界事件
	w.processWorldEvents()

	// 清理过期数据
	w.processCleanup()
}

// processResourceGeneration 处理资源生成
func (w *World) processResourceGeneration() {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for _, tile := range w.tiles {
		if tile.BuildingID != "" && tile.OwnerID != 0 {
			// 建筑资源生产逻辑
			// TODO: 根据建筑类型生产资源
		}
	}
}

// processWorldEvents 处理世界事件
func (w *World) processWorldEvents() {
	// TODO: 随机事件、天气变化等
}

// processCleanup 清理过期数据
func (w *World) processCleanup() {
	// TODO: 清理不活跃玩家的地块等
}

// GetTile gets a tile at the specified coordinates
func (w *World) GetTile(x, y int32) *WorldTile {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	coord := WorldCoord{X: x, Y: y}
	if tile, exists := w.tiles[coord]; exists {
		return tile
	}

	defaultTile := &WorldTile{
		Coord:    coord,
		TileType: "grass",
		OwnerID:  0,
		Resource: map[string]int32{"gold": 0, "wood": 0, "food": 0, "stone": 0},
	}

	return defaultTile
}

// SetTile sets a tile at the specified coordinates
func (w *World) SetTile(tile *WorldTile) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.tiles[tile.Coord] = tile
	return nil
}

// GetTilesInArea gets all tiles in a rectangular area
func (w *World) GetTilesInArea(x1, y1, x2, y2 int32) []*WorldTile {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var tiles []*WorldTile
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
			coord := WorldCoord{X: x, Y: y}
			if tile, exists := w.tiles[coord]; exists {
				tiles = append(tiles, tile)
			} else {
				defaultTile := &WorldTile{
					Coord:    coord,
					TileType: "grass",
					OwnerID:  0,
					Resource: map[string]int32{"gold": 0, "wood": 0, "food": 0, "stone": 0},
				}
				tiles = append(tiles, defaultTile)
			}
		}
	}

	return tiles
}

// ClaimTile allows a player to claim a tile
func (w *World) ClaimTile(playerID uint64, x, y int32) error {
	tile := w.GetTile(x, y)

	if tile.OwnerID != 0 {
		return nil // Tile already claimed
	}

	tile.OwnerID = playerID
	return w.SetTile(tile)
}

// AddPlayer adds a player to the world
func (w *World) AddPlayer(playerID uint64, playerData map[string]interface{}) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.players[playerID] = playerData
	log.Printf("[World] Player %d added to world", playerID)
}

// RemovePlayer removes a player from the world
func (w *World) RemovePlayer(playerID uint64) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	delete(w.players, playerID)
	log.Printf("[World] Player %d removed from world", playerID)
}

// GetPlayer gets a player by ID
func (w *World) GetPlayer(playerID uint64) map[string]interface{} {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return w.players[playerID]
}

// GetTick 获取当前 tick 数
func (w *World) GetTick() uint64 {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.currentTick
}

// GenerateWorld generates a new world with specified dimensions
func (w *World) GenerateWorld(width, height int32) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	log.Printf("[World] Generating world of size %dx%d", width, height)

	for x := int32(0); x < width; x++ {
		for y := int32(0); y < height; y++ {
			tileType := "grass"
			if x%10 == 0 && y%10 == 0 {
				tileType = "water"
			} else if x%7 == 0 || y%7 == 0 {
				tileType = "mountain"
			}

			resource := map[string]int32{
				"gold":  rand.Int31n(10),
				"wood":  rand.Int31n(15),
				"food":  rand.Int31n(20),
				"stone": rand.Int31n(8),
			}

			tile := &WorldTile{
				Coord:    WorldCoord{X: x, Y: y},
				TileType: tileType,
				OwnerID:  0,
				Resource: resource,
			}

			w.tiles[tile.Coord] = tile
		}
	}

	log.Printf("[World] World generated with %d tiles", len(w.tiles))
}
