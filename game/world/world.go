package world

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"slg-game/database"
)

const (
	// WorldSize 世界大小 1024x1024
	WorldSize = 1024
	
	// CenterOffset 中心偏移（王城区域）
	CenterOffset = 50
	
	// CastleSize 王城大小
	CastleSize = 100
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
	width         int32
	height        int32
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
	Zone       string           `json:"zone"` // 区域类型
}

// NewWorld creates a new world instance
func NewWorld(db database.DB) *World {
	world := &World{
		db:           db,
		tiles:        make(map[WorldCoord]*WorldTile),
		players:      make(map[uint64]map[string]interface{}),
		stopChan:     make(chan struct{}),
		tickInterval: 1000 * time.Millisecond,
		currentTick:  0,
		width:        WorldSize,
		height:       WorldSize,
	}

	log.Printf("[World] World initialized (%dx%d)", WorldSize, WorldSize)

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

	// 边界检查
	if x < 0 || x >= w.width || y < 0 || y >= w.height {
		return nil
	}

	coord := WorldCoord{X: x, Y: y}
	if tile, exists := w.tiles[coord]; exists {
		return tile
	}

	// 动态创建地块
	tile := w.createDefaultTile(x, y)
	w.tiles[coord] = tile
	return tile
}

// createDefaultTile 创建默认地块
func (w *World) createDefaultTile(x, y int32) *WorldTile {
	tileType := w.getTileType(x, y)
	zone := w.getZoneType(x, y)
	
	return &WorldTile{
		Coord:    WorldCoord{X: x, Y: y},
		TileType: tileType,
		OwnerID:  0,
		Zone:     zone,
		Resource: w.getResourceAmount(tileType, zone),
	}
}

// getTileType 根据坐标获取地形类型
func (w *World) getTileType(x, y int32) string {
	// 中心王城区域
	if w.isCastleZone(x, y) {
		return "castle"
	}
	
	// 王城周边安全区
	if w.isSafeZone(x, y) {
		return "plain"
	}
	
	// 使用种子生成固定地形
	seed := int64(x*10000 + y)
	r := rand.New(rand.NewSource(seed))
	
	randVal := r.Float32()
	
	// 地形分布
	switch {
	case randVal < 0.05: // 5% 水域
		return "water"
	case randVal < 0.10: // 5% 山脉
		return "mountain"
	case randVal < 0.20: // 10% 森林
		return "forest"
	case randVal < 0.25: // 5% 沙漠
		return "desert"
	case randVal < 0.30: // 5% 草地（资源丰富）
		return "grass_rich"
	default: // 70% 普通草地
		return "grass"
	}
}

// getZoneType 获取区域类型
func (w *World) getZoneType(x, y int32) string {
	if w.isCastleZone(x, y) {
		return "castle" // 王城区域
	}
	
	if w.isSafeZone(x, y) {
		return "safe" // 安全区
	}
	
	// 按距离划分区域
	centerX := int32(WorldSize / 2)
	centerY := int32(WorldSize / 2)
	
	dist := abs(x-centerX) + abs(y-centerY)
	
	switch {
	case dist < 100:
		return "center" // 中心区域
	case dist < 256:
		return "east" // 东区
	case dist < 512:
		return "south" // 南区
	case dist < 768:
		return "west" // 西区
	default:
		return "north" // 北区
	}
}

// isCastleZone 是否在王城区域内
func (w *World) isCastleZone(x, y int32) bool {
	center := WorldSize / 2
	halfCastle := CastleSize / 2
	
	return x >= int32(center-halfCastle) && x <= int32(center+halfCastle) &&
		   y >= int32(center-halfCastle) && y <= int32(center+halfCastle)
}

// isSafeZone 是否在安全区内
func (w *World) isSafeZone(x, y int32) bool {
	center := WorldSize / 2
	halfSafe := CenterOffset
	
	return x >= int32(center-halfSafe) && x <= int32(center+halfSafe) &&
		   y >= int32(center-halfSafe) && y <= int32(center+halfSafe)
}

// getResourceAmount 获取资源量
func (w *World) getResourceAmount(tileType, zone string) map[string]int32 {
	baseResources := map[string]int32{
		"gold":  0,
		"wood":  0,
		"food":  0,
		"stone": 0,
	}
	
	// 根据地形分配基础资源
	switch tileType {
	case "forest":
		baseResources["wood"] = 20 + rand.Int31n(30)
		baseResources["food"] = 5 + rand.Int31n(10)
	case "grass":
		baseResources["food"] = 15 + rand.Int31n(20)
		baseResources["wood"] = 5 + rand.Int31n(10)
	case "grass_rich":
		baseResources["food"] = 30 + rand.Int31n(40)
		baseResources["wood"] = 15 + rand.Int31n(20)
		baseResources["gold"] = 5 + rand.Int31n(10)
	case "mountain":
		baseResources["stone"] = 20 + rand.Int31n(30)
		baseResources["gold"] = 5 + rand.Int31n(15)
	case "desert":
		baseResources["gold"] = 10 + rand.Int31n(20)
	case "castle":
		baseResources["gold"] = 100
		baseResources["wood"] = 100
		baseResources["food"] = 100
		baseResources["stone"] = 100
	}
	
	// 区域加成
	switch zone {
	case "center":
		for k, v := range baseResources {
			baseResources[k] = v + v/4 // +25%
		}
	case "castle":
		for k, v := range baseResources {
			baseResources[k] = v * 2 // +100%
		}
	}
	
	return baseResources
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
			// 边界检查
			if x < 0 || x >= w.width || y < 0 || y >= w.height {
				continue
			}
			
			coord := WorldCoord{X: x, Y: y}
			if tile, exists := w.tiles[coord]; exists {
				tiles = append(tiles, tile)
			} else {
				defaultTile := w.createDefaultTile(x, y)
				tiles = append(tiles, defaultTile)
			}
		}
	}

	return tiles
}

// ClaimTile allows a player to claim a tile
func (w *World) ClaimTile(playerID uint64, x, y int32) error {
	tile := w.GetTile(x, y)

	if tile == nil {
		return nil // 超出边界
	}
	
	// 王城区域不可占领
	if tile.Zone == "castle" {
		return nil
	}

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

// GenerateWorld generates the full world map
func (w *World) GenerateWorld() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	log.Printf("[World] Generating world of size %dx%d", w.width, w.height)
	log.Printf("[World] Center Castle: (%d, %d) size %dx%d", 
		WorldSize/2, WorldSize/2, CastleSize, CastleSize)

	startTime := time.Now()
	
	// 预生成所有地块
	for x := int32(0); x < w.width; x++ {
		for y := int32(0); y < w.height; y++ {
			tileType := w.getTileType(x, y)
			zone := w.getZoneType(x, y)
			
			tile := &WorldTile{
				Coord:    WorldCoord{X: x, Y: y},
				TileType: tileType,
				OwnerID:  0,
				Zone:     zone,
				Resource: w.getResourceAmount(tileType, zone),
			}

			w.tiles[tile.Coord] = tile
		}
		
		// 每生成 10% 输出进度
		if (x+1)%(w.width/10) == 0 {
			log.Printf("[World] Generation progress: %d%%", ((x+1)*100)/w.width)
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("[World] World generated with %d tiles in %v", len(w.tiles), elapsed)
	
	// 统计各地形数量
	w.printTerrainStats()
}

// printTerrainStats 打印地形统计
func (w *World) printTerrainStats() {
	stats := make(map[string]int)
	for _, tile := range w.tiles {
		stats[tile.TileType]++
	}
	
	log.Println("[World] Terrain Statistics:")
	for tileType, count := range stats {
		percent := float64(count) / float64(len(w.tiles)) * 100
		log.Printf("  - %s: %d (%.2f%%)", tileType, count, percent)
	}
}

// GetWorldSize 获取世界大小
func (w *World) GetWorldSize() (int32, int32) {
	return w.width, w.height
}

// GetCastleInfo 获取王城信息
func (w *World) GetCastleInfo() map[string]interface{} {
	center := WorldSize / 2
	halfCastle := CastleSize / 2
	
	return map[string]interface{}{
		"center_x": center,
		"center_y": center,
		"top_left_x": center - halfCastle,
		"top_left_y": center - halfCastle,
		"bottom_right_x": center + halfCastle,
		"bottom_right_y": center + halfCastle,
		"size": CastleSize,
		"protected": true,
	}
}

// abs 绝对值辅助函数
func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
