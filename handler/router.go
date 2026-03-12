package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync/atomic"
	"time"

	"slg-game/database"
	"slg-game/errors"
	"slg-game/chat"
	"slg-game/log"
	"slg-game/network"
	pb "slg-game/protocol"
	"slg-game/protocol"
	sessionPkg "slg-game/session"
)

// PlayerManager 接口（避免循环导入）
type PlayerManager interface {
	AddPlayer(playerID uint64, username string, session sessionPkg.Session)
	AddPlayerCache(playerID uint64, username, passwordHash string)
	RemovePlayer(playerID uint64)
	UpdatePlayerPosition(playerID uint64, x, y int32)
	GetSession(playerID uint64) sessionPkg.Session
	GetAllPlayers() []interface{}
	GetPlayerIDByUsername(username string) (uint64, bool)
}

const (
	MsgID_C2S_LoginRequest     = 1001
	MsgID_C2S_RegisterRequest  = 1002
	MsgID_C2S_MoveRequest      = 1003
	MsgID_C2S_BuildRequest     = 1004
	MsgID_C2S_WhoRequest       = 1005
	MsgID_C2S_ChatRequest      = 1010
	MsgID_S2C_LoginResponse    = 2001
	MsgID_S2C_RegisterResponse = 2002
	MsgID_S2C_MoveResponse     = 2003
	MsgID_S2C_BuildResponse    = 2004
	MsgID_S2C_WhoResponse      = 2005
	MsgID_S2C_PlayerEnter      = 2006
	MsgID_S2C_PlayerLeave      = 2007
	MsgID_S2C_PlayerMove       = 2008
	MsgID_S2C_ChatResponse     = 2010
	MsgID_S2C_ChatBroadcast    = 2011
)

type MessageRouter struct {
	handlers     map[uint32]func(sessionPkg.Session, []byte) *network.Packet
	db           database.DB
	playerMgr    PlayerManager
	chatMgr      *chat.ChatManager
	nextPlayerID int64
}

func NewMessageRouter(db database.DB, playerMgr PlayerManager, chatMgr *chat.ChatManager) *MessageRouter {
	router := &MessageRouter{
		handlers:     make(map[uint32]func(sessionPkg.Session, []byte) *network.Packet),
		db:           db,
		playerMgr:    playerMgr,
		chatMgr:      chatMgr,
		nextPlayerID: 10001,
	}
	
	// 从数据库加载最大玩家 ID（遍历查找最大值）
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
			router.nextPlayerID = maxID + 1
		}
	}
	
	router.registerHandlers()
	return router
}

func (mr *MessageRouter) registerHandlers() {
	mr.handlers[MsgID_C2S_LoginRequest] = mr.handleLoginRequest
	mr.handlers[MsgID_C2S_RegisterRequest] = mr.handleRegisterRequest
	mr.handlers[MsgID_C2S_MoveRequest] = mr.handleMoveRequest
	mr.handlers[MsgID_C2S_BuildRequest] = mr.handleBuildRequest
	mr.handlers[MsgID_C2S_WhoRequest] = mr.handleWhoRequest
	mr.handlers[MsgID_C2S_ChatRequest] = mr.handleChatRequest
}

func (mr *MessageRouter) Route(session sessionPkg.Session, packet *network.Packet) *network.Packet {
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
func (mr *MessageRouter) handleLoginRequest(session sessionPkg.Session, data []byte) *network.Packet {
	request := &pb.C2S_LoginRequest{}
	if err := protocol.Unmarshal(data, request); err != nil {
		log.Errorf("Failed to unmarshal login request: %v", err)
		return createLoginErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.Username == "" || request.Password == "" {
		return createLoginErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Username and password required"))
	}

	// 查询数据库
	collection := mr.db.GetCollection("players")
	player, err := collection.FindOne(map[string]interface{}{"username": request.Username})
	if err != nil {
		log.WithFields(map[string]interface{}{
			"username": request.Username,
		}).Warn("Login failed - user not found")
		return createLoginErrorResponse(errors.ErrUserNotFoundErr)
	}

	// 验证密码
	hashedPassword := hashPassword(request.Password)
	if player["password_hash"] != hashedPassword {
		log.WithFields(map[string]interface{}{
			"username": request.Username,
		}).Warn("Login failed - wrong password")
		return createLoginErrorResponse(errors.ErrWrongPasswordErr)
	}

	// 更新最后登录时间
	collection.UpdateOne(
		map[string]interface{}{"player_id": player["player_id"]},
		map[string]interface{}{"last_login": time.Now()},
	)

	// 设置会话状态
	var playerID uint64
	switch v := player["player_id"].(type) {
	case int64:
		playerID = uint64(v)
	case uint64:
		playerID = v
	}
	
	session.SetPlayerID(playerID)
	session.SetUsername(player["username"].(string))
	session.SetLoggedIn(true)

	// 构建玩家数据响应
	playerData := &pb.PlayerData{
		PlayerId: playerID,
		Username: player["username"].(string),
		Email:    player["email"].(string),
		Level:    int32(player["level"].(int64)),
		Resources: map[string]int64{
			"gold":  player["gold"].(int64),
			"wood":  player["wood"].(int64),
			"food":  player["food"].(int64),
		},
	}

	response := &pb.S2C_LoginResponse{
		Success:    true,
		Message:    "Login successful",
		PlayerId:   uint64(player["player_id"].(int64)),
		PlayerData: playerData,
	}

	responseData, err := protocol.Marshal(response)
	if err != nil {
		log.Errorf("Failed to marshal login response: %v", err)
		return createLoginErrorResponse(errors.ErrInternalErr)
	}

	log.WithFields(map[string]interface{}{
		"player_id": player["player_id"],
		"username":  player["username"],
	}).Info("Player logged in")

	return &network.Packet{
		MsgID: MsgID_S2C_LoginResponse,
		Data:  responseData,
	}
}

// handleRegisterRequest 处理注册请求
func (mr *MessageRouter) handleRegisterRequest(session sessionPkg.Session, data []byte) *network.Packet {
	request := &pb.C2S_RegisterRequest{}
	if err := protocol.Unmarshal(data, request); err != nil {
		log.Errorf("Failed to unmarshal register request: %v", err)
		return createRegisterErrorResponse(errors.ErrInvalidRequestErr)
	}

	if request.Username == "" || request.Password == "" {
		return createRegisterErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Username and password required"))
	}

	// 检查用户名是否已存在
	collection := mr.db.GetCollection("players")
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
	newPlayerID := atomic.AddInt64(&mr.nextPlayerID, 1) - 1

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

	err = collection.InsertOne(newPlayer)
	if err != nil {
		log.Errorf("Failed to create player: %v", err)
		return createRegisterErrorResponse(errors.ErrDatabaseErrorErr)
	}

	// 设置会话状态
	session.SetPlayerID(uint64(newPlayerID))
	session.SetUsername(request.Username)
	session.SetLoggedIn(true)

	// 添加到玩家管理器缓存
	mr.playerMgr.AddPlayer(uint64(newPlayerID), request.Username, session)
	mr.playerMgr.AddPlayerCache(uint64(newPlayerID), request.Username, hashedPassword)

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
func (mr *MessageRouter) handleMoveRequest(session sessionPkg.Session, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
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

	playerID := session.GetPlayerID()

	// 更新玩家位置
	session.SetPosition(request.X, request.Y)

	// 更新数据库
	collection := mr.db.GetCollection("players")
	collection.UpdateOne(
		map[string]interface{}{"player_id": playerID},
		map[string]interface{}{"x": request.X, "y": request.Y},
	)

	// 通知视野内其他玩家
	mr.notifyPlayerMove(playerID, request.X, request.Y)

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
func (mr *MessageRouter) handleBuildRequest(session sessionPkg.Session, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
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

	playerID := session.GetPlayerID()

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
func (mr *MessageRouter) handleWhoRequest(session sessionPkg.Session, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createWhoErrorResponse(errors.ErrNotLoggedInErr)
	}

	playerID := session.GetPlayerID()
	
	// TODO: 获取视野内的玩家（应该调用 world 模块）
	// 暂时返回所有在线玩家
	sessionInfo := mr.playerMgr.GetSession(playerID)
	if sessionInfo == nil {
		return createWhoErrorResponse(errors.NewError(errors.ErrInternalErr.Code, "Session not found"))
	}
	
	// 获取所有玩家（临时实现）
	allPlayers := mr.playerMgr.GetAllPlayers()
	
	// 构建响应
	players := make([]*pb.WhoPlayerInfo, 0, len(allPlayers))
	for _, p := range allPlayers {
		if playerInfo, ok := p.(*sessionPkg.PlayerInfo); ok {
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

// notifyPlayerEnter 通知视野内玩家有新玩家进入
func (mr *MessageRouter) notifyPlayerEnter(playerID uint64, x, y int32) {
	// 视野逻辑已移到 world 模块
	// TODO: 通过 world.GetPlayersInVision() 获取视野内玩家
}

// notifyPlayerLeave 通知视野内玩家有玩家离开
func (mr *MessageRouter) notifyPlayerLeave(playerID uint64) {
	// 视野逻辑已移到 world 模块
	// TODO: 通过 world.GetPlayersInVision() 获取视野内玩家
}

// notifyPlayerMove 通知视野内玩家移动
func (mr *MessageRouter) notifyPlayerMove(playerID uint64, x, y int32) {
	// 视野逻辑已移到 world 模块
	// TODO: 通过 world.GetPlayersInVision() 获取视野内玩家
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

// handleChatRequest 处理聊天消息
func (mr *MessageRouter) handleChatRequest(session sessionPkg.Session, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createChatErrorResponse(errors.ErrNotLoggedInErr)
	}

	var content, channel string
	
	// 尝试解析为 Protobuf
	request := &pb.C2S_ChatRequest{}
	if err := protocol.Unmarshal(data, request); err == nil && request.Content != "" {
		content = request.Content
		channel = request.Channel
	} else {
		// 回退到文本解析（格式：channel content）
		text := strings.TrimSpace(string(data))
		parts := strings.SplitN(text, " ", 2)
		if len(parts) >= 2 {
			channel = parts[0]
			content = parts[1]
		} else if len(parts) == 1 {
			channel = "world"
			content = parts[0]
		}
	}

	// 验证消息内容
	if len(content) == 0 || len(content) > 500 {
		return createChatErrorResponse(errors.NewError(errors.ErrInvalidRequest, "Message length must be 1-500 characters"))
	}

	// 发送到聊天管理器
	if channel == "" {
		channel = "world"
	}

	mr.chatMgr.SendChat(session, content, channel)

	// 返回成功响应
	response := &pb.S2C_ChatResponse{
		Success:   true,
		Message:   "Message sent",
		Timestamp: time.Now().UnixMilli(),
	}

	responseData, err := protocol.Marshal(response)
	if err != nil {
		log.Errorf("Failed to marshal chat response: %v", err)
		return createChatErrorResponse(errors.ErrInternalErr)
	}

	return &network.Packet{
		MsgID: MsgID_S2C_ChatResponse,
		Data:  responseData,
	}
}

// createChatErrorResponse 创建聊天错误响应
func createChatErrorResponse(errDetail *errors.ErrorDetail) *network.Packet {
	response := &pb.S2C_ChatResponse{
		Success: false,
		Message: errDetail.Message,
	}
	if data, err := protocol.Marshal(response); err == nil {
		return &network.Packet{
			MsgID: MsgID_S2C_ChatResponse,
			Data:  data,
		}
	}
	return nil
}
