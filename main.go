package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"slg-game/config"
	"slg-game/database"
	"slg-game/game"
	"slg-game/log"
)

func main() {
	// 设置日志级别
	log.SetLevel(log.InfoLevel)

	log.Info("╔════════════════════════════════════════════════════════╗")
	log.Info("║          SLG Game Server - Starting                    ║")
	log.Info("╚════════════════════════════════════════════════════════╝")

	// 加载配置
	cfg := config.LoadConfig("config/game.json")
	if cfg == nil {
		log.Fatal("Failed to load config")
	}
	log.Info("✓ Config loaded")

	// 初始化数据库（内存模式）
	db := database.NewMemoryDB()
	log.Info("✓ Memory database initialized (MongoDB disabled)")

	// 初始化游戏服务器
	gameServer := game.NewGameServer(db, cfg)
	log.Info("✓ Game server initialized")

	// 启动 TCP 服务器
	listener, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Infof("✓ Game server listening on %s", cfg.Server.Addr)
	log.Info("")
	log.Info("═══════════════════════════════════════════════════════")
	log.Info("Server is ready!")
	log.Info("")
	log.Info("Test with:")
	log.Info("  cd client && ./slg-client -server localhost:8080 -test all")
	log.Info("═══════════════════════════════════════════════════════")
	log.Info("")

	// 处理优雅关闭
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("")
		log.Info("Shutting down server...")
		gameServer.Shutdown()
		listener.Close()
		log.Info("Server stopped")
		os.Exit(0)
	}()

	// 接受客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Accept error: %v", err)
			continue
		}

		log.Infof("New client connected: %s", conn.RemoteAddr())

		// 为每个客户端启动一个 goroutine
		go gameServer.HandleClient(conn)
	}
}
