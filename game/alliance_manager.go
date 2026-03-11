package game

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"slg-game/database"
)

// AllianceManager 联盟管理器
type AllianceManager struct {
	db *database.Database
}

// NewAllianceManager 创建联盟管理器
func NewAllianceManager(db *database.Database) *AllianceManager {
	return &AllianceManager{db: db}
}

// AllianceRole 联盟角色
type AllianceRole string

const (
	RoleLeader   AllianceRole = "leader"
	RoleOfficer  AllianceRole = "officer"
	RoleMember   AllianceRole = "member"
)

var (
	ErrAllianceFull      = errors.New("alliance is full")
	ErrAlreadyInAlliance = errors.New("player already in alliance")
	ErrNotInAlliance     = errors.New("player not in alliance")
	ErrNoPermission      = errors.New("no permission")
)

// CreateAlliance 创建联盟
func (am *AllianceManager) CreateAlliance(leaderID uint64, name, description string) (*database.Alliance, error) {
	// 检查玩家是否已在联盟中
	inAlliance, err := am.IsPlayerInAlliance(leaderID)
	if err != nil {
		return nil, err
	}
	if inAlliance {
		return nil, ErrAlreadyInAlliance
	}

	// 生成联盟 ID
	collection := am.db.GetCollection("alliances")
	var lastAlliance database.Alliance
	err = collection.FindOne(context.Background(), bson.M{}, options.FindOne().SetSort(bson.M{"alliance_id": -1})).Decode(&lastAlliance)
	newAllianceID := uint64(1001)
	if err == nil {
		newAllianceID = lastAlliance.AllianceID + 1
	}

	// 创建联盟
	alliance := &database.Alliance{
		AllianceID:  newAllianceID,
		Name:        name,
		Description: description,
		CreatorID:   leaderID,
		CreatedAt:   time.Now(),
		MemberCount: 1,
		MaxMembers:  50,
		Level:       1,
		Members: []database.AllianceMember{
			{
				PlayerID: leaderID,
				Role:     string(RoleLeader),
				JoinedAt: time.Now(),
			},
		},
	}

	_, err = collection.InsertOne(context.Background(), alliance)
	if err != nil {
		return nil, err
	}

	// 更新玩家联盟信息
	am.updatePlayerAlliance(leaderID, newAllianceID, string(RoleLeader))

	log.Printf("Alliance created: %s (ID: %d) by player %d", name, newAllianceID, leaderID)
	return alliance, nil
}

// JoinAlliance 加入联盟
func (am *AllianceManager) JoinAlliance(playerID uint64, allianceID uint64) error {
	// 检查玩家是否已在联盟中
	inAlliance, err := am.IsPlayerInAlliance(playerID)
	if err != nil {
		return err
	}
	if inAlliance {
		return ErrAlreadyInAlliance
	}

	// 获取联盟信息
	alliance, err := am.GetAlliance(allianceID)
	if err != nil {
		return err
	}

	// 检查是否满员
	if alliance.MemberCount >= alliance.MaxMembers {
		return ErrAllianceFull
	}

	// 添加成员
	collection := am.db.GetCollection("alliances")
	member := database.AllianceMember{
		PlayerID: playerID,
		Role:     string(RoleMember),
		JoinedAt: time.Now(),
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"alliance_id": allianceID},
		bson.M{
			"$push": bson.M{"members": member},
			"$inc":  bson.M{"member_count": 1},
		},
	)
	if err != nil {
		return err
	}

	// 更新玩家联盟信息
	am.updatePlayerAlliance(playerID, allianceID, string(RoleMember))

	log.Printf("Player %d joined alliance %d", playerID, allianceID)
	return nil
}

// LeaveAlliance 离开联盟
func (am *AllianceManager) LeaveAlliance(playerID uint64) error {
	allianceInfo, err := am.GetPlayerAllianceInfo(playerID)
	if err != nil {
		return err
	}
	if allianceInfo == nil {
		return ErrNotInAlliance
	}

	// 如果是盟主，需要转让或解散
	if allianceInfo.Role == string(RoleLeader) {
		// 简单处理：直接解散联盟
		return am.DisbandAlliance(allianceInfo.AllianceID)
	}

	// 移除成员
	collection := am.db.GetCollection("alliances")
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"alliance_id": allianceInfo.AllianceID},
		bson.M{
			"$pull": bson.M{"members": bson.M{"player_id": playerID}},
			"$inc":  bson.M{"member_count": -1},
		},
	)
	if err != nil {
		return err
	}

	// 更新玩家联盟信息
	am.updatePlayerAlliance(playerID, 0, "")

	log.Printf("Player %d left alliance %d", playerID, allianceInfo.AllianceID)
	return nil
}

// DisbandAlliance 解散联盟
func (am *AllianceManager) DisbandAlliance(allianceID uint64) error {
	// 获取所有成员
	alliance, err := am.GetAlliance(allianceID)
	if err != nil {
		return err
	}

	// 清除所有成员的联盟信息
	for _, member := range alliance.Members {
		am.updatePlayerAlliance(member.PlayerID, 0, "")
	}

	// 删除联盟
	collection := am.db.GetCollection("alliances")
	_, err = collection.DeleteOne(context.Background(), bson.M{"alliance_id": allianceID})
	if err != nil {
		return err
	}

	log.Printf("Alliance %d disbanded", allianceID)
	return nil
}

// GetAlliance 获取联盟信息
func (am *AllianceManager) GetAlliance(allianceID uint64) (*database.Alliance, error) {
	collection := am.db.GetCollection("alliances")
	var alliance database.Alliance
	err := collection.FindOne(context.Background(), bson.M{"alliance_id": allianceID}).Decode(&alliance)
	if err != nil {
		return nil, err
	}
	return &alliance, nil
}

// PlayerAllianceInfo 玩家联盟信息
type PlayerAllianceInfo struct {
	AllianceID uint64 `bson:"alliance_id" json:"alliance_id"`
	Role       string `bson:"role" json:"role"`
}

// GetPlayerAllianceInfo 获取玩家联盟信息
func (am *AllianceManager) GetPlayerAllianceInfo(playerID uint64) (*PlayerAllianceInfo, error) {
	collection := am.db.GetCollection("players")
	var player database.Player
	err := collection.FindOne(context.Background(), bson.M{"player_id": playerID}).Decode(&player)
	if err != nil {
		return nil, err
	}

	if player.AllianceID == 0 {
		return nil, nil
	}

	return &PlayerAllianceInfo{
		AllianceID: player.AllianceID,
		Role:       player.AllianceRole,
	}, nil
}

// IsPlayerInAlliance 检查玩家是否在联盟中
func (am *AllianceManager) IsPlayerInAlliance(playerID uint64) (bool, error) {
	info, err := am.GetPlayerAllianceInfo(playerID)
	if err != nil {
		return false, err
	}
	return info != nil, nil
}

// updatePlayerAlliance 更新玩家联盟信息
func (am *AllianceManager) updatePlayerAlliance(playerID uint64, allianceID uint64, role string) error {
	collection := am.db.GetCollection("players")
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"player_id": playerID},
		bson.M{"$set": bson.M{
			"alliance_id":   allianceID,
			"alliance_role": role,
		}},
	)
	return err
}

// GetAllianceMembers 获取联盟所有成员
func (am *AllianceManager) GetAllianceMembers(allianceID uint64) ([]database.AllianceMember, error) {
	alliance, err := am.GetAlliance(allianceID)
	if err != nil {
		return nil, err
	}
	return alliance.Members, nil
}

// SetMemberRole 设置成员角色（需要官员权限）
func (am *AllianceManager) SetMemberRole(operatorID uint64, targetID uint64, role AllianceRole) error {
	// 检查操作者权限
	operatorInfo, _ := am.GetPlayerAllianceInfo(operatorID)
	if operatorInfo == nil || (operatorInfo.Role != string(RoleLeader) && operatorInfo.Role != string(RoleOfficer)) {
		return ErrNoPermission
	}

	// 获取联盟信息
	allianceInfo, _ := am.GetPlayerAllianceInfo(operatorID)
	if allianceInfo == nil {
		return ErrNotInAlliance
	}

	// 更新成员角色
	collection := am.db.GetCollection("alliances")
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{
			"alliance_id": allianceInfo.AllianceID,
			"members.player_id": targetID,
		},
		bson.M{
			"$set": bson.M{"members.$.role": string(role)},
		},
	)
	if err != nil {
		return err
	}

	// 更新玩家信息
	am.updatePlayerAlliance(targetID, allianceInfo.AllianceID, string(role))

	log.Printf("Player %d role changed to %s in alliance %d", targetID, role, allianceInfo.AllianceID)
	return nil
}
