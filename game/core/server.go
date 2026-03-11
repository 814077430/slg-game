package core

import (
	"log"
	"net"
	"time"

	"slg-game/config"
	"slg-game/database"
	"slg-game/game/city"
	"slg-game/network"
	"slg-game/world"
	"slg-game/game/resource"
	"slg-game/battle"
	"slg-game/game/alliance"
	"slg-game/game/tech"
	"slg-game/chat"
)

type GameServer struct {
	db       database.DB
	config   *config.Config
	router   *MessageRouter
	gameLoop *GameLoop
	world    *world.World
	players  *PlayerManager
	chatMgr  *chat.ChatManager
	
	// 模块管理器
	buildingMgr  *city.BuildingManager
	resourceMgr  *resource.ResourceManager
	battleMgr    *battle.BattleManager
	allianceMgr  *alliance.AllianceManager
	techMgr      *tech.TechnologyManager
}

func NewGameServer(db database.DB, cfg *config.Config) *GameServer {
	// 创建世界实例（独立线程）
	world := world.NewWorld(db)

	// 创建玩家管理器（视野范围 10 格）
	players := NewPlayerManager(10)

	// 创建聊天管理器（独立线程）
	chatMgr := chat.NewChatManager(players)

	// 创建消息路由器
	router := NewMessageRouter(db, players, chatMgr)

	// 创建游戏主循环（独立线程）
	tickInterval := time.Duration(cfg.Game.TickInterval) * time.Millisecond
	gameLoop := NewGameLoop(db, tickInterval)

	// 初始化各模块管理器
	buildingMgr := city.NewBuildingManager(db)
	resourceMgr := resource.NewResourceManager(db)
	battleMgr := battle.NewBattleManager(db)
	allianceMgr := alliance.NewAllianceManager(db)
	techMgr := tech.NewTechnologyManager(db)

	// 启动独立线程
	world.StartLoop()      // World 独立循环
	gameLoop.Start()       // GameLoop 独立循环
	battleMgr.StartLoop()  // Battle 独立循环
	chatMgr.StartLoop()    // Chat 独立循环

	return &GameServer{
		db:           db,
		config:       cfg,
		router:       router,
		gameLoop:     gameLoop,
		world:        world,
		players:      players,
		chatMgr:      chatMgr,
		buildingMgr:  buildingMgr,
		resourceMgr:  resourceMgr,
		battleMgr:    battleMgr,
		allianceMgr:  allianceMgr,
		techMgr:      techMgr,
	}
}

func (gs *GameServer) HandleClient(conn net.Conn) {
	defer conn.Close()

	connection := network.NewConnection(conn)
	connection.Start()

	log.Printf("New client connected: %s", conn.RemoteAddr())

	// 创建玩家会话
	session := NewPlayerSession(connection, gs.db, gs.config, gs.players)

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

// Shutdown 优雅关闭服务器
func (gs *GameServer) Shutdown() {
	log.Println("Shutting down server...")
	
	// 停止所有独立线程
	if gs.gameLoop != nil {
		gs.gameLoop.Stop()
	}
	if gs.world != nil {
		gs.world.StopLoop()
	}
	if gs.battleMgr != nil {
		gs.battleMgr.Stop()
	}
	if gs.chatMgr != nil {
		gs.chatMgr.StopLoop()
	}
	
	log.Println("Game server shutdown complete")
}

// GetWorld 获取世界实例
func (gs *GameServer) GetWorld() *world.World {
	return gs.world
}

// GetGameLoop 获取游戏主循环
func (gs *GameServer) GetGameLoop() *GameLoop {
	return gs.gameLoop
}
