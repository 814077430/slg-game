package core

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"slg-game/database"
	"slg-game/errors"
	"slg-game/log"
	"slg-game/network"
	pb "slg-game/protocol"
	"slg-game/protocol"
)

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

type MessageRouter struct {
	handlers  map[uint32]func(*PlayerSession, []byte) *network.Packet
	db        database.DB
	playerMgr *PlayerManager
}

func NewMessageRouter(db database.DB, playerMgr *PlayerManager) *MessageRouter {
	router := &MessageRouter{
		handlers:  make(map[uint32]func(*PlayerSession, []byte) *network.Packet),
		db:        db,
		playerMgr: playerMgr,
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
func (mr *MessageRouter) handleRegisterRequest(session *PlayerSession, data []byte) *network.Packet {
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

	// 生成新的玩家 ID
	lastPlayer, err := collection.FindOne(map[string]interface{}{})
	newPlayerID := int64(10001)
	if err == nil && lastPlayer != nil {
		if lastID, ok := lastPlayer["player_id"].(int64); ok {
			newPlayerID = lastID + 1
		}
	}

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
func (mr *MessageRouter) handleMoveRequest(session *PlayerSession, data []byte) *network.Packet {
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
func (mr *MessageRouter) handleBuildRequest(session *PlayerSession, data []byte) *network.Packet {
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
func (mr *MessageRouter) handleWhoRequest(session *PlayerSession, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createWhoErrorResponse(errors.ErrNotLoggedInErr)
	}

	playerID := session.GetPlayerID()
	
	// 获取视野内的玩家
	visiblePlayers := mr.playerMgr.GetPlayersInVision(playerID)
	
	// 构建响应
	players := make([]*pb.WhoPlayerInfo, 0, len(visiblePlayers))
	for _, p := range visiblePlayers {
		players = append(players, &pb.WhoPlayerInfo{
			PlayerId: p.ID,
			Username: p.Username,
			X:        p.X,
			Y:        p.Y,
		})
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
	if mr.playerMgr == nil {
		return
	}
	
	// 获取视野内的其他玩家
	visiblePlayers := mr.playerMgr.GetPlayersInVision(playerID)
	
	notification := &pb.PlayerEnterNotification{
		PlayerId: playerID,
		X:        x,
		Y:        y,
	}
	data, _ := protocol.Marshal(notification)
	
	// 通知视野内的玩家
	for _, p := range visiblePlayers {
		session := mr.playerMgr.GetSession(p.ID)
		if session != nil {
			session.SendPacket(&network.Packet{
				MsgID: MsgID_S2C_PlayerEnter,
				Data:  data,
			})
		}
	}
}

// notifyPlayerLeave 通知视野内玩家有玩家离开
func (mr *MessageRouter) notifyPlayerLeave(playerID uint64) {
	if mr.playerMgr == nil {
		return
	}
	
	visiblePlayers := mr.playerMgr.GetPlayersInVision(playerID)
	
	notification := &pb.PlayerLeaveNotification{
		PlayerId: playerID,
	}
	data, _ := protocol.Marshal(notification)
	
	for _, p := range visiblePlayers {
		session := mr.playerMgr.GetSession(p.ID)
		if session != nil {
			session.SendPacket(&network.Packet{
				MsgID: MsgID_S2C_PlayerLeave,
				Data:  data,
			})
		}
	}
}

// notifyPlayerMove 通知视野内玩家移动
func (mr *MessageRouter) notifyPlayerMove(playerID uint64, x, y int32) {
	if mr.playerMgr == nil {
		return
	}
	
	visiblePlayers := mr.playerMgr.GetPlayersInVision(playerID)
	
	notification := &pb.PlayerMoveNotification{
		PlayerId: playerID,
		X:        x,
		Y:        y,
	}
	data, _ := protocol.Marshal(notification)
	
	for _, p := range visiblePlayers {
		session := mr.playerMgr.GetSession(p.ID)
		if session != nil {
			session.SendPacket(&network.Packet{
				MsgID: MsgID_S2C_PlayerMove,
				Data:  data,
			})
		}
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
