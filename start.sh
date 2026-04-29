#!/bin/bash

# Linux Port Traffic Monitor - 快速启动脚本

set -e

echo "🚀 Linux Port Traffic Monitor - Quick Start"
echo "=========================================="
echo ""

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then
    echo "⚠️  This script requires root privileges."
    echo "Please run with sudo: sudo ./start.sh"
    exit 1
fi

# 检查 libpcap 是否安装
if ! ldconfig -p | grep -q libpcap; then
    echo "❌ libpcap not found!"
    echo ""
    echo "Please install libpcap first:"
    echo "  Ubuntu/Debian: sudo apt-get install libpcap-dev"
    echo "  CentOS/RHEL:   sudo yum install libpcap-devel"
    exit 1
fi

# 检查 Go 是否安装（保留原用户的 PATH）
GO_CMD=""
if command -v go &> /dev/null; then
    GO_CMD="go"
elif [ -n "$SUDO_USER" ]; then
    # 尝试使用原用户的环境查找 go
    GO_CMD=$(su - "$SUDO_USER" -c "command -v go" 2>/dev/null || echo "")
fi

if [ -z "$GO_CMD" ]; then
    echo "❌ Go is not installed or not in PATH!"
    echo "Please install Go from https://golang.org/dl/"
    echo ""
    echo "If Go is already installed, try running directly:"
    echo "  go build -o traffic-monitor main.go"
    echo "  sudo ./traffic-monitor"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# 下载依赖（使用原用户身份）
if [ ! -d "vendor" ] && [ ! -f "go.sum" ]; then
    echo "📦 Downloading Go dependencies..."
    if [ -n "$SUDO_USER" ]; then
        su - "$SUDO_USER" -c "cd '$PWD' && $GO_CMD mod download"
    else
        $GO_CMD mod download
    fi
    echo "✅ Dependencies downloaded"
    echo ""
fi

# 编译程序（使用原用户身份）
if [ ! -f "traffic-monitor" ] || [ "main.go" -nt "traffic-monitor" ]; then
    echo "🔨 Building traffic-monitor..."
    if [ -n "$SUDO_USER" ]; then
        su - "$SUDO_USER" -c "cd '$PWD' && $GO_CMD build -o traffic-monitor main.go"
    else
        $GO_CMD build -o traffic-monitor main.go
    fi
    echo "✅ Build complete"
    echo ""
fi

# 启动程序
echo "🚀 Starting traffic-monitor..."
echo ""
echo "📊 Web interface will be available at: http://localhost:8080"
echo "Press Ctrl+C to stop"
echo ""
echo "=========================================="
echo ""

./traffic-monitor
