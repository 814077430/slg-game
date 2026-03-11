package game

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"slg-game/database"
)

// World represents the game world with all tiles and players
type World struct {
	db          *database.Database
	tiles       map[WorldCoord]*WorldTile
	players     map[uint64]*PlayerData
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
	Coord      WorldCoord `bson:"coord"`
	TileType   string     `bson:"tile_type"`   // "grass", "water", "mountain", etc.
	OwnerID    uint64     `bson:"owner_id"`    // 0 means unowned
	BuildingID string     `bson:"building_id"` // empty means no building
	Resource   Resource   `bson:"resource"`    // natural resources on this tile
}

// NewWorld creates a new world instance
func NewWorld(db *database.Database) *World {
	world := &World{
		db:          db,
		tiles:       make(map[WorldCoord]*WorldTile),
		players:     make(map[uint64]*PlayerData),
		tickChannel: make(chan struct{}, 1),
		stopChannel: make(chan struct{}),
	}
	
	// Load world data from database
	world.loadWorldFromDB()
	
	return world
}

// loadWorldFromDB loads the world state from the database
func (w *World) loadWorldFromDB() {
	collection := w.db.GetCollection("world_tiles")
	
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Failed to load world tiles: %v", err)
		return
	}
	defer cursor.Close(context.Background())
	
	var tiles []WorldTile
	if err = cursor.All(context.Background(), &tiles); err != nil {
		log.Printf("Failed to decode world tiles: %v", err)
		return
	}
	
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	for _, tile := range tiles {
		w.tiles[tile.Coord] = &tile
	}
	
	log.Printf("Loaded %d world tiles from database", len(tiles))
}

// saveTile saves a single tile to the database
func (w *World) saveTile(tile *WorldTile) error {
	collection := w.db.GetCollection("world_tiles")
	
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"coord.x": tile.Coord.X, "coord.y": tile.Coord.Y},
		bson.M{"$set": tile},
		options.Update().SetUpsert(true),
	)
	
	return err
}

// GetTile gets a tile at the specified coordinates
func (w *World) GetTile(x, y int32) *WorldTile {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	coord := WorldCoord{X: x, Y: y}
	if tile, exists := w.tiles[coord]; exists {
		return tile
	}
	
	// Create default tile if not exists
	defaultTile := &WorldTile{
		Coord:    coord,
		TileType: "grass",
		OwnerID:  0,
		Resource: Resource{Gold: 0, Wood: 0, Food: 0, Stone: 0},
	}
	
	// Save to database and cache
	w.tiles[coord] = defaultTile
	go w.saveTile(defaultTile)
	
	return defaultTile
}

// SetTile sets a tile at the specified coordinates
func (w *World) SetTile(tile *WorldTile) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	w.tiles[tile.Coord] = tile
	return w.saveTile(tile)
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
				// Create default tile
				defaultTile := &WorldTile{
					Coord:    coord,
					TileType: "grass",
					OwnerID:  0,
					Resource: Resource{Gold: 0, Wood: 0, Food: 0, Stone: 0},
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
	
	// Check if tile is already claimed
	if tile.OwnerID != 0 {
		return fmt.Errorf("tile already claimed by player %d", tile.OwnerID)
	}
	
	// Check if tile is claimable (not water or mountain)
	if tile.TileType == "water" || tile.TileType == "mountain" {
		return fmt.Errorf("cannot claim %s tile", tile.TileType)
	}
	
	// Claim the tile
	tile.OwnerID = playerID
	
	return w.SetTile(tile)
}

// BuildOnTile builds a building on a tile
func (w *World) BuildOnTile(playerID uint64, buildingID string, x, y int32) error {
	tile := w.GetTile(x, y)
	
	// Check if player owns the tile
	if tile.OwnerID != playerID {
		return fmt.Errorf("player %d does not own tile at (%d, %d)", playerID, x, y)
	}
	
	// Check if there's already a building
	if tile.BuildingID != "" {
		return fmt.Errorf("tile already has a building: %s", tile.BuildingID)
	}
	
	// Build the building
	tile.BuildingID = buildingID
	
	return w.SetTile(tile)
}

// GetPlayerTiles gets all tiles owned by a player
func (w *World) GetPlayerTiles(playerID uint64) []*WorldTile {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	var tiles []*WorldTile
	for _, tile := range w.tiles {
		if tile.OwnerID == playerID {
			tiles = append(tiles, tile)
		}
	}
	
	return tiles
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
	// Process resource generation from buildings
	w.processResourceGeneration()
	
	// Process building construction completion
	w.processBuildingCompletion()
	
	// Process army movement and actions
	w.processArmyActions()
	
	// Process technology research completion
	w.processTechnologyCompletion()
}

// processResourceGeneration processes resource generation from all buildings
func (w *World) processResourceGeneration() {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	for _, tile := range w.tiles {
		if tile.BuildingID != "" && tile.OwnerID != 0 {
			// Get building info and generate resources
			buildingInfo, exists := BuildingTemplates[tile.BuildingID]
			if exists && buildingInfo.ResourceProduction != nil {
				// Update player resources
				if player, exists := w.players[tile.OwnerID]; exists {
					player.Resources.Add(*buildingInfo.ResourceProduction)
					// Save player data
					go w.savePlayerData(player)
				}
			}
		}
	}
}

// processBuildingCompletion processes building construction completion
func (w *World) processBuildingCompletion() {
	// This would check construction queues and complete buildings
	// Implementation depends on how construction is tracked
}

// processArmyActions processes army movement and combat
func (w *World) processArmyActions() {
	// This would handle army movement, combat resolution, etc.
}

// processTechnologyCompletion processes technology research completion
func (w *World) processTechnologyCompletion() {
	// This would check research queues and complete technologies
}

// AddPlayer adds a player to the world
func (w *World) AddPlayer(player *PlayerData) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	w.players[player.PlayerID] = player
}

// RemovePlayer removes a player from the world
func (w *World) RemovePlayer(playerID uint64) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	delete(w.players, playerID)
}

// GetPlayer gets a player by ID
func (w *World) GetPlayer(playerID uint64) *PlayerData {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	return w.players[playerID]
}

// savePlayerData saves player data to database
func (w *World) savePlayerData(player *PlayerData) error {
	collection := w.db.GetCollection("players")
	
	_, err := collection.ReplaceOne(
		context.Background(),
		bson.M{"player_id": player.PlayerID},
		player,
		options.Replace().SetUpsert(true),
	)
	
	return err
}

// GenerateWorld generates a new world with specified dimensions
func (w *World) GenerateWorld(width, height int32) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	for x := int32(0); x < width; x++ {
		for y := int32(0); y < height; y++ {
			// Simple terrain generation
			tileType := "grass"
			if x%10 == 0 && y%10 == 0 {
				tileType = "water"
			} else if x%7 == 0 || y%7 == 0 {
				tileType = "mountain"
			}
			
			// Add some natural resources
			resource := Resource{
				Gold:  rand.Int31n(10),
				Wood:  rand.Int31n(15),
				Food:  rand.Int31n(20),
				Stone: rand.Int31n(8),
			}
			
			tile := &WorldTile{
				Coord:    WorldCoord{X: x, Y: y},
				TileType: tileType,
				OwnerID:  0,
				Resource: resource,
			}
			
			w.tiles[tile.Coord] = tile
			go w.saveTile(tile)
		}
	}
	
	log.Printf("Generated world of size %dx%d", width, height)
}