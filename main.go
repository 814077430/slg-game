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

	db := database.NewMemoryDB()
	log.Info("✓ Memory database initialized (MongoDB disabled)")

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
