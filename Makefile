.PHONY: all build run clean install-deps help

# 默认目标
all: build

# 编译程序
build:
	@echo "🔨 Building traffic-monitor..."
	go build -o traffic-monitor main.go
	@echo "✅ Build complete: ./traffic-monitor"

# 编译并运行（需要 root 权限）
run: build
	@echo "🚀 Starting traffic-monitor (requires root)..."
	sudo ./traffic-monitor

# 安装依赖
install-deps:
	@echo "📦 Installing Go dependencies..."
	go mod download
	@echo "✅ Dependencies installed"

# 安装 libpcap（Ubuntu/Debian）
install-libpcap-debian:
	@echo "📦 Installing libpcap-dev..."
	sudo apt-get update
	sudo apt-get install -y libpcap-dev
	@echo "✅ libpcap-dev installed"

# 安装 libpcap（CentOS/RHEL）
install-libpcap-centos:
	@echo "📦 Installing libpcap-devel..."
	sudo yum install -y libpcap-devel
	@echo "✅ libpcap-devel installed"

# 赋予 CAP_NET_RAW 能力（避免 root 运行）
setcap: build
	@echo "🔐 Setting CAP_NET_RAW capability..."
	sudo setcap cap_net_raw+ep ./traffic-monitor
	@echo "✅ Capability set. You can now run without sudo."

# 清理编译产物
clean:
	@echo "🧹 Cleaning..."
	rm -f traffic-monitor
	rm -f traffic_monitor.db
	rm -f traffic_monitor.db-shm
	rm -f traffic_monitor.db-wal
	@echo "✅ Clean complete"

# 格式化代码
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✅ Format complete"

# 代码检查
lint:
	@echo "🔍 Running linter..."
	go vet ./...
	@echo "✅ Lint complete"

# 显示帮助信息
help:
	@echo "Linux Port Traffic Monitor - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build                    - Compile the program"
	@echo "  run                      - Build and run (requires root)"
	@echo "  install-deps             - Install Go dependencies"
	@echo "  install-libpcap-debian   - Install libpcap on Ubuntu/Debian"
	@echo "  install-libpcap-centos   - Install libpcap on CentOS/RHEL"
	@echo "  setcap                   - Set CAP_NET_RAW capability"
	@echo "  clean                    - Remove build artifacts and database"
	@echo "  fmt                      - Format Go code"
	@echo "  lint                     - Run code linter"
	@echo "  help                     - Show this help message"
