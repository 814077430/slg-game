package game

import (
	"log"

	"github.com/golang/protobuf/proto"
	"slg-game/network"
	"slg-game/proto"
)

const (
	MsgID_C2S_LoginRequest  = 1001
	MsgID_C2S_MoveRequest   = 1002
	MsgID_C2S_BuildRequest  = 1003
	MsgID_S2C_LoginResponse = 2001
	MsgID_S2C_MoveResponse  = 2002
	MsgID_S2C_BuildResponse = 2003
	MsgID_S2C_PlayerUpdate  = 2004
)

type MessageRouter struct {
	handlers map[uint32]func(*PlayerSession, []byte) *network.Packet
}

func NewMessageRouter() *MessageRouter {
	router := &MessageRouter{
		handlers: make(map[uint32]func(*PlayerSession, []byte) *network.Packet),
	}
	router.registerHandlers()
	return router
}

func (mr *MessageRouter) registerHandlers() {
	mr.handlers[MsgID_C2S_LoginRequest] = handleLoginRequest
	mr.handlers[MsgID_C2S_MoveRequest] = handleMoveRequest
	mr.handlers[MsgID_C2S_BuildRequest] = handleBuildRequest
}

func (mr *MessageRouter) Route(session *PlayerSession, packet *network.Packet) *network.Packet {
	handler, exists := mr.handlers[packet.MsgID]
	if !exists {
		log.Printf("Unknown message ID: %d", packet.MsgID)
		return nil
	}

	return handler(session, packet.Data)
}

// C2S handlers - 在这里实现你的游戏逻辑
func handleLoginRequest(session *PlayerSession, data []byte) *network.Packet {
	request := &proto.C2S_LoginRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		log.Printf("Failed to unmarshal login request: %v", err)
		return createErrorResponse(MsgID_S2C_LoginResponse, "Invalid login request")
	}

	// TODO: 在这里实现登录逻辑
	// 1. 验证用户名和密码
	// 2. 查询数据库
	// 3. 创建或加载玩家数据
	// 4. 设置会话状态

	response := &proto.S2C_LoginResponse{
		Success:  true,
		Message:  "Login successful",
		PlayerId: 12345, // 示例玩家ID
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal login response: %v", err)
		return createErrorResponse(MsgID_S2C_LoginResponse, "Internal error")
	}

	session.SetPlayerID(response.PlayerId)
	session.SetUsername(request.Username)
	session.SetLoggedIn(true)

	return &network.Packet{
		MsgID: MsgID_S2C_LoginResponse,
		Data:  responseData,
	}
}

func handleMoveRequest(session *PlayerSession, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createErrorResponse(MsgID_S2C_MoveResponse, "Not logged in")
	}

	request := &proto.C2S_MoveRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		log.Printf("Failed to unmarshal move request: %v", err)
		return createErrorResponse(MsgID_S2C_MoveResponse, "Invalid move request")
	}

	// TODO: 在这里实现移动逻辑
	// 1. 验证移动坐标
	// 2. 更新玩家位置
	// 3. 保存到数据库
	// 4. 广播给其他玩家（如果需要）

	response := &proto.S2C_MoveResponse{
		Success: true,
		Message: "Move successful",
		X:       request.X,
		Y:       request.Y,
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal move response: %v", err)
		return createErrorResponse(MsgID_S2C_MoveResponse, "Internal error")
	}

	return &network.Packet{
		MsgID: MsgID_S2C_MoveResponse,
		Data:  responseData,
	}
}

func handleBuildRequest(session *PlayerSession, data []byte) *network.Packet {
	if !session.IsLoggedIn() {
		return createErrorResponse(MsgID_S2C_BuildResponse, "Not logged in")
	}

	request := &proto.C2S_BuildRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		log.Printf("Failed to unmarshal build request: %v", err)
		return createErrorResponse(MsgID_S2C_BuildResponse, "Invalid build request")
	}

	// TODO: 在这里实建造逻辑
	// 1. 验证建筑类型和位置
	// 2. 检查资源是否足够
	// 3. 扣除资源
	// 4. 创建建筑
	// 5. 保存到数据库

	response := &proto.S2C_BuildResponse{
		Success:      true,
		Message:      "Build successful",
		BuildingType: request.BuildingType,
		X:            request.X,
		Y:            request.Y,
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal build response: %v", err)
		return createErrorResponse(MsgID_S2C_BuildResponse, "Internal error")
	}

	return &network.Packet{
		MsgID: MsgID_S2C_BuildResponse,
		Data:  responseData,
	}
}

func createErrorResponse(msgID uint32, message string) *network.Packet {
	// 这里需要根据具体的响应类型创建错误响应
	// 为了简化，我们只处理登录错误
	if msgID == MsgID_S2C_LoginResponse {
		response := &proto.S2C_LoginResponse{
			Success: false,
			Message: message,
		}
		if data, err := proto.Marshal(response); err == nil {
			return &network.Packet{
				MsgID: msgID,
				Data:  data,
			}
		}
	}
	return nil
}