package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"slg-game/session"
)

// PlayerCache 玩家数据缓存
type PlayerCache struct {
	client  *redis.Client
	prefix  string
	expiry  time.Duration
}

// NewPlayerCache 创建玩家缓存
func NewPlayerCache(addr, password string, db int) *PlayerCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &PlayerCache{
		client: client,
		prefix: "player:",
		expiry: 30 * time.Minute, // 玩家数据过期时间
	}
}

// GetPlayer 获取玩家数据
func (c *PlayerCache) GetPlayer(playerID uint64) (*session.PlayerInfo, error) {
	ctx := context.Background()
	key := c.playerKey(playerID)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var player session.PlayerInfo
	if err := json.Unmarshal(data, &player); err != nil {
		return nil, err
	}

	return &player, nil
}

// SetPlayer 设置玩家数据
func (c *PlayerCache) SetPlayer(player *session.PlayerInfo) error {
	ctx := context.Background()
	key := c.playerKey(player.ID)

	data, err := json.Marshal(player)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.expiry).Err()
}

// DeletePlayer 删除玩家缓存
func (c *PlayerCache) DeletePlayer(playerID uint64) error {
	ctx := context.Background()
	key := c.playerKey(playerID)
	return c.client.Del(ctx, key).Err()
}

// UpdatePlayerPosition 更新玩家位置（只更新位置，不影响其他字段）
func (c *PlayerCache) UpdatePlayerPosition(playerID uint64, x, y int32) error {
	ctx := context.Background()
	key := c.playerKey(playerID)

	// 获取现有数据
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil // 缓存不存在，无需更新
		}
		return err
	}

	var player session.PlayerInfo
	if err := json.Unmarshal(data, &player); err != nil {
		return err
	}

	// 更新位置
	player.X = x
	player.Y = y

	// 重新设置
	data, err = json.Marshal(player)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.expiry).Err()
}

// playerKey 生成玩家缓存键
func (c *PlayerCache) playerKey(playerID uint64) string {
	return c.prefix + string(rune(playerID))
}

// Ping 测试 Redis 连接
func (c *PlayerCache) Ping() error {
	ctx := context.Background()
	return c.client.Ping(ctx).Err()
}

// Close 关闭 Redis 连接
func (c *PlayerCache) Close() error {
	return c.client.Close()
}
