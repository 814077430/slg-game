# Game Modules 游戏模块

SLG 游戏服务器模块化架构说明。

## 📁 目录结构

```
game/
├── core/           # 基础模块 - 服务器核心功能
├── world/          # 世界模块 - 地图和地块管理
├── city/           # 城建模块 - 建筑系统
├── resource/       # 资源模块 - 资源管理
├── army/           # 军队模块 - 战斗系统
├── alliance/       # 联盟模块 - 联盟管理
├── tech/           # 科技模块 - 科技树
└── player/         # 玩家模块 - 玩家数据 (TODO)
```

## 🔧 模块说明

### core/ - 基础模块

**职责：** 服务器核心功能，消息处理，会话管理

| 文件 | 说明 |
|------|------|
| `server.go` | 游戏服务器主逻辑，模块初始化 |
| `session.go` | 玩家会话管理，登录状态 |
| `router.go` | 消息路由，C2S/S2C 处理 |
| `game_loop.go` | 游戏主循环，Tick 系统 |

**依赖：** 无（其他模块依赖 core）

---

### world/ - 世界模块

**职责：** 世界地图，地块管理，区域划分

| 文件 | 说明 |
|------|------|
| `world.go` | 世界地图管理 |
| `tile.go` | 地块数据结构（TODO） |
| `region.go` | 区域管理（TODO） |

**核心功能：**
- 地块生成和管理
- 地块所有权
- 资源点分布
- 世界状态 Tick 更新

---

### city/ - 城建模块

**职责：** 建筑系统，建造队列

| 文件 | 说明 |
|------|------|
| `buildings.go` | 建筑管理 |
| `building_types.go` | 建筑类型定义（TODO） |
| `construction.go` | 建造队列（TODO） |

**核心功能：**
- 建筑创建/升级
- 建造时间管理
- 建筑产量计算
- 建筑配置

---

### resource/ - 资源模块

**职责：** 资源管理，生产，交易

| 文件 | 说明 |
|------|------|
| `resource.go` | 资源管理 |
| `production.go` | 资源生产（TODO） |
| `trade.go` | 资源交易（TODO） |

**核心功能：**
- 资源获取/扣除
- 资源生产（定时）
- 资源容量限制
- 资源检查

---

### army/ - 军队模块

**职责：** 军队管理，战斗系统

| 文件 | 说明 |
|------|------|
| `army.go` | 军队管理 |
| `troop.go` | 兵种定义（TODO） |
| `battle.go` | 战斗系统（TODO） |
| `march.go` | 行军系统（TODO） |

**核心功能：**
- 军队创建/训练
- 战力计算
- 战斗逻辑
- 行军移动

---

### alliance/ - 联盟模块

**职责：** 联盟管理，成员，科技

| 文件 | 说明 |
|------|------|
| `alliance.go` | 联盟管理 |
| `member.go` | 成员管理（TODO） |
| `tech.go` | 联盟科技（TODO） |

**核心功能：**
- 联盟创建/解散
- 成员管理（加入/退出/踢出）
- 联盟角色（盟主/官员/成员）
- 联盟科技

---

### tech/ - 科技模块

**职责：** 科技树，研究队列

| 文件 | 说明 |
|------|------|
| `technology.go` | 科技管理 |
| `research.go` | 研究队列（TODO） |

**核心功能：**
- 科技树定义
- 科技研究
- 科技效果应用
- 研究队列管理

---

### player/ - 玩家模块 (TODO)

**职责：** 玩家数据，背包，任务

| 文件 | 说明 |
|------|------|
| `player.go` | 玩家数据管理 |
| `inventory.go` | 背包系统 |
| `quest.go` | 任务系统 |

**核心功能：**
- 玩家数据持久化
- 背包物品管理
- 任务接取/完成
- 成就系统

---

## 🔄 模块依赖关系

```
                    main.go
                      │
                      ▼
                   core/
                      │
         ┌────────────┼────────────┐
         │            │            │
         ▼            ▼            ▼
      world/       city/      resource/
         │            │            │
         └────────────┼────────────┘
                      │
         ┌────────────┼────────────┐
         │            │            │
         ▼            ▼            ▼
       army/     alliance/      tech/
                      │
                      ▼
                   player/
```

**依赖规则：**
- `core` 是基础模块，不依赖其他业务模块
- `world` 只依赖 `core`
- `city`, `resource` 依赖 `core` 和 `world`
- `army`, `alliance`, `tech` 可以依赖上层模块
- `player` 是最高层，可以依赖所有模块

---

## 📝 开发指南

### 添加新模块

1. 在 `game/` 下创建新目录
2. 设置 `package <模块名>`
3. 在 `core/server.go` 中初始化模块管理器
4. 在 `core/game_loop.go` 中添加 Tick 处理（如需要）

### 模块间通信

```go
// 通过 GameServer 获取其他模块
buildingMgr := gs.GetBuildingMgr()
resourceMgr := gs.GetResourceMgr()

// 模块间不要直接依赖，通过接口或事件通信
```

### 添加新消息类型

1. 在 `protocol/messages.proto` 定义消息
2. 重新生成 Go 代码：`protoc --go_out=. protocol/messages.proto`
3. 在 `core/router.go` 添加消息 ID 常量
4. 在 `registerHandlers()` 注册处理器
5. 实现处理函数

---

## 🎯 待实现功能

| 模块 | 功能 | 优先级 |
|------|------|--------|
| city | 建筑类型配置 | 🔴 高 |
| city | 建造队列 | 🔴 高 |
| resource | 资源生产 | 🔴 高 |
| army | 兵种定义 | 🟡 中 |
| army | 战斗系统 | 🟡 中 |
| alliance | 成员管理 | 🟡 中 |
| tech | 科技树 | 🟢 低 |
| player | 背包系统 | 🟢 低 |
| player | 任务系统 | 🟢 低 |

---

*Last updated: 2026-03-11*
