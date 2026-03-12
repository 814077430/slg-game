package world

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"slg-game/database"
	"slg-game/messenger"
)

// 世界尺寸常量
const (
	WorldSize       int32 = 1024 // 世界大小 1024x1024
	CenterSize      int32 = 256  // 中心安全区 256x256
	StateSize       int32 = 256  // 每个州 256x256
	CastleSize      int32 = 64   // 皇城 64x64
	CitySize        int32 = 32   // 县城 32x32
	BarbarianWidth  int32 = 128  // 蛮荒带宽度
	EdgeWidth       int32 = 64   // 边缘绝境带宽度
)

// 坐标常量
const (
	CenterStart = (WorldSize - CenterSize) / 2 // 384
	CenterEnd   = CenterStart + CenterSize     // 640
	CastleStart = (WorldSize - CastleSize) / 2 // 480
	CastleEnd   = CastleStart + CastleSize     // 544
)

// ZoneType 区域类型
type ZoneType string

const (
	ZoneCastle    ZoneType = "castle"     // 皇城
	ZoneSafe      ZoneType = "safe"       // 安全区
	ZoneQing      ZoneType = "qing"       // 青州（东北）
	ZoneJing      ZoneType = "jing"       // 荆州（东南）
	ZoneYong      ZoneType = "yong"       // 雍州（西北）
	ZoneYang      ZoneType = "yang"       // 扬州（西南）
	ZoneBarbarian ZoneType = "barbarian"  // 蛮荒带
	ZoneEdge      ZoneType = "edge"       // 边缘绝境
)

// TileType 地形类型
type TileType string

const (
	TilePlain    TileType = "plain"    // 平原 30%
	TileForest   TileType = "forest"   // 森林 25%
	TileMountain TileType = "mountain" // 山地 15%
	TileHill     TileType = "hill"     // 丘陵 15%
	TileRiver    TileType = "river"    // 河流 10%
	TileDesert   TileType = "desert"   // 荒漠 5%
	TileSnow     TileType = "snow"     // 雪山 5%
)

// ResourceLevel 资源等级（从外到内 0-6 级）
type ResourceLevel int

const (
	ResourceLevel0 ResourceLevel = iota // 0 级资源（边缘绝境，无资源）
	ResourceLevel1                      // 1 级资源（蛮荒带外层）
	ResourceLevel2                      // 2 级资源（蛮荒带内层）
	ResourceLevel3                      // 3 级资源（四大州外层）
	ResourceLevel4                      // 4 级资源（四大州内层）
	ResourceLevel5                      // 5 级资源（中心安全区）
	ResourceLevel6                      // 6 级资源（皇城，顶级）
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceGold  ResourceType = "gold"  // 金币
	ResourceWood  ResourceType = "wood"  // 木材
	ResourceFood  ResourceType = "food"  // 粮食
	ResourceStone ResourceType = "stone" // 石料
)

// World 游戏世界
type World struct {
	db            database.DB
	messageBus    *messenger.MessageBus
	tiles         map[WorldCoord]*WorldTile
	players       map[uint64]map[string]interface{}
	mutex         sync.RWMutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
	tickInterval  time.Duration
	currentTick   uint64
	width         int32
	height        int32
	generated     bool
}

// WorldCoord 世界坐标
type WorldCoord struct {
	X int32
	Y int32
}

// WorldTile 世界地块
type WorldTile struct {
	Coord        WorldCoord       `json:"coord"`
	TileType     TileType         `json:"tile_type"`
	Zone         ZoneType         `json:"zone"`
	OwnerID      uint64           `json:"owner_id"`
	BuildingID   string           `json:"building_id"`
	Resources    map[string]int32 `json:"resources"`    // 资源点 {resource_type: amount}
	ResourceLvl  ResourceLevel    `json:"resource_lvl"` // 资源等级
	Passable     bool             `json:"passable"`
	CityType     string           `json:"city_type"` // 城市类型：castle/state/county
	ResourceSpot bool             `json:"resource_spot"` // 是否是资源点
}

// NewWorld 创建世界实例
func NewWorld(db database.DB, messageBus *messenger.MessageBus) *World {
	world := &World{
		db:           db,
		messageBus:   messageBus,
		tiles:        make(map[WorldCoord]*WorldTile),
		players:      make(map[uint64]map[string]interface{}),
		stopChan:     make(chan struct{}),
		tickInterval: 1000 * time.Millisecond,
		width:        WorldSize,
		height:       WorldSize,
	}

	log.Printf("[World] World initialized (%dx%d)", WorldSize, WorldSize)
	log.Printf("[World] Center: (%d,%d)~(%d,%d) Size: %dx%d",
		CenterStart, CenterStart, CenterEnd, CenterEnd, CenterSize, CenterSize)
	log.Printf("[World] Castle: (%d,%d)~(%d,%d) Size: %dx%d",
		CastleStart, CastleStart, CastleEnd, CastleEnd, CastleSize, CastleSize)

	return world
}

// StartLoop 启动世界循环
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

// tick 执行世界 tick
func (w *World) tick() {
	w.mutex.Lock()
	w.currentTick++
	tick := w.currentTick
	w.mutex.Unlock()

	if tick%100 == 0 {
		log.Printf("[World] Tick: %d", tick)
	}

	w.processResourceGeneration()
	w.processWorldEvents()
	w.processCleanup()
}

// processResourceGeneration 资源生成
func (w *World) processResourceGeneration() {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for _, tile := range w.tiles {
		if tile.BuildingID != "" && tile.OwnerID != 0 {
			// 建筑资源生产
		}
	}
}

// processWorldEvents 世界事件
func (w *World) processWorldEvents() {
	// TODO: 随机事件
}

// processCleanup 清理
func (w *World) processCleanup() {
	// TODO: 清理过期数据
}

// GetZoneType 获取区域类型
func (w *World) GetZoneType(x, y int32) ZoneType {
	// 边缘绝境带（最外圈 64 格）
	if x < EdgeWidth || x >= WorldSize-EdgeWidth ||
		y < EdgeWidth || y >= WorldSize-EdgeWidth {
		return ZoneEdge
	}

	// 中心区域 256x256
	if x >= CenterStart && x < CenterEnd && y >= CenterStart && y < CenterEnd {
		// 皇城 64x64
		if x >= CastleStart && x < CastleEnd && y >= CastleStart && y < CastleEnd {
			return ZoneCastle
		}
		return ZoneSafe
	}

	// 蛮荒带（中心区外 128 格）
	barbarianStart := CenterEnd
	barbarianEnd := barbarianStart + BarbarianWidth

	if (x >= barbarianStart && x < barbarianEnd) ||
		(x >= WorldSize-barbarianEnd && x < WorldSize-barbarianStart) ||
		(y >= barbarianStart && y < barbarianEnd) ||
		(y >= WorldSize-barbarianEnd && y < WorldSize-barbarianStart) {
		return ZoneBarbarian
	}

	// 四大州
	mid := WorldSize / 2 // 512

	if x < mid && y < mid {
		return ZoneYong // 雍州（西北）
	} else if x >= mid && y < mid {
		return ZoneQing // 青州（东北）
	} else if x < mid && y >= mid {
		return ZoneYang // 扬州（西南）
	} else {
		return ZoneJing // 荆州（东南）
	}
}

// GetResourceLevel 获取资源等级（从外到内递增）
func (w *World) GetResourceLevel(x, y int32, zone ZoneType) ResourceLevel {
	// 计算到中心的距离
	centerX := WorldSize / 2
	centerY := WorldSize / 2
	dx := abs32(x - centerX)
	dy := abs32(y - centerY)
	distance := dx + dy // 曼哈顿距离

	// 根据距离计算资源等级（从外到内 0-6 级）
	// 边缘 (distance > 700): 0 级
	// 蛮荒带外层 (600-700): 1 级
	// 蛮荒带内层 (500-600): 2 级
	// 四大州外层 (400-500): 3 级
	// 四大州内层 (300-400): 4 级
	// 中心安全区 (200-300): 5 级
	// 皇城 (< 200): 6 级

	switch {
	case distance > 700:
		return ResourceLevel0 // 边缘绝境
	case distance > 600:
		return ResourceLevel1 // 蛮荒带外层
	case distance > 500:
		return ResourceLevel2 // 蛮荒带内层
	case distance > 400:
		return ResourceLevel3 // 四大州外层
	case distance > 300:
		return ResourceLevel4 // 四大州内层
	case distance > 200:
		return ResourceLevel5 // 中心安全区
	default:
		return ResourceLevel6 // 皇城
	}
}

// GetTileType 获取地形类型
func (w *World) GetTileType(x, y int32, zone ZoneType) TileType {
	// 边缘绝境带
	if zone == ZoneEdge {
		seed := int64(x*10000 + y)
		r := rand.New(rand.NewSource(seed))
		if r.Float32() < 0.5 {
			return TileMountain // 高山
		}
		return TileDesert // 荒漠
	}

	// 皇城区域
	if zone == ZoneCastle {
		return TilePlain
	}

	// 使用固定种子生成地形
	seed := int64(x*10000 + y)
	r := rand.New(rand.NewSource(seed))
	randVal := r.Float32()

	// 地形比例
	switch {
	case randVal < 0.30: // 30% 平原
		return TilePlain
	case randVal < 0.55: // 25% 森林
		return TileForest
	case randVal < 0.70: // 15% 山地
		return TileMountain
	case randVal < 0.85: // 15% 丘陵
		return TileHill
	case randVal < 0.95: // 10% 河流
		return TileRiver
	case randVal < 0.975: // 2.5% 荒漠
		return TileDesert
	default: // 2.5% 雪山
		return TileSnow
	}
}

// GenerateResourceSpot 生成资源点
func (w *World) GenerateResourceSpot(x, y int32, level ResourceLevel) map[string]int32 {
	resources := make(map[string]int32)

	// 0 级无资源
	if level == ResourceLevel0 {
		return resources
	}

	// 基础资源量 = 等级 * 15
	baseAmount := int32(level) * 15

	// 随机生成 1-3 种资源
	rand.Seed(time.Now().UnixNano())
	resourceCount := rand.Intn(3) + 1 // 1-3 种资源

	resourceTypes := []ResourceType{ResourceGold, ResourceWood, ResourceFood, ResourceStone}
	rand.Shuffle(len(resourceTypes), func(i, j int) {
		resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i]
	})

	// 分配资源
	for i := 0; i < resourceCount; i++ {
		resType := resourceTypes[i]
		// 资源量 = 基础量 * (0.5-1.5 随机系数)
		multiplier := 0.5 + rand.Float32()
		amount := int32(float32(baseAmount) * multiplier)
		resources[string(resType)] = amount
	}

	return resources
}

// GetTileType 获取地形类型
func (w *World) createTile(x, y int32) *WorldTile {
	zone := w.GetZoneType(x, y)
	tileType := w.GetTileType(x, y, zone)
	resourceLvl := w.GetResourceLevel(x, y, zone)

	// 生成资源点（30% 概率有资源）
	rand.Seed(time.Now().UnixNano())
	hasResourceSpot := rand.Float32() < 0.3

	var resources map[string]int32
	if hasResourceSpot {
		resources = w.GenerateResourceSpot(x, y, resourceLvl)
	} else {
		resources = make(map[string]int32)
	}

	return &WorldTile{
		Coord:        WorldCoord{X: x, Y: y},
		TileType:     tileType,
		Zone:         zone,
		OwnerID:      0,
		Resources:    resources,
		ResourceLvl:  resourceLvl,
		Passable:     w.IsPassable(tileType, zone),
		CityType:     w.GetCityType(x, y, zone),
		ResourceSpot: hasResourceSpot,
	}
}

// IsPassable 是否可通行
func (w *World) IsPassable(tileType TileType, zone ZoneType) bool {
	if zone == ZoneEdge {
		return false // 边缘绝境不可通行
	}

	switch tileType {
	case TileRiver:
		return false // 河流不可通行（需要船）
	case TileMountain:
		return false // 高山不可通行
	default:
		return true
	}
}

// GetCityType 获取城市类型
func (w *World) GetCityType(x, y int32, zone ZoneType) string {
	if zone == ZoneCastle {
		return "castle" // 皇城
	}

	// 州府位置（每个州中心）
	mid := WorldSize / 2
	stateCapitals := map[ZoneType][2]int32{
		ZoneQing: {mid + StateSize/4, StateSize/4},
		ZoneJing: {mid + StateSize/4, mid + StateSize/4},
		ZoneYong: {StateSize/4, StateSize/4},
		ZoneYang: {StateSize/4, mid + StateSize/4},
	}

	if cap, ok := stateCapitals[zone]; ok {
		if x >= cap[0]-StateSize/4 && x < cap[0]+StateSize/4 &&
			y >= cap[1]-StateSize/4 && y < cap[1]+StateSize/4 {
			return "state" // 州府
		}
	}

	return "" // 普通地块
}

// GetTile 获取地块
func (w *World) GetTile(x, y int32) *WorldTile {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if x < 0 || x >= w.width || y < 0 || y >= w.height {
		return nil
	}

	coord := WorldCoord{X: x, Y: y}
	if tile, exists := w.tiles[coord]; exists {
		return tile
	}

	// 动态创建
	tile := w.createTile(x, y)
	w.tiles[coord] = tile
	return tile
}

// SetTile 设置地块
func (w *World) SetTile(tile *WorldTile) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.tiles[tile.Coord] = tile
	return nil
}

// GetTilesInArea 获取区域地块
func (w *World) GetTilesInArea(x1, y1, x2, y2 int32) []*WorldTile {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var tiles []*WorldTile
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
			if x < 0 || x >= w.width || y < 0 || y >= w.height {
				continue
			}
			coord := WorldCoord{X: x, Y: y}
			if tile, exists := w.tiles[coord]; exists {
				tiles = append(tiles, tile)
			} else {
				tiles = append(tiles, w.createTile(x, y))
			}
		}
	}
	return tiles
}

// ClaimTile 占领地块
func (w *World) ClaimTile(playerID uint64, x, y int32) error {
	tile := w.GetTile(x, y)
	if tile == nil {
		return nil
	}

	// 皇城和安全区不可占领
	if tile.Zone == ZoneCastle || tile.Zone == ZoneSafe {
		return nil
	}

	// 不可通行地块不可占领
	if !tile.Passable {
		return nil
	}

	if tile.OwnerID != 0 {
		return nil // 已被占领
	}

	tile.OwnerID = playerID
	return w.SetTile(tile)
}

// GenerateWorld 生成世界
func (w *World) GenerateWorld() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.generated {
		log.Println("[World] World already generated")
		return
	}

	log.Printf("[World] Generating world of size %dx%d", w.width, w.height)
	startTime := time.Now()

	// 统计
	stats := make(map[ZoneType]int)
	tileStats := make(map[TileType]int)
	resourceStats := make(map[ResourceLevel]int)
	spotCount := 0

	// 生成所有地块
	for x := int32(0); x < w.width; x++ {
		for y := int32(0); y < w.height; y++ {
			tile := w.createTile(x, y)
			w.tiles[tile.Coord] = tile

			stats[tile.Zone]++
			tileStats[tile.TileType]++
			resourceStats[tile.ResourceLvl]++
			if tile.ResourceSpot {
				spotCount++
			}
		}

		// 进度
		if (x+1)%(w.width/10) == 0 {
			log.Printf("[World] Generation: %d%%", ((x+1)*100)/w.width)
		}
	}

	elapsed := time.Since(startTime)
	w.generated = true

	log.Printf("[World] World generated: %d tiles in %v", len(w.tiles), elapsed)
	w.printStats(stats, tileStats, resourceStats, spotCount)
}

// printStats 打印统计
func (w *World) printStats(zoneStats map[ZoneType]int, tileStats map[TileType]int, resourceStats map[ResourceLevel]int, spotCount int) {
	total := len(w.tiles)

	log.Println("[World] === Zone Statistics ===")
	for zone, count := range zoneStats {
		pct := float64(count) / float64(total) * 100
		log.Printf("  %s: %d (%.2f%%)", zone, count, pct)
	}

	log.Println("[World] === Terrain Statistics ===")
	for tileType, count := range tileStats {
		pct := float64(count) / float64(total) * 100
		log.Printf("  %s: %d (%.2f%%)", tileType, count, pct)
	}

	log.Println("[World] === Resource Level Distribution (从外到内) ===")
	for lvl := ResourceLevel0; lvl <= ResourceLevel6; lvl++ {
		count := resourceStats[lvl]
		pct := float64(count) / float64(total) * 100
		log.Printf("  Level %d: %d (%.2f%%)", lvl, count, pct)
	}

	log.Printf("[World] Resource Spots: %d (%.2f%%)", spotCount, float64(spotCount)/float64(total)*100)

	log.Println("[World] === Resource Distribution ===")
	log.Println("  边缘绝境 (distance > 700):   0 级资源 - 无资源")
	log.Println("  蛮荒带外层 (600-700):        1 级资源 - 最低")
	log.Println("  蛮荒带内层 (500-600):        2 级资源 - 低")
	log.Println("  四大州外层 (400-500):        3 级资源 - 标准")
	log.Println("  四大州内层 (300-400):        4 级资源 - 中级")
	log.Println("  中心安全区 (200-300):        5 级资源 - 高级")
	log.Println("  皇城 (< 200):               6 级资源 - 顶级")
}

// GetWorldInfo 获取世界信息
func (w *World) GetWorldInfo() map[string]interface{} {
	return map[string]interface{}{
		"size":          WorldSize,
		"total_tiles":   WorldSize * WorldSize,
		"center_start":  CenterStart,
		"center_end":    CenterEnd,
		"castle_start":  CastleStart,
		"castle_end":    CastleEnd,
		"states":        []string{"qing", "jing", "yong", "yang"},
		"barbarian_width": BarbarianWidth,
		"edge_width":    EdgeWidth,
	}
}

// AddPlayer 添加玩家
func (w *World) AddPlayer(playerID uint64, playerData map[string]interface{}) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.players[playerID] = playerData
	log.Printf("[World] Player %d added", playerID)
}

// RemovePlayer 移除玩家
func (w *World) RemovePlayer(playerID uint64) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	delete(w.players, playerID)
	log.Printf("[World] Player %d removed", playerID)
}

// GetPlayer 获取玩家
func (w *World) GetPlayer(playerID uint64) map[string]interface{} {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.players[playerID]
}

// GetTick 获取 tick
func (w *World) GetTick() uint64 {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.currentTick
}

// abs32 int32 绝对值
func abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
