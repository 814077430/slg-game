#!/bin/bash

# SLG Game Server 启动脚本
# 自动启动 MongoDB 和游戏服务器

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MONGO_BIN="/tmp/mongodb-linux-x86_64-ubuntu2204-7.0.5/bin"
DATA_DIR="/data/db"
LOG_DIR="$SCRIPT_DIR/logs"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "╔════════════════════════════════════════════════════════╗"
echo "║          SLG Game Server - Startup Script              ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

# 创建日志目录
mkdir -p "$LOG_DIR"

# 检查 MongoDB 是否已在运行
if pgrep -x "mongod" > /dev/null; then
    echo -e "${GREEN}✓${NC} MongoDB 已在运行"
else
    echo -e "${YELLOW}⟳${NC} 启动 MongoDB..."
    
    # 检查 MongoDB 二进制文件
    if [ ! -f "$MONGO_BIN/mongod" ]; then
        echo -e "${RED}✗${NC} MongoDB 未找到：$MONGO_BIN/mongod"
        echo "请先下载并解压 MongoDB"
        exit 1
    fi
    
    # 检查数据目录
    if [ ! -d "$DATA_DIR" ]; then
        echo -e "${YELLOW}⟳${NC} 创建数据目录：$DATA_DIR"
        sudo mkdir -p "$DATA_DIR"
        sudo chown $(whoami):$(whoami) "$DATA_DIR"
    fi
    
    # 启动 MongoDB
    $MONGO_BIN/mongod --dbpath "$DATA_DIR" --logpath "$LOG_DIR/mongod.log" --fork --bind_ip localhost
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} MongoDB 启动成功"
        sleep 2
    else
        echo -e "${RED}✗${NC} MongoDB 启动失败"
        exit 1
    fi
fi

# 检查 MongoDB 端口
if ! nc -z localhost 27017 2>/dev/null; then
    echo -e "${RED}✗${NC} MongoDB 端口 27017 未监听"
    exit 1
fi

echo ""
echo -e "${YELLOW}⟳${NC} 启动 SLG 游戏服务器..."

# 停止旧的游戏服务器进程
if pgrep -f "slg-server" > /dev/null; then
    echo -e "${YELLOW}⟳${NC} 停止旧的游戏服务器进程..."
    pkill -f "slg-server"
    sleep 2
fi

# 启动游戏服务器
cd "$SCRIPT_DIR"
./slg-server > "$LOG_DIR/server.log" 2>&1 &
SERVER_PID=$!

# 等待服务器启动
sleep 3

# 检查服务器是否运行
if ps -p $SERVER_PID > /dev/null; then
    echo -e "${GREEN}✓${NC} 游戏服务器启动成功 (PID: $SERVER_PID)"
    echo ""
    echo "═══════════════════════════════════════════════════════"
    echo "服务器状态:"
    echo "  - 监听地址：localhost:8080"
    echo "  - 数据库：MongoDB (localhost:27017)"
    echo "  - 日志文件：$LOG_DIR/server.log"
    echo ""
    echo "停止服务器：pkill -f slg-server"
    echo "查看日志：tail -f $LOG_DIR/server.log"
    echo "═══════════════════════════════════════════════════════"
else
    echo -e "${RED}✗${NC} 游戏服务器启动失败"
    echo "查看日志：$LOG_DIR/server.log"
    exit 1
fi
