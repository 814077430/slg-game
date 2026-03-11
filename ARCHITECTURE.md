# SLG Game Server Architecture

## 系统架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              CLIENT LAYER                                │
│                          (Game Client / Web)                             │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │ TCP Connection
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           NETWORK LAYER                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐      │
│  │   Connection     │  │     Packet       │  │   Connection     │      │
│  │   Manager        │  │   Encode/Decode  │  │   Pool           │      │
│  │                  │  │   (JSON)         │  │                  │      │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘      │
│                          connection.go, packet.go                        │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           SERVER LAYER                                   │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                        GameServer                               │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐    │    │
│  │  │  Message    │  │   Player    │  │     Game Loop       │    │    │
│  │  │   Router    │  │   Session   │  │   (Tick System)     │    │    │
│  │  │             │  │             │  │                     │    │    │
│  │  │ • Login     │  │ • State     │  │ • Resource Collect  │    │    │
│  │  │ • Register  │  │ • Cleanup   │  │ • Building Complete │    │    │
│  │  │ • Move      │  │ • Auth      │  │ • Tech Complete     │    │    │
│  │  │ • Build     │  │             │  │ • Army Movement     │    │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘    │    │
│  │     router.go         session.go      game_loop.go, server.go │    │
│  └────────────────────────────────────────────────────────────────┘    │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          GAME LOGIC LAYER                                │
│                                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │   World      │  │   Resource   │  │   Building   │                  │
│  │   Manager    │  │   Manager    │  │   Manager    │                  │
│  │              │  │              │  │              │                  │
│  │ • Map Tiles  │  │ • Get/Set    │  │ • Create     │                  │
│  │ • Claim      │  │ • Add/Deduct │  │ • Upgrade    │                  │
│  │ • Generate   │  │ • Collect    │  │ • Complete   │                  │
│  │ • Tick       │  │ • Check      │  │ • Cancel     │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
│     world.go       resources_manager.go    buildings.go                │
│                                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │    Army      │  │  Alliance    │  │  Technology  │                  │
│  │   Manager    │  │   Manager    │  │   Manager    │                  │
│  │              │  │              │  │              │                  │
│  │ • Attack     │  │ • Create     │  │ • Research   │                  │
│  │ • Battle     │  │ • Join       │  │ • Complete   │                  │
│  │ • Calculate  │  │ • Leave      │  │ • Get Config │                  │
│  │ • Loot       │  │ • Manage     │  │              │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
│    army_manager.go   alliance_manager.go   technology.go               │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          DATA ACCESS LAYER                               │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                      MongoDB Database                             │  │
│  │                                                                   │  │
│  │  Collections:                                                     │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │  │
│  │  │ players  │  │alliances │  │world_    │  │ battle_logs      │ │  │
│  │  │          │  │          │  │tiles     │  │                  │ │  │
│  │  │ • Player │  │ • Guild  │  │ • Map    │  │ • Combat Records │ │  │
│  │  │   Data   │  │   Data   │  │   Data   │  │                  │ │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘ │  │
│  │                                                                   │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐                        │  │
│  │  │research_ │  │technology│  │  (More   │                        │  │
│  │  │queue     │  |_config   │  │  tables) │                        │  │
│  │  └──────────┘  └──────────┘  └──────────┘                        │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                      database.go, models.go                              │
└─────────────────────────────────────────────────────────────────────────┘
```

## 模块依赖关系

```
main.go
  │
  ├─► config/           (配置加载)
  │
  ├─► database/         (数据库连接)
  │     └─► MongoDB
  │
  ├─► world/            (世界地图 - 独立线程)
  │
  ├─► battle/           (战斗系统 - 独立线程)
  │
  ├─► chat/             (聊天系统 - 独立线程)
  │
  └─► game/
        │
        ├─► core/
        │     │
        │     ├─► GameServer
        │     │     │
        │     │     ├─► MessageRouter
        │     │     │     └─► 消息处理 (Login/Register/Move/Build)
        │     │     │
        │     │     ├─► PlayerSession
        │     │     │     └─► 玩家状态管理
        │     │     │
        │     │     └─► GameLoop
        │     │           └─► Tick 系统
        │     │
        │     └─► PlayerManager
        │
        ├─► city/             (建筑系统)
        │
        ├─► resource/         (资源系统)
        │
        ├─► alliance/         (联盟系统)
        │
        └─► tech/             (科技系统)
```

## 数据流图

### 登录流程

```
Client                    Server                    Database
  │                         │                          │
  │──LoginRequest─────────►│                          │
  │                         │──Find Player───────────►│
  │                         │◄────Player Data─────────│
  │                         │                          │
  │                         │  [Verify Password]       │
  │                         │                          │
  │◄──LoginResponse────────│                          │
  │   (Success + Data)      │                          │
```

### 战斗流程

```
Client                    Server                    Database
  │                         │                          │
  │──AttackRequest────────►│                          │
  │                         │──Get Attacker Troops───►│
  │                         │──Get Defender Troops───►│
  │                         │                          │
  │                         │  [Calculate Power]       │
  │                         │  [Determine Winner]      │
  │                         │  [Calculate Losses]      │
  │                         │  [Calculate Loot]        │
  │                         │                          │
  │                         │──Update Troops─────────►│
  │                         │──Transfer Loot─────────►│
  │                         │──Save Battle Log───────►│
  │                         │                          │
  │◄──AttackResponse───────│                          │
  │   (Battle Result)       │                          │
```

### 游戏主循环 (Tick)

```
GameLoop
  │
  ├─► Every Tick (e.g., 1000ms)
  │     │
  │     ├─► processBuildingCompletion()
  │     ├─► processTechnologyCompletion()
  │     ├─► processArmyMovement()
  │     └─► world.Tick()
  │
  └─► ResourceCollector (Every 60s)
        │
        └─► For each player:
              └─► Add resources based on buildings
```

## 核心数据结构

### Player (玩家)

```go
type Player struct {
    PlayerID     uint64
    Username     string
    PasswordHash string
    Email        string
    Level        int32
    Experience   int64
    Gold         int64
    Wood         int64
    Food         int64
    X, Y         int32          // 坐标
    Buildings    []Building
    Troops       []Troop
    AllianceID   uint64
    AllianceRole string
}
```

### Message (网络消息)

```go
type Packet struct {
    MsgID uint32  // 消息 ID
    Data  []byte  // JSON 编码的数据
}

// Message IDs
const (
    MsgID_C2S_LoginRequest    = 1001
    MsgID_C2S_RegisterRequest = 1002
    MsgID_C2S_MoveRequest     = 1003
    MsgID_C2S_BuildRequest    = 1004
    MsgID_S2C_LoginResponse   = 2001
    MsgID_S2C_RegisterResponse = 2002
    MsgID_S2C_MoveResponse    = 2003
    MsgID_S2C_BuildResponse   = 2004
)
```

### BattleResult (战斗结果)

```go
type BattleResult struct {
    AttackerID      uint64
    DefenderID      uint64
    AttackerWon     bool
    AttackerLosses  map[string]int32
    DefenderLosses  map[string]int32
    LootedResources map[string]int64
    BattleTime      time.Time
}
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| 数据库 | MongoDB |
| 网络 | TCP (自定义协议) |
| 序列化 | JSON |
| 并发 | Goroutine + Channel |
| 数据持久化 | MongoDB Driver |

## 扩展点

### 可添加的功能

1. **任务系统** - 每日任务、成就系统
2. **邮件系统** - 站内信、系统通知
3. **排行榜** - 战力/资源/联盟排名
4. **聊天系统** - 世界频道/联盟频道
5. **交易系统** - 玩家间资源交易
6. **PVP 系统** - 竞技场、实时对战
7. **活动系统** - 限时活动、节日活动

### 性能优化方向

1. **Redis 缓存** - 在线玩家数据、会话缓存
2. **消息队列** - 异步处理战斗、邮件等
3. **分服架构** - 多服负载均衡
4. **数据库优化** - 索引、分片
5. **连接池** - 数据库连接复用

## 配置示例

```json
{
  "server": {
    "addr": "0.0.0.0:8080",
    "max_connections": 1000,
    "read_timeout": 30,
    "write_timeout": 30
  },
  "database": {
    "url": "mongodb://localhost:27017",
    "database_name": "slg_game",
    "max_pool_size": 20
  },
  "game": {
    "tick_interval": 1000,
    "max_players": 10000
  }
}
```

## 启动流程

```
1. main.go
   │
   ├─► Load Config (config/game.json)
   │
   ├─► Init MongoDB Connection
   │
   ├─► Create GameServer
   │     │
   │     ├─► Initialize Independent Thread Modules:
   │     │     ├─► world.NewWorld()      → Start World Loop
   │     │     ├─► battle.NewBattleManager() → Start Battle Loop
   │     │     └─► chat.NewChatManager()     → Start Chat Loop
   │     │
   │     ├─► Create MessageRouter
   │     ├─► Create PlayerManager
   │     └─► Start GameLoop
   │
   └─► Start TCP Listener
         │
         └─► Accept Connections
               │
               └─► HandleClient (goroutine per client)
```

---

*Last updated: 2026-03-12*
