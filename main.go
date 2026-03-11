package main

import (
	"context"
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

	log.Info("Starting SLG Game Server...")

	// 加载配置
	cfg := config.LoadConfig("config/game.json")
	if cfg == nil {
		log.Fatal("Failed to load config")
	}
	log.Info("Config loaded successfully")

	// 初始化 MongoDB 连接
	db, err := database.InitMongoDB(cfg.Database.URL, cfg.Database.DatabaseName)
	if err != nil {
		log.Warnf("MongoDB connection failed: %v", err)
		log.Warn("Install MongoDB to enable data persistence")
		log.Warn("Download: https://www.mongodb.com/try/download/community")
		log.Warn("")
		log.Warn("Starting server anyway (login/register will fail without DB)...")
	}
	defer func() {
		if db != nil && db.Client() != nil {
			db.Client().Disconnect(context.Background())
		}
	}()

	// 初始化游戏服务器
	gameServer := game.NewGameServer(db, cfg)
	log.Info("Game server initialized")

	// 启动 TCP 服务器
	listener, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Infof("Game server started on %s", cfg.Server.Addr)

	// 处理优雅关闭
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("Shutting down server...")
		gameServer.Shutdown()
		listener.Close()
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
