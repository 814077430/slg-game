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
	"slg-game/handler"
	"slg-game/messenger"
)

type GameServer struct {
	db          database.DB
	config      *config.Config
	router      *network.Router
	gameLoop    *GameLoop
	world       *world.World
	players     *PlayerManager
	chatMgr     *chat.ChatManager
	mongoWriter *database.MongoAsyncWriter
	messageBus  *messenger.MessageBus  // 消息总线
	
	// 模块管理器
	buildingMgr  *city.BuildingManager
	resourceMgr  *resource.ResourceManager
	battleMgr    *battle.BattleManager
	allianceMgr  *alliance.AllianceManager
	techMgr      *tech.TechnologyManager
	
	// 协议处理器
	coreHandler  *CoreHandler
	chatHandler  *chat.ChatHandler
}

func NewGameServer(db database.DB, cfg *config.Config) *GameServer {
	// 创建消息总线
	messageBus := messenger.NewMessageBus()

	// 创建 MongoDB 异步写入器（100 条或 100ms 批量写入）
	var mongoWriter *database.MongoAsyncWriter
	if _, ok := db.(*database.MongoDatabase); ok {
		// MongoDB 可用，创建异步写入器
		mongoWriter, _ = database.NewMongoAsyncWriter(
			"mongodb://localhost:27017",
			"slg_game",
			100,
			100*time.Millisecond,
		)
	}

	// 创建世界实例（独立线程）
	world := world.NewWorld(db, messageBus)

	// 创建玩家管理器（传入 MongoDB 异步写入器）
	players := NewPlayerManager(mongoWriter)

	// 创建聊天管理器（独立线程）
	chatMgr := chat.NewChatManager(players, messageBus)

	// 创建协议处理器
	coreHandler := NewCoreHandler(db, players)
	chatHandler := chat.NewChatHandler(chatMgr)

	// 创建统一路由器
	router := network.NewRouter()
	
	// 注册消息处理器（按消息 ID 范围）
	router.RegisterRangeHandler(1000, 1999, coreHandler)  // 核心协议
	router.RegisterRangeHandler(4000, 4999, chatHandler)  // 聊天协议

	// 创建游戏主循环（独立线程）
	tickInterval := time.Duration(cfg.Game.TickInterval) * time.Millisecond
	gameLoop := NewGameLoop(db, tickInterval, messageBus)

	// 初始化各模块管理器
	buildingMgr := city.NewBuildingManager(db)
	resourceMgr := resource.NewResourceManager(db)
	battleMgr := battle.NewArmyManager(db, messageBus).GetBattleManager()
	allianceMgr := alliance.NewAllianceManager(db)
	techMgr := tech.NewTechnologyManager(db)

	// 启动独立线程
	world.StartLoop()       // World 独立循环
	gameLoop.Start()        // GameLoop 独立循环
	battleMgr.StartLoop()   // Battle 独立循环
	chatMgr.StartLoop()     // Chat 独立循环

	return &GameServer{
		db:           db,
		config:       cfg,
		router:       router,
		gameLoop:     gameLoop,
		world:        world,
		players:      players,
		chatMgr:      chatMgr,
		mongoWriter:  mongoWriter,
		messageBus:   messageBus,
		buildingMgr:  buildingMgr,
		resourceMgr:  resourceMgr,
		battleMgr:    battleMgr,
		allianceMgr:  allianceMgr,
		techMgr:      techMgr,
		coreHandler:  coreHandler,
		chatHandler:  chatHandler,
	}
}

func (gs *GameServer) HandleClient(conn net.Conn) {
	defer conn.Close()

	connection := network.NewConnection(conn)
	connection.Start()

	log.Printf("New client connected: %s", conn.RemoteAddr())

	// 创建玩家会话
	session := handler.NewPlayerSession(connection, gs.db, gs.config, gs.players)

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
	
	// 停止消息总线
	if gs.messageBus != nil {
		gs.messageBus.Stop()
	}
	
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
	if gs.players != nil {
		gs.players.Stop()
	}
	// 停止 MongoDB 异步写入器（等待剩余数据写入）
	if gs.mongoWriter != nil {
		gs.mongoWriter.Stop()
	}
	
	// 异步日志器会在程序退出时自动停止
	
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
