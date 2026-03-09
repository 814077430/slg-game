# SLG Game Server Framework

这是一个基于Go语言的SLG（策略类游戏）服务器基础框架，具有以下特性：

- **协议**: 使用Protocol Buffers进行客户端-服务器通信
- **数据库**: 集成MongoDB作为数据存储
- **配置**: JSON格式的配置文件
- **架构**: 清晰的C2S（客户端到服务器）和S2C（服务器到客户端）逻辑分离

## 项目结构

```
slg-game/
├── main.go                 # 主程序入口
├── config/                 # 配置文件和加载逻辑
│   ├── game.json          # 游戏配置
│   └── config.go          # 配置加载
├── proto/                  # Protocol Buffers定义
│   └── messages.proto     # 消息协议定义
├── database/               # 数据库封装
│   └── database.go        # MongoDB连接管理
├── network/                # 网络层
│   ├── packet.go          # 数据包编解码
│   └── connection.go      # 连接管理
├── game/                   # 游戏逻辑
│   ├── server.go          # 游戏服务器主逻辑
│   ├── session.go         # 玩家会话管理
│   └── router.go          # 消息路由和处理器
├── go.mod                  # Go模块定义
├── Makefile                # 构建脚本
└── README.md               # 说明文档
```

## 快速开始

### 1. 安装依赖

```bash
# 安装Protocol Buffers编译器
# Ubuntu/Debian: sudo apt-get install protobuf-compiler
# macOS: brew install protobuf

# 安装Go protobuf插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 安装项目依赖
make deps
```

### 2. 启动MongoDB

确保MongoDB服务正在运行：

```bash
# 本地启动MongoDB
mongod --dbpath /path/to/data/directory
```

### 3. 构建和运行

```bash
# 构建项目（自动包含protobuf生成）
make build

# 或直接运行
make run
```

## 开发指南

### 添加新的C2S消息

1. 在 `proto/messages.proto` 中定义新的请求和响应消息
2. 在 `game/router.go` 中添加新的消息ID常量
3. 在 `registerHandlers()` 中注册新的处理器函数
4. 实现处理器函数，处理业务逻辑

### 数据库操作

在处理器函数中，可以通过 `session.db` 访问MongoDB：

```go
collection := session.db.GetCollection("players")
// 执行数据库操作
```

### 配置文件

修改 `config/game.json` 来调整服务器设置：

- `server.addr`: 服务器监听地址
- `database.url`: MongoDB连接URL
- `game.tick_interval`: 游戏主循环间隔（毫秒）

## 框架特点

- **高性能**: 基于Go的goroutine处理并发连接
- **可扩展**: 模块化设计，易于添加新功能
- **安全**: 输入验证和错误处理
- **易维护**: 清晰的代码结构和注释

## TODO

- [ ] 添加玩家数据模型
- [ ] 实现完整的认证系统
- [ ] 添加游戏世界管理
- [ ] 实现定时任务系统
- [ ] 添加日志系统
- [ ] 添加监控和指标

这个框架为你提供了一个坚实的基础，你只需要在 `game/router.go` 中实现具体的C2S和S2C逻辑即可。