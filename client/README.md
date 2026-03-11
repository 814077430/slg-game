# SLG Game Client - Test Client

基于 Go 语言的 SLG 游戏服务器测试客户端。

## 功能特性

- ✅ TCP 连接管理
- ✅ 自动编解码（JSON）
- ✅ 消息收发
- ✅ 测试用例：注册/登录/移动/建造
- ✅ 批量测试支持
- ✅ 响应解析和打印

## 编译

```bash
cd client
go build -o slg-client .
```

## 使用方法

### 1. 运行所有测试

```bash
./slg-client -server localhost:8080 -test all
```

### 2. 单独测试

#### 注册测试

```bash
./slg-client -server localhost:8080 -test register
```

#### 登录测试

```bash
./slg-client -server localhost:8080 -test login
```

#### 移动测试

```bash
./slg-client -server localhost:8080 -test move
```

#### 建造测试

```bash
./slg-client -server localhost:8080 -test build
```

### 3. 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-server` | 服务器地址 | localhost:8080 |
| `-test` | 测试模式 | all |

测试模式：
- `all` - 运行所有测试
- `login` - 仅登录测试
- `register` - 仅注册测试
- `move` - 仅移动测试
- `build` - 仅建造测试

## 测试流程

### 完整测试流程（-test all）

```
1. 连接到服务器
   ↓
2. 注册新账号（自动生成唯一用户名）
   ↓
3. 登录账号
   ↓
4. 测试移动功能
   ↓
5. 测试建造功能
   ↓
6. 批量移动测试（5 个位置）
   ↓
7. 断开连接
```

### 示例输出

```
╔════════════════════════════════════════════════════════╗
║          SLG Game Server Test Client                   ║
╚════════════════════════════════════════════════════════╝
Server: localhost:8080
Test Mode: all

=== Running All Tests ===

[TEST 1] Register new account
[TEST] Register response received (MsgID: 2002)

=== Packet (MsgID: 2002) ===
Data:
{
  "success": true,
  "message": "Registration successful",
  "player_id": 10001
}
=========================

✓ Register successful! Player ID: 10001

[TEST 2] Login
[TEST] Login response received (MsgID: 2001)

=== Packet (MsgID: 2001) ===
Data:
{
  "success": true,
  "message": "Login successful",
  "player_id": 10001,
  "player_data": {
    "player_id": 10001,
    "username": "testuser_1234567890",
    "level": 1,
    "gold": 1000,
    "wood": 1000,
    "food": 1000
  }
}
=========================

✓ Login successful! Player ID: 10001

[TEST 3] Move player
[TEST] Move response received (MsgID: 2003)

=== Packet (MsgID: 2003) ===
Data:
{
  "success": true,
  "message": "Move successful",
  "x": 100,
  "y": 200
}
=========================

✓ Move successful! Position: (100, 200)

[TEST 4] Build structure
[TEST] Build response received (MsgID: 2004)

=== Packet (MsgID: 2004) ===
Data:
{
  "success": true,
  "message": "Build request received",
  "building": {
    "building_type": "farm",
    "x": 50,
    "y": 50,
    "level": 1
  }
}
=========================

✓ Build successful! Building: farm at (50, 50)

[TEST 5] Multiple moves
  Move 1/5 to (10, 10)... ✓ Success
  Move 2/5 to (20, 20)... ✓ Success
  Move 3/5 to (30, 30)... ✓ Success
  Move 4/5 to (40, 40)... ✓ Success
  Move 5/5 to (50, 50)... ✓ Success

=== All Tests Completed ===
```

## 代码结构

```
client/
├── main.go              # 主程序和测试用例
├── network.go           # TCP 客户端网络层
├── messages.go          # 消息定义和解析
└── README.md            # 本文档
```

## 核心 API

### 创建客户端

```go
client := NewTestClient("localhost:8080")
```

### 连接服务器

```go
err := client.Connect()
```

### 注册账号

```go
resp, err := client.Register("username", "password", "email@example.com")
if resp.Success {
    fmt.Printf("Player ID: %d\n", resp.PlayerId)
}
```

### 登录

```go
resp, err := client.Login("username", "password")
if resp.Success {
    fmt.Printf("Logged in as: %s\n", resp.PlayerData.Username)
}
```

### 移动

```go
resp, err := client.Move(100, 200)
```

### 建造

```go
resp, err := client.Build("farm", 50, 50)
```

### 关闭连接

```go
client.Close()
```

## 消息协议

### 客户端 → 服务器（C2S）

| 消息 ID | 名称 | 说明 |
|--------|------|------|
| 1001 | LoginRequest | 登录请求 |
| 1002 | RegisterRequest | 注册请求 |
| 1003 | MoveRequest | 移动请求 |
| 1004 | BuildRequest | 建造请求 |

### 服务器 → 客户端（S2C）

| 消息 ID | 名称 | 说明 |
|--------|------|------|
| 2001 | LoginResponse | 登录响应 |
| 2002 | RegisterResponse | 注册响应 |
| 2003 | MoveResponse | 移动响应 |
| 2004 | BuildResponse | 建造响应 |

## 自定义测试

可以基于 `TestClient` 创建自己的测试：

```go
package main

import "fmt"

func main() {
    client := NewTestClient("localhost:8080")
    client.Connect()
    defer client.Close()

    // 注册
    client.Register("myuser", "pass123", "my@example.com")

    // 登录
    client.Login("myuser", "pass123")

    // 移动
    client.Move(100, 200)

    // 建造
    client.Build("barracks", 50, 50)

    fmt.Println("Custom test completed!")
}
```

## 故障排查

### 连接失败

```
Failed to connect: dial tcp 127.0.0.1:8080: connect: connection refused
```

**解决：** 确保服务器已启动
```bash
cd ..
./slg-server
```

### 登录失败

```
Login failed: Invalid username or password
```

**解决：** 先注册账号，或使用已存在的账号

### 超时错误

```
timeout
```

**解决：** 检查服务器是否正常运行，或增加超时时间

## 扩展测试

可以添加更多测试功能：

- [ ] 战斗测试
- [ ] 联盟测试
- [ ] 资源收集测试
- [ ] 科技研究测试
- [ ] 压力测试（多客户端并发）

---

**Happy Testing! 🎮**
