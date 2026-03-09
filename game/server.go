package game

import (
	"log"

	"slg-game/config"
	"slg-game/database"
	"slg-game/network"
)

type GameServer struct {
	db     *database.Database
	config *config.Config
	router *MessageRouter
}

func NewGameServer(db *database.Database, config *config.Config) *GameServer {
	return &GameServer{
		db:     db,
		config: config,
		router: NewMessageRouter(),
	}
}

func (gs *GameServer) HandleClient(conn net.Conn) {
	defer conn.Close()

	connection := network.NewConnection(conn)
	connection.Start()

	log.Printf("New client connected: %s", conn.RemoteAddr())

	// 创建玩家会话
	session := NewPlayerSession(connection, gs.db, gs.config)

	for {
		packet, err := connection.ReadPacket()
		if err != nil {
			log.Printf("Read packet error from %s: %v", conn.RemoteAddr(), err)
			break
		}

		// 路由消息到对应的处理器
		responsePacket := gs.router.Route(session, packet)
		if responsePacket != nil {
			if err := connection.SendPacket(responsePacket); err != nil {
				log.Printf("Send packet error to %s: %v", conn.RemoteAddr(), err)
				break
			}
		}
	}

	session.Cleanup()
	log.Printf("Client disconnected: %s", conn.RemoteAddr())
}