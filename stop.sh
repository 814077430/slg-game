#!/bin/bash

# SLG Game Server 停止脚本

echo "╔════════════════════════════════════════════════════════╗"
echo "║          SLG Game Server - Shutdown Script             ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

# 停止游戏服务器
if pgrep -f "slg-server" > /dev/null; then
    echo "⟳ 停止游戏服务器..."
    pkill -f "slg-server"
    echo "✓ 游戏服务器已停止"
else
    echo "ℹ 游戏服务器未运行"
fi

# 询问是否停止 MongoDB
read -p "是否停止 MongoDB? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if pgrep -x "mongod" > /dev/null; then
        echo "⟳ 停止 MongoDB..."
        pkill -x "mongod"
        echo "✓ MongoDB 已停止"
    else
        echo "ℹ MongoDB 未运行"
    fi
fi

echo ""
echo "═══════════════════════════════════════════════════════"
echo "关闭完成"
echo "═══════════════════════════════════════════════════════"
