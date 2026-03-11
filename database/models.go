package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Player represents a game player with all their data
type Player struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	PlayerID     uint64             `bson:"player_id"`
	Username     string             `bson:"username"`
	PasswordHash string             `bson:"password_hash"`
	Email        string             `bson:"email"`
	CreatedAt    time.Time          `bson:"created_at"`
	LastLogin    time.Time          `bson:"last_login"`
	
	// Player stats
	Level        int32              `bson:"level"`
	Experience   int64              `bson:"experience"`
	Gold         int64              `bson:"gold"`
	Wood         int64              `bson:"wood"`
	Food         int64              `bson:"food"`
	Population   int32              `bson:"population"`
	MaxPopulation int32             `bson:"max_population"`
	
	// Coordinates
	X            int32              `bson:"x"`
	Y            int32              `bson:"y"`
	
	// Buildings
	Buildings    []Building         `bson:"buildings"`
	
	// Troops
	Troops       []Troop            `bson:"troops"`
	
	// Research
	Research     map[string]int32   `bson:"research"`
	
	// Alliance
	AllianceID   uint64             `bson:"alliance_id"`
	AllianceRole string             `bson:"alliance_role"`
	
	// VIP and premium features
	VIPLevel     int32              `bson:"vip_level"`
	VIPExpire    time.Time          `bson:"vip_expire"`
	
	// Settings
	Settings     PlayerSettings     `bson:"settings"`
}

type Building struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Type         string             `bson:"type"`
	Level        int32              `bson:"level"`
	X            int32              `bson:"x"`
	Y            int32              `bson:"y"`
	BuildTime    time.Time          `bson:"build_time"`
	FinishTime   time.Time          `bson:"finish_time"`
	IsCompleted  bool               `bson:"is_completed"`
}

type Troop struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Type         string             `bson:"type"`
	Count        int64              `bson:"count"`
	TrainingTime time.Time          `bson:"training_time"`
	FinishTime   time.Time          `bson:"finish_time"`
	IsCompleted  bool               `bson:"is_completed"`
}

type PlayerSettings struct {
	Language     string             `bson:"language"`
	Notifications bool              `bson:"notifications"`
	SoundEnabled bool              `bson:"sound_enabled"`
	MusicEnabled bool              `bson:"music_enabled"`
}

// Alliance represents a player alliance/guild
type Alliance struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	AllianceID   uint64             `bson:"alliance_id"`
	Name         string             `bson:"name"`
	Description  string             `bson:"description"`
	CreatorID    uint64             `bson:"creator_id"`
	CreatedAt    time.Time          `bson:"created_at"`
	MemberCount  int32              `bson:"member_count"`
	MaxMembers   int32              `bson:"max_members"`
	Level        int32              `bson:"level"`
	
	// Members
	Members      []AllianceMember   `bson:"members"`
}

type AllianceMember struct {
	PlayerID     uint64             `bson:"player_id"`
	Username     string             `bson:"username"`
	Role         string             `bson:"role"` // "leader", "officer", "member"
	JoinedAt     time.Time          `bson:"joined_at"`
}

// WorldMap represents the game world
type WorldMap struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	MapID        string             `bson:"map_id"`
	Width        int32              `bson:"width"`
	Height       int32              `bson:"height"`
	Tiles        [][]Tile           `bson:"tiles"`
}

type Tile struct {
	Type         string             `bson:"type"` // "plain", "forest", "mountain", "water", etc.
	OwnerID      uint64             `bson:"owner_id"`
	BuildingID   primitive.ObjectID `bson:"building_id"`
	ResourceType string             `bson:"resource_type"`
	ResourceAmount int32             `bson:"resource_amount"`
}

// BattleLog represents a battle record
type BattleLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	AttackerID   uint64             `bson:"attacker_id"`
	DefenderID   uint64             `bson:"defender_id"`
	BattleTime   time.Time          `bson:"battle_time"`
	Result       string             `bson:"result"` // "attacker_win", "defender_win", "draw"
	AttackerLoss int64              `bson:"attacker_loss"`
	DefenderLoss int64              `bson:"defender_loss"`
	LootGold     int64              `bson:"loot_gold"`
	LootWood     int64              `bson:"loot_wood"`
	LootFood     int64              `bson:"loot_food"`
}