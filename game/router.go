package game

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"slg-game/database"
	"slg-game/errors"
	"slg-game/log"
	"slg-game/network"
	"slg-game/proto"
)

const (
	MsgID_C2S_LoginRequest     = 1001
	MsgID_C2S_RegisterRequest  = 1002
	MsgID_C2S_MoveRequest      = 1003
	MsgID_C2S_BuildRequest     = 1004
	MsgID_S2C_LoginResponse    = 2001
	MsgID_S2C_RegisterResponse = 2002
	MsgID_S2C_MoveResponse     = 2003
	MsgID_S2C_BuildResponse    = 2004
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
		log.Warnf("Unknown message ID: %d", packet.MsgID)
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
	if err := network.UnmarshalJSON(data, request); err != nil {
		log.Errorf("Failed to unmarshal login request: %v", err)
		return createLoginErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.Username == "" || request.Password == "" {
		return createLoginErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Username and password required"))
	}

	// 查询数据库
	collection := mr.db.GetCollection("players")
	var player database.Player
	err := collection.FindOne(context.Background(), bson.M{"username": request.Username}).Decode(&player)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"username": request.Username,
		}).Warn("Login failed - user not found")
		return createLoginErrorResponse(errors.ErrUserNotFoundErr)
	}

	// 验证密码
	hashedPassword := hashPassword(request.Password)
	if player.PasswordHash != hashedPassword {
		log.WithFields(map[string]interface{}{
			"username": request.Username,
		}).Warn("Login failed - wrong password")
		return createLoginErrorResponse(errors.ErrWrongPasswordErr)
	}

	// 更新最后登录时间
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"player_id": player.PlayerID},
		bson.M{"$set": bson.M{"last_login": time.Now()}},
	)
	if err != nil {
		log.Errorf("Failed to update last login: %v", err)
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
			BuildingId:   player.PlayerID,
			BuildingType: b.Type,
			Level:        b.Level,
			X:            b.X,
			Y:            b.Y,
		})
	}

	response := &proto.S2C_LoginResponse{
		Success:    true,
		Message:    "Login successful",
		PlayerId:   player.PlayerID,
		PlayerData: playerData,
	}

	responseData, err := network.MarshalJSON(response)
	if err != nil {
		log.Errorf("Failed to marshal login response: %v", err)
		return createLoginErrorResponse(errors.ErrInternalErr)
	}

	log.WithFields(map[string]interface{}{
		"player_id": player.PlayerID,
		"username":  player.Username,
	}).Info("Player logged in")

	return &network.Packet{
		MsgID: MsgID_S2C_LoginResponse,
		Data:  responseData,
	}
}

// handleRegisterRequest 处理注册请求
func (mr *MessageRouter) handleRegisterRequest(session *PlayerSession, data []byte) *network.Packet {
	request := &proto.C2S_RegisterRequest{}
	if err := network.UnmarshalJSON(data, request); err != nil {
		log.Errorf("Failed to unmarshal register request: %v", err)
		return createRegisterErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.Username == "" || request.Password == "" {
		return createRegisterErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Username and password required"))
	}

	// 检查用户名是否已存在
	collection := mr.db.GetCollection("players")
	count, err := collection.CountDocuments(context.Background(), bson.M{"username": request.Username})
	if err != nil {
		log.Errorf("Failed to check username: %v", err)
		return createRegisterErrorResponse(errors.ErrDatabaseErrorErr)
	}
	if count > 0 {
		log.WithFields(map[string]interface{}{
			"username": request.Username,
		}).Warn("Register failed - username exists")
		return createRegisterErrorResponse(errors.ErrUserExistsErr)
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
			Language:      "zh-CN",
			Notifications: true,
			SoundEnabled:  true,
			MusicEnabled:  true,
		},
	}

	_, err = collection.InsertOne(context.Background(), newPlayer)
	if err != nil {
		log.Errorf("Failed to create player: %v", err)
		return createRegisterErrorResponse(errors.ErrDatabaseErrorErr)
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

	responseData, err := network.MarshalJSON(response)
	if err != nil {
		log.Errorf("Failed to marshal register response: %v", err)
		return createRegisterErrorResponse(errors.ErrInternalErr)
	}

	log.WithFields(map[string]interface{}{
		"player_id": newPlayer.PlayerID,
		"username":  newPlayer.Username,
	}).Info("New player registered")

	return &network.Packet{
		MsgID: MsgID_S2C_RegisterResponse,
		Data:  responseData,
	}
}

// handleMoveRequest 处理移动请求
func (mr *MessageRouter) handleMoveRequest(session *PlayerSession, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createMoveErrorResponse(errors.ErrNotLoggedInErr)
	}

	request := &proto.C2S_MoveRequest{}
	if err := network.UnmarshalJSON(data, request); err != nil {
		log.Errorf("Failed to unmarshal move request: %v", err)
		return createMoveErrorResponse(errors.ErrInvalidRequestErr)
	}

	// 验证坐标范围
	if request.X < -10000 || request.X > 10000 || request.Y < -10000 || request.Y > 10000 {
		return createMoveErrorResponse(errors.ErrInvalidPositionErr)
	}

	playerID := session.GetPlayerID()

	// 更新玩家位置
	collection := mr.db.GetCollection("players")
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"player_id": playerID},
		bson.M{"$set": bson.M{"x": request.X, "y": request.Y}},
	)

	if err != nil {
		log.WithFields(map[string]interface{}{
			"player_id": playerID,
		}).Errorf("Failed to update player position: %v", err)
		return createMoveErrorResponse(errors.ErrDatabaseErrorErr)
	}

	response := &proto.S2C_MoveResponse{
		Success: true,
		Message: "Move successful",
		X:       request.X,
		Y:       request.Y,
	}

	responseData, err := network.MarshalJSON(response)
	if err != nil {
		log.Errorf("Failed to marshal move response: %v", err)
		return createMoveErrorResponse(errors.ErrInternalErr)
	}

	return &network.Packet{
		MsgID: MsgID_S2C_MoveResponse,
		Data:  responseData,
	}
}

// handleBuildRequest 处理建造请求
func (mr *MessageRouter) handleBuildRequest(session *PlayerSession, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createBuildErrorResponse(errors.ErrNotLoggedInErr)
	}

	request := &proto.C2S_BuildRequest{}
	if err := network.UnmarshalJSON(data, request); err != nil {
		log.Errorf("Failed to unmarshal build request: %v", err)
		return createBuildErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.BuildingType == "" {
		return createBuildErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Building type required"))
	}

	playerID := session.GetPlayerID()

	// TODO: 检查资源是否足够
	// TODO: 扣除资源
	// TODO: 创建建筑

	response := &proto.S2C_BuildResponse{
		Success: true,
		Message: "Build request received",
		Building: &proto.Building{
			BuildingType: request.BuildingType,
			X:            request.X,
			Y:            request.Y,
			Level:        1,
		},
	}

	responseData, err := network.MarshalJSON(response)
	if err != nil {
		log.Errorf("Failed to marshal build response: %v", err)
		return createBuildErrorResponse(errors.ErrInternalErr)
	}

	log.WithFields(map[string]interface{}{
		"player_id":     playerID,
		"building_type": request.BuildingType,
		"x":             request.X,
		"y":             request.Y,
	}).Info("Build request received")

	return &network.Packet{
		MsgID: MsgID_S2C_BuildResponse,
		Data:  responseData,
	}
}

// 错误响应辅助函数
func createLoginErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &proto.S2C_LoginResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := network.MarshalJSON(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_LoginResponse,
			Data:  data,
		}
	}
	return nil
}

func createRegisterErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &proto.S2C_RegisterResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := network.MarshalJSON(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_RegisterResponse,
			Data:  data,
		}
	}
	return nil
}

func createMoveErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &proto.S2C_MoveResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := network.MarshalJSON(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_MoveResponse,
			Data:  data,
		}
	}
	return nil
}

func createBuildErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &proto.S2C_BuildResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := network.MarshalJSON(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_BuildResponse,
			Data:  data,
		}
	}
	return nil
}
