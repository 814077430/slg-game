package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"slg-game/config"
	"slg-game/database"
	"slg-game/game"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig("config/game.json")
	if cfg == nil {
		log.Fatal("Failed to load config")
	}

	// 初始化 MongoDB 连接
	db, err := database.InitMongoDB(cfg.Database.URL, cfg.Database.DatabaseName)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.Client().Disconnect(context.Background())

	// 初始化游戏服务器
	gameServer := game.NewGameServer(db, cfg)

	// 启动 TCP 服务器
	listener, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Printf("Game server started on %s", cfg.Server.Addr)

	// 处理优雅关闭
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down server...")
		listener.Close()
		os.Exit(0)
	}()

	// 接受客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		// 为每个客户端启动一个 goroutine
		go gameServer.HandleClient(conn)
	}
}
