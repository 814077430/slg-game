package core

import (
	"log"
	"net"
	"time"

	"slg-game/config"
	"slg-game/database"
	"slg-game/game/city"
	"slg-game/network"
	"slg-game/game/world"
	"slg-game/game/resource"
	"slg-game/game/army"
	"slg-game/game/alliance"
	"slg-game/game/tech"
)

type GameServer struct {
	db       *database.MemoryDB
	config   *config.Config
	router   *MessageRouter
	gameLoop *GameLoop
	world    *world.World
	
	// 模块管理器
	buildingMgr  *city.BuildingManager
	resourceMgr  *resource.ResourceManager
	armyMgr      *army.ArmyManager
	allianceMgr  *alliance.AllianceManager
	techMgr      *tech.TechnologyManager
}

func NewGameServer(db *database.MemoryDB, cfg *config.Config) *GameServer {
	// 创建消息路由器
	router := NewMessageRouter(db)

	// 创建世界实例
	world := world.NewWorld(db)

	// 创建游戏主循环
	tickInterval := time.Duration(cfg.Game.TickInterval) * time.Millisecond
	gameLoop := NewGameLoop(db, tickInterval, world)

	// 初始化各模块管理器
	buildingMgr := city.NewBuildingManager(db)
	resourceMgr := resource.NewResourceManager(db)
	armyMgr := army.NewArmyManager(db)
	allianceMgr := alliance.NewAllianceManager(db)
	techMgr := tech.NewTechnologyManager(db)

	// 启动游戏主循环
	gameLoop.Start()

	return &GameServer{
		db:           db,
		config:       cfg,
		router:       router,
		gameLoop:     gameLoop,
		world:        world,
		buildingMgr:  buildingMgr,
		resourceMgr:  resourceMgr,
		armyMgr:      armyMgr,
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

// Shutdown 优雅关闭服务器
func (gs *GameServer) Shutdown() {
	if gs.gameLoop != nil {
		gs.gameLoop.Stop()
	}
	if gs.world != nil {
		gs.world.StopGameLoop()
	}
	log.Println("Game server shutdown complete")
}

// GetBuildingMgr 获取建筑管理器
func (gs *GameServer) GetBuildingMgr() *city.BuildingManager {
	return gs.buildingMgr
}

// GetResourceMgr 获取资源管理器
func (gs *GameServer) GetResourceMgr() *resource.ResourceManager {
	return gs.resourceMgr
}

// GetArmyMgr 获取军队管理器
func (gs *GameServer) GetArmyMgr() *army.ArmyManager {
	return gs.armyMgr
}

// GetAllianceMgr 获取联盟管理器
func (gs *GameServer) GetAllianceMgr() *alliance.AllianceManager {
	return gs.allianceMgr
}

// GetTechMgr 获取科技管理器
func (gs *GameServer) GetTechMgr() *tech.TechnologyManager {
	return gs.techMgr
}
