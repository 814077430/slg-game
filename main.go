package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"slg-game/config"
	"slg-game/database"
	"slg-game/game/core"
	"slg-game/log"
)

func main() {
	log.SetLevel(log.InfoLevel)

	log.Info("╔════════════════════════════════════════════════════════╗")
	log.Info("║          SLG Game Server - Starting                    ║")
	log.Info("╚════════════════════════════════════════════════════════╝")

	cfg := config.LoadConfig("config/game.json")
	if cfg == nil {
		log.Fatal("Failed to load config")
	}
	log.Info("✓ Config loaded")

	// 使用 MongoDB（数据持久化 + 连接池优化 + 批量写入）
	var db database.DB
	
	mongoDB, err := database.InitMongoDB("mongodb://localhost:27017", "slg_game")
	if err != nil {
		log.Warnf("MongoDB connection failed: %v", err)
		log.Warn("Falling back to memory database")
		db = database.NewMemoryDB()
		log.Info("✓ Memory database initialized")
	} else {
		db = mongoDB
		log.Info("✓ MongoDB connected successfully")
		log.Info("  Database: slg_game")
		log.Info("  Connection pool: Max 100, Min 20")
		log.Info("  Batch write: 500 ops/50ms")
		log.Info("  Data will be persisted across restarts")
	}
	defer db.Disconnect()

	gameServer := core.NewGameServer(db, cfg)
	log.Info("✓ Game server initialized")

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
	log.Info("  cd client && ./slg-stress -clients 100 -requests 10")
	log.Info("═══════════════════════════════════════════════════════")
	log.Info("")

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

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Accept error: %v", err)
			continue
		}

		log.Infof("New client connected: %s", conn.RemoteAddr())
		go gameServer.HandleClient(conn)
	}
}
