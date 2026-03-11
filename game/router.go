package game

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"slg-game/database"
	"slg-game/network"
	"slg-game/proto"
)

const (
	MsgID_C2S_LoginRequest   = 1001
	MsgID_C2S_RegisterRequest = 1002
	MsgID_C2S_MoveRequest    = 1002
	MsgID_C2S_BuildRequest   = 1003
	MsgID_S2C_LoginResponse  = 2001
	MsgID_S2C_RegisterResponse = 2002
	MsgID_S2C_MoveResponse   = 2002
	MsgID_S2C_BuildResponse  = 2003
	MsgID_S2C_PlayerUpdate   = 2004
)

type MessageRouter struct {
	handlers map[uint32]func(*PlayerSession, []byte) *network.Packet
	db       *database.Database
}

func NewMessageRouter(db *database.Database) *MessageRouter {
	router := &MessageRouter{
		handlers: make(map[uint32]func(*PlayerSession, []byte) *network.Packet),
		db:       db,
	}
	router.registerHandlers()
	return router
}

func (mr *MessageRouter) registerHandlers() {
	mr.handlers[MsgID_C2S_LoginRequest] = mr.handleLoginRequest
	mr.handlers[MsgID_C2S_RegisterRequest] = mr.handleRegisterRequest
	mr.handlers[MsgID_C2S_MoveRequest] = mr.handleMoveRequest
	mr.handlers[MsgID_C2S_BuildRequest] = mr.handleBuildRequest
}

func (mr *MessageRouter) Route(session *PlayerSession, packet *network.Packet) *network.Packet {
	handler, exists := mr.handlers[packet.MsgID]
	if !exists {
		log.Printf("Unknown message ID: %d", packet.MsgID)
		return nil
	}

	return handler(session, packet.Data)
}

// hashPassword 对密码进行 SHA256 哈希
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// handleLoginRequest 处理登录请求
func (mr *MessageRouter) handleLoginRequest(session *PlayerSession, data []byte) *network.Packet {
	request := &proto.C2S_LoginRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		log.Printf("Failed to unmarshal login request: %v", err)
		return createLoginErrorResponse("Invalid login request")
	}

	// 查询数据库
	collection := mr.db.GetCollection("players")
	var player database.Player
	err := collection.FindOne(context.Background(), bson.M{"username": request.Username}).Decode(&player)
	if err != nil {
		log.Printf("Login failed - user not found: %s", request.Username)
		return createLoginErrorResponse("Invalid username or password")
	}

	// 验证密码
	hashedPassword := hashPassword(request.Password)
	if player.PasswordHash != hashedPassword {
		log.Printf("Login failed - wrong password for user: %s", request.Username)
		return createLoginErrorResponse("Invalid username or password")
	}

	// 更新最后登录时间
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"player_id": player.PlayerID},
		bson.M{"$set": bson.M{"last_login": time.Now()}},
	)
	if err != nil {
		log.Printf("Failed to update last login: %v", err)
	}

	// 设置会话状态
	session.SetPlayerID(player.PlayerID)
	session.SetUsername(player.Username)
	session.SetLoggedIn(true)

	// 构建玩家数据响应
	playerData := &proto.PlayerData{
		PlayerId:   player.PlayerID,
		Username:   player.Username,
		Email:      player.Email,
		Level:      player.Level,
		Experience: player.Experience,
		X:          player.X,
		Y:          player.Y,
		Resources: map[string]int64{
			"gold":  player.Gold,
			"wood":  player.Wood,
			"food":  player.Food,
		},
		CreatedAt: player.CreatedAt.UnixMilli(),
		LastLogin: player.LastLogin.UnixMilli(),
	}

	// 转换建筑数据
	for _, b := range player.Buildings {
		playerData.Buildings = append(playerData.Buildings, &proto.Building{
			BuildingId: uint64(b.ID.Hex()[0:8]),
			BuildingType: b.Type,
			Level:      b.Level,
			X:          b.X,
			Y:          b.Y,
		})
	}

	response := &proto.S2C_LoginResponse{
		Success:     true,
		Message:     "Login successful",
		PlayerId:    player.PlayerID,
		PlayerData:  playerData,
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal login response: %v", err)
		return createLoginErrorResponse("Internal error")
	}

	log.Printf("Player logged in: %s (ID: %d)", player.Username, player.PlayerID)

	return &network.Packet{
		MsgID: MsgID_S2C_LoginResponse,
		Data:  responseData,
	}
}

// handleRegisterRequest 处理注册请求
func (mr *MessageRouter) handleRegisterRequest(session *PlayerSession, data []byte) *network.Packet {
	request := &proto.C2S_RegisterRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		log.Printf("Failed to unmarshal register request: %v", err)
		return createRegisterErrorResponse("Invalid register request")
	}

	// 检查用户名是否已存在
	collection := mr.db.GetCollection("players")
	count, err := collection.CountDocuments(context.Background(), bson.M{"username": request.Username})
	if err != nil {
		log.Printf("Failed to check username: %v", err)
		return createRegisterErrorResponse("Internal error")
	}
	if count > 0 {
		return createRegisterErrorResponse("Username already exists")
	}

	// 生成新的玩家 ID
	var lastPlayer database.Player
	err = collection.FindOne(context.Background(), bson.M{}, options.FindOne().SetSort(bson.M{"player_id": -1})).Decode(&lastPlayer)
	newPlayerID := uint64(10001)
	if err == nil {
		newPlayerID = lastPlayer.PlayerID + 1
	}

	// 创建新玩家
	hashedPassword := hashPassword(request.Password)
	newPlayer := &database.Player{
		PlayerID:     newPlayerID,
		Username:     request.Username,
		PasswordHash: hashedPassword,
		Email:        request.Email,
		CreatedAt:    time.Now(),
		LastLogin:    time.Now(),
		Level:        1,
		Experience:   0,
		Gold:         1000,
		Wood:         1000,
		Food:         1000,
		Population:   0,
		MaxPopulation: 100,
		X:            0,
		Y:            0,
		Buildings:    []database.Building{},
		Troops:       []database.Troop{},
		Research:     make(map[string]int32),
		Settings: database.PlayerSettings{
			Language:     "zh-CN",
			Notifications: true,
			SoundEnabled: true,
			MusicEnabled: true,
		},
	}

	_, err = collection.InsertOne(context.Background(), newPlayer)
	if err != nil {
		log.Printf("Failed to create player: %v", err)
		return createRegisterErrorResponse("Failed to create account")
	}

	// 设置会话状态
	session.SetPlayerID(newPlayer.PlayerID)
	session.SetUsername(newPlayer.Username)
	session.SetLoggedIn(true)

	response := &proto.S2C_RegisterResponse{
		Success:  true,
		Message:  "Registration successful",
		PlayerId: newPlayer.PlayerID,
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal register response: %v", err)
		return createRegisterErrorResponse("Internal error")
	}

	log.Printf("New player registered: %s (ID: %d)", newPlayer.Username, newPlayer.PlayerID)

	return &network.Packet{
		MsgID: MsgID_S2C_RegisterResponse,
		Data:  responseData,
	}
}

// handleMoveRequest 处理移动请求
func (mr *MessageRouter) handleMoveRequest(session *PlayerSession, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createMoveErrorResponse("Not logged in")
	}

	request := &proto.C2S_MoveRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		log.Printf("Failed to unmarshal move request: %v", err)
		return createMoveErrorResponse("Invalid move request")
	}

	// TODO: 验证移动坐标是否在地图范围内
	// TODO: 更新玩家位置
	// TODO: 保存到数据库

	response := &proto.S2C_MoveResponse{
		Success: true,
		Message: "Move successful",
		X:       request.X,
		Y:       request.Y,
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal move response: %v", err)
		return createMoveErrorResponse("Internal error")
	}

	return &network.Packet{
		MsgID: MsgID_S2C_MoveResponse,
		Data:  responseData,
	}
}

// handleBuildRequest 处理建造请求
func (mr *MessageRouter) handleBuildRequest(session *PlayerSession, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createBuildErrorResponse("Not logged in")
	}

	request := &proto.C2S_BuildRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		log.Printf("Failed to unmarshal build request: %v", err)
		return createBuildErrorResponse("Invalid build request")
	}

	// TODO: 验证建筑类型和位置
	// TODO: 检查资源是否足够
	// TODO: 扣除资源
	// TODO: 创建建筑
	// TODO: 保存到数据库

	response := &proto.S2C_BuildResponse{
		Success: true,
		Message: "Build successful",
		Building: &proto.Building{
			BuildingType: request.BuildingType,
			X:            request.X,
			Y:            request.Y,
			Level:        1,
		},
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal build response: %v", err)
		return createBuildErrorResponse("Internal error")
	}

	return &network.Packet{
		MsgID: MsgID_S2C_BuildResponse,
		Data:  responseData,
	}
}

// 错误响应辅助函数
func createLoginErrorResponse(message string) *network.Packet {
	response := &proto.S2C_LoginResponse{
		Success: false,
		Message: message,
	}
	if data, err := proto.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_LoginResponse,
			Data:  data,
		}
	}
	return nil
}

func createRegisterErrorResponse(message string) *network.Packet {
	response := &proto.S2C_RegisterResponse{
		Success: false,
		Message: message,
	}
	if data, err := proto.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_RegisterResponse,
			Data:  data,
		}
	}
	return nil
}

func createMoveErrorResponse(message string) *network.Packet {
	response := &proto.S2C_MoveResponse{
		Success: false,
		Message: message,
	}
	if data, err := proto.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_MoveResponse,
			Data:  data,
		}
	}
	return nil
}

func createBuildErrorResponse(message string) *network.Packet {
	response := &proto.S2C_BuildResponse{
		Success: false,
		Message: message,
	}
	if data, err := proto.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_BuildResponse,
			Data:  data,
		}
	}
	return nil
}
