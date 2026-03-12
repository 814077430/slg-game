package core

import (
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"
	"time"

	"slg-game/database"
	"slg-game/errors"
	"slg-game/log"
	"slg-game/network"
	pb "slg-game/protocol"
	"slg-game/protocol"
	"slg-game/session"
)

// 核心协议消息 ID
const (
	MsgID_C2S_LoginRequest     = 1001
	MsgID_C2S_RegisterRequest  = 1002
	MsgID_C2S_MoveRequest      = 1003
	MsgID_C2S_BuildRequest     = 1004
	MsgID_C2S_WhoRequest       = 1005
	MsgID_S2C_LoginResponse    = 2001
	MsgID_S2C_RegisterResponse = 2002
	MsgID_S2C_MoveResponse     = 2003
	MsgID_S2C_BuildResponse    = 2004
	MsgID_S2C_WhoResponse      = 2005
	MsgID_S2C_PlayerEnter      = 2006
	MsgID_S2C_PlayerLeave      = 2007
	MsgID_S2C_PlayerMove       = 2008
)

// CoreHandler 核心协议处理器
type CoreHandler struct {
	db           database.DB
	playerMgr    *PlayerManager
	nextPlayerID int64
	// 注意：MongoDB 异步写入由 database.MongoWriter 独立线程处理
	// 不需要在 handler 中维护 batchWriter
}

// NewCoreHandler 创建核心协议处理器
func NewCoreHandler(db database.DB, playerMgr *PlayerManager) *CoreHandler {
	handler := &CoreHandler{
		db:           db,
		playerMgr:    playerMgr,
		nextPlayerID: 10001,
	}

	// 从数据库加载最大玩家 ID
	collection := db.GetCollection("players")
	allPlayers := collection.GetAll()
	if len(allPlayers) > 0 {
		var maxID int64 = 0
		for _, player := range allPlayers {
			if pid, ok := player["player_id"].(int64); ok && pid > maxID {
				maxID = pid
			}
		}
		if maxID > 0 {
			handler.nextPlayerID = maxID + 1
		}
	}

	// MongoDB 异步写入器在 database.InitMongoDB 中创建
	// 通过 db.(*database.MongoDatabase).GetWriter() 访问

	return handler
}

// Handle 处理核心协议消息
func (h *CoreHandler) Handle(sess session.Session, packet *network.Packet) *network.Packet {
	switch packet.MsgID {
	case MsgID_C2S_LoginRequest:
		return h.handleLoginRequest(sess, packet.Data)
	case MsgID_C2S_RegisterRequest:
		return h.handleRegisterRequest(sess, packet.Data)
	case MsgID_C2S_MoveRequest:
		return h.handleMoveRequest(sess, packet.Data)
	case MsgID_C2S_BuildRequest:
		return h.handleBuildRequest(sess, packet.Data)
	case MsgID_C2S_WhoRequest:
		return h.handleWhoRequest(sess, packet.Data)
	default:
		return nil
	}
}

// hashPassword 对密码进行 SHA256 哈希
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// handleLoginRequest 处理登录请求
func (h *CoreHandler) handleLoginRequest(sess session.Session, data []byte) *network.Packet {
	request := &pb.C2S_LoginRequest{}
	if err := protocol.Unmarshal(data, request); err != nil {
		log.Errorf("Failed to unmarshal login request: %v", err)
		return createLoginErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.Username == "" || request.Password == "" {
		return createLoginErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Username and password required"))
	}

	// 直接从内存查找玩家 ID（不访问 MongoDB）
	playerID, exists := h.playerMgr.GetPlayerIDByUsername(request.Username)
	if !exists {
		log.WithFields(map[string]interface{}{
			"username": request.Username,
		}).Warn("Login failed - user not found in memory")
		return createLoginErrorResponse(errors.ErrUserNotFoundErr)
	}

	// 从内存缓存验证密码（不访问 MongoDB）
	cache, cacheExists := h.playerMgr.GetPlayerCache(playerID)
	if !cacheExists {
		log.WithFields(map[string]interface{}{
			"player_id": playerID,
		}).Warn("Login failed - player cache not found")
		return createLoginErrorResponse(errors.ErrUserNotFoundErr)
	}

	hashedPassword := hashPassword(request.Password)
	if cache.PasswordHash != hashedPassword {
		log.WithFields(map[string]interface{}{
			"username": request.Username,
		}).Warn("Login failed - wrong password")
		return createLoginErrorResponse(errors.ErrWrongPasswordErr)
	}

	// 更新最后登录时间（异步写入）
	// 使用 MongoDB 异步写入器
	if mongoDB, ok := h.db.(*database.MongoDatabase); ok {
		writer := mongoDB.GetWriter()
		writer.UpdateOne("players",
			map[string]interface{}{"player_id": playerID},
			map[string]interface{}{"last_login": time.Now()},
		)
	}

	// 设置会话状态
	sess.SetPlayerID(playerID)
	sess.SetUsername(request.Username)
	sess.SetLoggedIn(true)

	// 构建玩家数据响应（从缓存或数据库获取）
	var username, email string
	var level int32
	var gold, wood, food int64

	// 尝试从缓存获取玩家数据
	if cache, cacheExists := h.playerMgr.GetPlayerCache(playerID); cacheExists {
		username = cache.Username
		email = "" // 缓存中不包含 email
		level = 1
		gold, wood, food = 1000, 1000, 1000
	} else {
		// 缓存没有，查数据库
		collection := h.db.GetCollection("players")
		playerData, _ := collection.FindOne(map[string]interface{}{"player_id": int64(playerID)})
		if playerData != nil {
			username = playerData["username"].(string)
			email = playerData["email"].(string)
			level = int32(playerData["level"].(int64))
			gold = playerData["gold"].(int64)
			wood = playerData["wood"].(int64)
			food = playerData["food"].(int64)
		}
	}

	playerDataResp := &pb.PlayerData{
		PlayerId: playerID,
		Username: username,
		Email:    email,
		Level:    level,
		Resources: map[string]int64{
			"gold": gold,
			"wood": wood,
			"food": food,
		},
	}

	response := &pb.S2C_LoginResponse{
		Success:    true,
		Message:    "Login successful",
		PlayerId:   playerID,
		PlayerData: playerDataResp,
	}

	responseData, err := protocol.Marshal(response)
	if err != nil {
		log.Errorf("Failed to marshal login response: %v", err)
		return createLoginErrorResponse(errors.ErrInternalErr)
	}

	log.WithFields(map[string]interface{}{
		"player_id": playerID,
		"username":  username,
	}).Info("Player logged in")

	return &network.Packet{
		MsgID: MsgID_S2C_LoginResponse,
		Data:  responseData,
	}
}

// handleRegisterRequest 处理注册请求
func (h *CoreHandler) handleRegisterRequest(sess session.Session, data []byte) *network.Packet {
	request := &pb.C2S_RegisterRequest{}
	if err := protocol.Unmarshal(data, request); err != nil {
		log.Errorf("Failed to unmarshal register request: %v", err)
		return createRegisterErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.Username == "" || request.Password == "" {
		return createRegisterErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Username and password required"))
	}

	// 检查用户名是否已存在
	collection := h.db.GetCollection("players")
	count, err := collection.CountDocuments(map[string]interface{}{"username": request.Username})
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

	// 生成新的玩家 ID（原子递增）
	newPlayerID := atomic.AddInt64(&h.nextPlayerID, 1) - 1

	// 创建新玩家
	hashedPassword := hashPassword(request.Password)
	newPlayer := map[string]interface{}{
		"player_id":      newPlayerID,
		"username":       request.Username,
		"password_hash":  hashedPassword,
		"email":          request.Email,
		"created_at":     time.Now(),
		"last_login":     time.Now(),
		"level":          int64(1),
		"experience":     int64(0),
		"gold":           int64(1000),
		"wood":           int64(1000),
		"food":           int64(1000),
		"population":     int64(0),
		"max_population": int64(100),
		"x":              int64(0),
		"y":              int64(0),
		"buildings":      []interface{}{},
		"troops":         []interface{}{},
		"research":       make(map[string]int32),
	}

	// 注册必须同步写入，确保立即可查
	err = collection.InsertOne(newPlayer)
	if err != nil {
		log.Errorf("Failed to create player: %v", err)
		return createRegisterErrorResponse(errors.ErrDatabaseErrorErr)
	}

	// 方案 B 关键优化：注册时同时更新内存索引和玩家缓存
	h.playerMgr.AddPlayerCache(uint64(newPlayerID), request.Username, hashedPassword)

	// 设置会话状态
	sess.SetPlayerID(uint64(newPlayerID))
	sess.SetUsername(request.Username)
	sess.SetLoggedIn(true)

	response := &pb.S2C_RegisterResponse{
		Success:  true,
		Message:  "Registration successful",
		PlayerId: uint64(newPlayerID),
	}

	responseData, err := protocol.Marshal(response)
	if err != nil {
		log.Errorf("Failed to marshal register response: %v", err)
		return createRegisterErrorResponse(errors.ErrInternalErr)
	}

	log.WithFields(map[string]interface{}{
		"player_id": newPlayerID,
		"username":  request.Username,
	}).Info("New player registered")

	return &network.Packet{
		MsgID: MsgID_S2C_RegisterResponse,
		Data:  responseData,
	}
}

// handleMoveRequest 处理移动请求
func (h *CoreHandler) handleMoveRequest(sess session.Session, data []byte) *network.Packet {
	if !sess.IsLoggedIn() {
		return createMoveErrorResponse(errors.ErrNotLoggedInErr)
	}

	request := &pb.C2S_MoveRequest{}
	if err := protocol.Unmarshal(data, request); err != nil {
		log.Errorf("Failed to unmarshal move request: %v", err)
		return createMoveErrorResponse(errors.ErrInvalidRequestErr)
	}

	// 验证坐标范围
	if request.X < -10000 || request.X > 10000 || request.Y < -10000 || request.Y > 10000 {
		return createMoveErrorResponse(errors.ErrInvalidPositionErr)
	}

	playerID := sess.GetPlayerID()

	// 更新玩家位置
	sess.SetPosition(request.X, request.Y)

	// 使用 MongoDB 异步写入器（不阻塞）
	if mongoDB, ok := h.db.(*database.MongoDatabase); ok {
		writer := mongoDB.GetWriter()
		writer.UpdateOne("players",
			map[string]interface{}{"player_id": playerID},
			map[string]interface{}{"x": request.X, "y": request.Y},
		)
	} else {
		// 降级为同步写入（MemoryDB）
		collection := h.db.GetCollection("players")
		collection.UpdateOne(
			map[string]interface{}{"player_id": playerID},
			map[string]interface{}{"x": request.X, "y": request.Y},
		)
	}

	response := &pb.S2C_MoveResponse{
		Success: true,
		Message: "Move successful",
		X:       request.X,
		Y:       request.Y,
	}

	responseData, err := protocol.Marshal(response)
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
func (h *CoreHandler) handleBuildRequest(sess session.Session, data []byte) *network.Packet {
	if !sess.IsLoggedIn() {
		return createBuildErrorResponse(errors.ErrNotLoggedInErr)
	}

	request := &pb.C2S_BuildRequest{}
	if err := protocol.Unmarshal(data, request); err != nil {
		log.Errorf("Failed to unmarshal build request: %v", err)
		return createBuildErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.BuildingType == "" {
		return createBuildErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Building type required"))
	}

	playerID := sess.GetPlayerID()

	response := &pb.S2C_BuildResponse{
		Success: true,
		Message: "Build request received",
		Building: &pb.Building{
			BuildingType: request.BuildingType,
			X:            request.X,
			Y:            request.Y,
			Level:        1,
		},
	}

	responseData, err := protocol.Marshal(response)
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

// handleWhoRequest 处理视野内玩家列表请求
func (h *CoreHandler) handleWhoRequest(sess session.Session, data []byte) *network.Packet {
	if !sess.IsLoggedIn() {
		return createWhoErrorResponse(errors.ErrNotLoggedInErr)
	}

	playerID := sess.GetPlayerID()

	// TODO: 获取视野内的玩家（应该调用 world 模块）
	// 暂时返回所有在线玩家
	sessionInfo := h.playerMgr.GetSession(playerID)
	if sessionInfo == nil {
		return createWhoErrorResponse(errors.NewError(errors.ErrInternalErr.Code, "Session not found"))
	}

	// 获取所有玩家（临时实现）
	allPlayers := h.playerMgr.GetAllPlayers()

	// 构建响应
	players := make([]*pb.WhoPlayerInfo, 0, len(allPlayers))
	for _, p := range allPlayers {
		if playerInfo, ok := p.(*session.PlayerInfo); ok {
			if playerInfo.ID == playerID || !playerInfo.Online {
				continue
			}
			players = append(players, &pb.WhoPlayerInfo{
				PlayerId: playerInfo.ID,
				Username: playerInfo.Username,
				X:        playerInfo.X,
				Y:        playerInfo.Y,
			})
		}
	}

	response := &pb.S2C_WhoResponse{
		Success: true,
		Players: players,
	}

	responseData, err := protocol.Marshal(response)
	if err != nil {
		log.Errorf("Failed to marshal who response: %v", err)
		return createWhoErrorResponse(errors.ErrInternalErr)
	}

	return &network.Packet{
		MsgID: MsgID_S2C_WhoResponse,
		Data:  responseData,
	}
}

// 错误响应辅助函数
func createLoginErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &pb.S2C_LoginResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := protocol.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_LoginResponse,
			Data:  data,
		}
	}
	return nil
}

func createRegisterErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &pb.S2C_RegisterResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := protocol.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_RegisterResponse,
			Data:  data,
		}
	}
	return nil
}

func createMoveErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &pb.S2C_MoveResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := protocol.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_MoveResponse,
			Data:  data,
		}
	}
	return nil
}

func createBuildErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &pb.S2C_BuildResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := protocol.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_BuildResponse,
			Data:  data,
		}
	}
	return nil
}

func createWhoErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &pb.S2C_WhoResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := protocol.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_WhoResponse,
			Data:  data,
		}
	}
	return nil
}
