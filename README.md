# SLG Game Server

一个基于 Go 语言的 **SLG（策略类游戏）服务器框架**，提供完整的游戏核心功能。

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![MongoDB](https://img.shields.io/badge/MongoDB-4.4+-47A248?style=flat&logo=mongodb)](https://www.mongodb.com/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 🎮 功能特性

### 核心功能 ✅

| 模块 | 功能 | 状态 |
|------|------|------|
| **用户系统** | 注册/登录/认证 | ✅ |
| **玩家数据** | 持久化存储/会话管理 | ✅ |
| **移动系统** | 坐标更新/地图导航 | ✅ |
| **建造系统** | 建筑创建/升级/取消 | ✅ |
| **资源系统** | 资源生产/交易/收集 | ✅ |
| **游戏循环** | Tick 系统/定时任务 | ✅ |
| **战斗系统** | 攻击/战力计算/掠夺 | ✅ |
| **联盟系统** | 创建/加入/管理/角色 | ✅ |
| **科技系统** | 研究/升级 | ✅ |
| **世界地图** | 地块/所有权/资源点 | ✅ |

### 技术特性

- 🚀 **高性能** - Goroutine 并发处理，单服支持 1000+ 在线
- 📦 **模块化** - 清晰的分层架构，易于扩展
- 💾 **数据持久化** - MongoDB 存储，支持备份和恢复
- 🔐 **安全认证** - SHA256 密码哈希，会话管理
- ⏰ **定时任务** - Tick 系统驱动，支持资源生产、建筑完成等
- 🌐 **TCP 协议** - 自定义二进制协议 + JSON 序列化

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────┐
│                    Client Layer                          │
│                  (Game Client / Web)                     │
└────────────────────────┬────────────────────────────────┘
                         │ TCP Connection
                         ▼
┌─────────────────────────────────────────────────────────┐
│                   Network Layer                          │
│   Connection Manager | Packet Encode/Decode (JSON)      │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                    Server Layer                          │
│   GameServer | MessageRouter | PlayerSession | GameLoop │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                  Game Logic Layer                        │
│  World | Resource | Building | Army | Alliance | Tech   │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                 Data Access Layer                        │
│                    MongoDB                               │
│  players | alliances | world_tiles | battle_logs | ...  │
└─────────────────────────────────────────────────────────┘
```

详细架构文档：[ARCHITECTURE.md](ARCHITECTURE.md)

## 📁 项目结构

```
slg-game/
├── main.go                     # 程序入口
├── go.mod / go.sum             # Go 依赖管理
├── Makefile                    # 构建脚本
├── README.md                   # 项目说明
├── ARCHITECTURE.md             # 架构文档
│
├── config/                     # 配置模块
│   ├── config.go               # 配置加载
│   └── game.json               # 游戏配置
│
├── database/                   # 数据库层
│   ├── database.go             # MongoDB 连接
│   └── models.go               # 数据模型定义
│
├── network/                    # 网络层
│   ├── packet.go               # 数据包编解码
│   └── connection.go           # 连接管理
│
├── world/                      # 世界地图模块（独立线程）
│   └── world.go                # 世界地图管理
│
├── battle/                     # 战斗系统模块（独立线程）
│   ├── battle_manager.go       # 战斗管理
│   └── army_manager.go         # 军队管理
│
├── chat/                       # 聊天系统模块（独立线程）
│   └── chat_manager.go         # 聊天管理
│
├── game/                       # 游戏核心逻辑层
│   ├── core/                   # 核心服务
│   │   ├── server.go           # 游戏服务器
│   │   ├── router.go           # 消息路由
│   │   ├── session.go          # 玩家会话
│   │   ├── game_loop.go        # 游戏主循环
│   │   └── player_manager.go   # 玩家管理
│   ├── city/                   # 建筑系统
│   │   └── buildings.go        # 建筑管理
│   ├── resource/               # 资源系统
│   │   └── resources_manager.go # 资源管理
│   ├── alliance/               # 联盟系统
│   │   └── alliance_manager.go # 联盟管理
│   └── tech/                   # 科技系统
│       └── technology.go       # 科技管理
│
└── proto/                      # 协议定义
    └── messages.pb.go          # 消息数据结构
```

## 🚀 快速开始

### 环境要求

- Go 1.21+
- MongoDB 4.4+
- Git

### 1. 安装依赖

```bash
# 克隆项目
git clone https://gitee.com/liang-bowei/slg-game.git
cd slg-game

# 下载依赖
go mod download
```

### 2. 启动 MongoDB

```bash
# 使用 Docker（推荐）
docker run -d --name mongodb -p 27017:27017 mongo:4.4

# 或本地安装
mongod --dbpath /data/db
```

### 3. 配置服务器

编辑 `config/game.json`：

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

### 4. 编译运行

```bash
# 编译
make build

# 或直接运行
make run

# 手动运行
go build -o slg-server .
./slg-server
```

### 5. 测试连接

使用 TCP 客户端连接测试：

```bash
# 使用 telnet 测试
telnet localhost 8080

# 发送登录请求（JSON 格式）
{"msg_id": 1001, "data": {"username": "test", "password": "123456"}}
```

## 📡 消息协议

### 协议格式

```
┌─────────────┬─────────────┬─────────────────────────┐
│   MsgID     │   MsgLen    │        Data (JSON)      │
│  (4 bytes)  │  (4 bytes)  │    (variable length)    │
└─────────────┴─────────────┴─────────────────────────┘
```

### 消息类型

| 方向 | 消息 ID | 名称 | 说明 |
|------|--------|------|------|
| C2S | 1001 | LoginRequest | 登录请求 |
| C2S | 1002 | RegisterRequest | 注册请求 |
| C2S | 1003 | MoveRequest | 移动请求 |
| C2S | 1004 | BuildRequest | 建造请求 |
| S2C | 2001 | LoginResponse | 登录响应 |
| S2C | 2002 | RegisterResponse | 注册响应 |
| S2C | 2003 | MoveResponse | 移动响应 |
| S2C | 2004 | BuildResponse | 建造响应 |

### 请求示例

#### 登录请求

```json
{
  "username": "player1",
  "password": "password123"
}
```

#### 登录响应

```json
{
  "success": true,
  "message": "Login successful",
  "player_id": 10001,
  "player_data": {
    "player_id": 10001,
    "username": "player1",
    "level": 1,
    "gold": 1000,
    "wood": 1000,
    "food": 1000,
    "x": 0,
    "y": 0
  }
}
```

#### 移动请求

```json
{
  "x": 100,
  "y": 200
}
```

#### 建造请求

```json
{
  "building_type": "farm",
  "x": 50,
  "y": 50
}
```

## 🎯 核心系统说明

### 1. 用户系统

- 用户名唯一性检查
- 密码 SHA256 哈希存储
- 自动分配玩家 ID（从 10001 开始）
- 登录状态管理

### 2. 资源系统

- 基础资源：金币、木材、粮食
- 定时生产（每分钟）
- 建筑产量加成
- 资源检查（建造/升级前）

### 3. 建筑系统

- 建筑类型：城堡、兵营、农场、伐木场、矿场等
- 建造时间机制
- 升级系统
- 位置占用检查

### 4. 战斗系统

- 战力计算（兵种 + 等级）
- 胜负判定（±20% 随机因素）
- 损失计算（胜方 10%，败方 50%）
- 资源掠夺（30%）
- 战斗记录保存

### 5. 联盟系统

- 创建联盟（需不在联盟中）
- 加入/离开联盟
- 成员角色：盟主、官员、成员
- 权限管理
- 联盟解散

### 6. 游戏主循环

- Tick 间隔：1000ms（可配置）
- 每 Tick 处理：
  - 建筑建造完成检查
  - 科技研究完成检查
  - 军队移动处理
  - 世界状态更新
- 资源收集器：每 60 秒自动收集资源

## 🔧 开发指南

### 添加新消息类型

1. 在 `game/router.go` 添加消息 ID 常量：

```go
const (
    MsgID_C2S_NewRequest = 1005
    MsgID_S2C_NewResponse = 2005
)
```

2. 在 `registerHandlers()` 注册处理器：

```go
mr.handlers[MsgID_C2S_NewRequest] = mr.handleNewRequest
```

3. 实现处理器函数：

```go
func (mr *MessageRouter) handleNewRequest(session *PlayerSession, data []byte) *network.Packet {
    // 处理逻辑
    return response
}
```

### 添加新建筑类型

在 `game/buildings.go` 的 `BuildingTemplates` 中添加：

```go
BuildingTypeAcademy: {
    1: {
        Level: 1,
        BuildTime: 300,
        ResourceCost: map[string]int32{"gold": 500, "wood": 300},
        Stats: map[string]int32{"can_research": 1},
    },
}
```

### 添加新资源类型

在 `game/resources_manager.go` 添加常量：

```go
const (
    ResourceDiamond ResourceType = "diamond"  // 钻石
)
```

## 📊 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 单服在线 | 1000+ | 取决于服务器配置 |
| Tick 间隔 | 1000ms | 可配置 |
| 数据库连接池 | 20 | 可配置 |
| 最大连接超时 | 30s | 读写超时 |

## 🛠️ 常用命令

```bash
# 编译
make build

# 运行
make run

# 清理
make clean

# 安装依赖
make deps

# 查看依赖
go list -m all

# 格式化代码
go fmt ./...

# 运行测试
go test ./...
```

## 📝 待开发功能

- [ ] 任务系统（每日任务/成就）
- [ ] 邮件系统（站内信）
- [ ] 排行榜（战力/资源/联盟）
- [ ] 聊天系统（世界/联盟频道）
- [ ] 交易系统（玩家间交易）
- [ ] PVP 竞技场
- [ ] 活动系统（限时活动）
- [ ] GM 命令工具
- [ ] Web 管理后台

## 🚀 性能优化方向

- [ ] Redis 缓存（在线玩家数据）
- [ ] 消息队列（异步处理）
- [ ] 数据库索引优化
- [ ] 连接池优化
- [ ] 分服架构支持

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 联系方式

- 项目地址：https://gitee.com/liang-bowei/slg-game
- 问题反馈：提交 Issue

---

**Happy Coding! 🎮**
