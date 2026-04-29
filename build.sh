#!/bin/bash

# Linux Port Traffic Monitor - 简化启动脚本（不检查 root）
# 适用于：已设置 CAP_NET_RAW 或开发环境

set -e

echo "🚀 Linux Port Traffic Monitor - Build & Run"
echo "============================================"
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed!"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

echo "✅ Go found: $(go version)"
echo ""

# 下载依赖
if [ ! -f "go.sum" ]; then
    echo "📦 Downloading Go dependencies..."
    go mod download
    echo "✅ Dependencies downloaded"
    echo ""
fi

# 编译程序
if [ ! -f "traffic-monitor" ] || [ "main.go" -nt "traffic-monitor" ]; then
    echo "🔨 Building traffic-monitor..."
    go build -o traffic-monitor main.go
    echo "✅ Build complete"
    echo ""
fi

# 检查权限
if [ "$EUID" -ne 0 ]; then
    echo "⚠️  Running without root privileges."
    echo ""
    echo "If you get 'Operation not permitted' error, you need to either:"
    echo "  1. Run with sudo: sudo ./traffic-monitor"
    echo "  2. Set CAP_NET_RAW: sudo setcap cap_net_raw+ep ./traffic-monitor"
    echo ""
    echo "Attempting to start anyway..."
    echo ""
fi

# 启动程序
echo "🚀 Starting traffic-monitor..."
echo ""
echo "📊 Web interface will be available at: http://localhost:8080"
echo "Press Ctrl+C to stop"
echo ""
echo "============================================"
echo ""

./traffic-monitor
