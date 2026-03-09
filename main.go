package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// 加载配置
	config := LoadConfig("config/game.json")
	if config == nil {
		log.Fatal("Failed to load config")
	}

	// 初始化MongoDB连接
	db, err := InitMongoDB(config.Database.URL, config.Database.DatabaseName)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.Client().Disconnect(context.Background())

	// 初始化游戏服务器
	gameServer := NewGameServer(db, config)

	// 启动TCP服务器
	listener, err := net.Listen("tcp", config.Server.Addr)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Printf("Game server started on %s", config.Server.Addr)

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

		// 为每个客户端启动一个goroutine
		go gameServer.HandleClient(conn)
	}
}