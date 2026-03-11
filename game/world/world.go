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
	db          database.DB
	tiles       map[WorldCoord]*WorldTile
	players     map[uint64]map[string]interface{}
	mutex       sync.RWMutex
	tickChannel chan struct{}
	stopChannel chan struct{}
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
		db:          db,
		tiles:       make(map[WorldCoord]*WorldTile),
		players:     make(map[uint64]map[string]interface{}),
		tickChannel: make(chan struct{}, 1),
		stopChannel: make(chan struct{}),
	}

	log.Println("World initialized (memory mode)")

	return world
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
		return nil
	}

	tile.OwnerID = playerID
	return w.SetTile(tile)
}

// StartGameLoop starts the world game loop
func (w *World) StartGameLoop(tickInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.tick()
			case <-w.stopChannel:
				return
			}
		}
	}()
}

// StopGameLoop stops the world game loop
func (w *World) StopGameLoop() {
	close(w.stopChannel)
}

// tick processes one game tick for the world
func (w *World) tick() {
	w.processResourceGeneration()
}

// processResourceGeneration processes resource generation
func (w *World) processResourceGeneration() {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for _, tile := range w.tiles {
		if tile.BuildingID != "" && tile.OwnerID != 0 {
			if player, exists := w.players[tile.OwnerID]; exists {
				for resType, amount := range map[string]int64{"gold": 10} {
					if current, ok := player[resType].(int64); ok {
						player[resType] = current + amount
					}
				}
			}
		}
	}
}

// AddPlayer adds a player to the world
func (w *World) AddPlayer(playerID uint64, playerData map[string]interface{}) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.players[playerID] = playerData
}

// RemovePlayer removes a player from the world
func (w *World) RemovePlayer(playerID uint64) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	delete(w.players, playerID)
}

// GetPlayer gets a player by ID
func (w *World) GetPlayer(playerID uint64) map[string]interface{} {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return w.players[playerID]
}

// Tick 世界 tick 更新
func (w *World) Tick() {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
}

// GenerateWorld generates a new world with specified dimensions
func (w *World) GenerateWorld(width, height int32) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

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

	log.Printf("Generated world of size %dx%d", width, height)
}
