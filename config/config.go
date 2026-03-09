package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Game     GameConfig     `json:"game"`
}

type ServerConfig struct {
	Addr           string `json:"addr"`
	MaxConnections int    `json:"max_connections"`
	ReadTimeout    int    `json:"read_timeout"`
	WriteTimeout   int    `json:"write_timeout"`
}

type DatabaseConfig struct {
	URL          string `json:"url"`
	DatabaseName string `json:"database_name"`
	MaxPoolSize  int    `json:"max_pool_size"`
}

type GameConfig struct {
	TickInterval int `json:"tick_interval"`
	MaxPlayers   int `json:"max_players"`
}

func LoadConfig(filePath string) *Config {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open config file: %v", err)
		return nil
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Printf("Failed to decode config file: %v", err)
		return nil
	}

	return &config
}