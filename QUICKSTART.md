# 🚀 快速使用指南

## 三种启动方式

### 方式 1：使用 start.sh（自动化，推荐新手）

```bash
sudo ./start.sh
```

**特点：**
- ✅ 自动检查依赖（libpcap、Go）
- ✅ 自动下载 Go 依赖
- ✅ 自动编译
- ✅ 自动启动
- ✅ 处理 sudo 环境下的 PATH 问题

**适用场景：** 首次运行、生产部署

---

### 方式 2：使用 build.sh（开发调试）

```bash
./build.sh              # 编译（不需要 sudo）
sudo ./traffic-monitor  # 运行（需要 sudo）
```

**或者设置 CAP_NET_RAW 后无需 sudo：**
```bash
./build.sh
sudo setcap cap_net_raw+ep ./traffic-monitor
./traffic-monitor       # 无需 sudo
```

**特点：**
- ✅ 不需要 sudo 编译
- ✅ 适合开发调试
- ✅ 可以分离编译和运行步骤

**适用场景：** 开发调试、频繁修改代码

---

### 方式 3：使用 Makefile（专业用户）

```bash
make install-deps       # 安装依赖
make build              # 编译
sudo make run           # 运行
```

**其他命令：**
```bash
make clean              # 清理编译产物和数据库
make setcap             # 设置 CAP_NET_RAW
make fmt                # 格式化代码
make lint               # 代码检查
```

**适用场景：** 专业开发、CI/CD 集成

---

## 常见问题解决

### 问题 1：sudo 找不到 go 命令

**现象：**
```
❌ Go is not installed or not in PATH!
```

**解决方案 A：** 使用 build.sh 分离编译和运行
```bash
./build.sh              # 用当前用户编译
sudo ./traffic-monitor  # 用 root 运行
```

**解决方案 B：** 手动编译
```bash
go build -o traffic-monitor main.go
sudo ./traffic-monitor
```

**解决方案 C：** 将 Go 添加到 root 的 PATH
```bash
# 编辑 /etc/environment 或 /root/.bashrc
export PATH=$PATH:/usr/local/go/bin
```

---

### 问题 2：权限不足

**现象：**
```
Error: failed to open interface eth0: Operation not permitted
```

**解决方案：** 使用 sudo 或设置 CAP_NET_RAW
```bash
# 方式 1：使用 sudo
sudo ./traffic-monitor

# 方式 2：设置能力
sudo setcap cap_net_raw+ep ./traffic-monitor
./traffic-monitor
```

---

### 问题 3：端口被占用

**现象：**
```
Error: listen tcp :8080: bind: address already in use
```

**解决方案：** 修改端口或停止占用进程
```bash
# 查找占用进程
sudo lsof -i :8080

# 停止进程
sudo kill <PID>

# 或修改 main.go 中的端口（第 596 行）
Addr: ":9090",  # 改为其他端口
```

---

### 问题 4：libpcap 未安装

**现象：**
```
Error: pcap.h: No such file or directory
```

**解决方案：**
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y libpcap-dev

# CentOS/RHEL
sudo yum install -y libpcap-devel

# Arch Linux
sudo pacman -S libpcap
```

---

## 验证安装

### 1. 检查程序启动
启动后应该看到：
```
=== Linux Port Traffic Monitor Starting ===
Memory aggregator initialized
Database initialized with WAL mode
Monitoring X listening ports
Started packet capture on interface: eth0
Web server started on http://0.0.0.0:8080
```

### 2. 访问 Web 界面
打开浏览器访问：
```
http://localhost:8080
```

### 3. 检查数据采集
等待 1-2 分钟后，应该看到：
```
Persisted X traffic snapshots
```

---

## 推荐工作流

### 开发环境
```bash
# 1. 首次运行
./build.sh
sudo setcap cap_net_raw+ep ./traffic-monitor
./traffic-monitor

# 2. 修改代码后
./build.sh              # 重新编译
./traffic-monitor       # 运行（无需 sudo）
```

### 生产环境
```bash
# 1. 首次部署
sudo ./start.sh

# 2. 后续启动
sudo ./traffic-monitor

# 3. 配置为系统服务（推荐）
sudo systemctl enable traffic-monitor
sudo systemctl start traffic-monitor
```

---

## 性能监控

### 查看资源占用
```bash
# CPU 和内存
top -p $(pgrep traffic-monitor)

# 数据库大小
du -h traffic_monitor.db*

# 网络流量
iftop -i eth0
```

### 查看日志
```bash
# 实时日志
sudo ./traffic-monitor

# 后台运行并记录日志
sudo ./traffic-monitor >> /var/log/traffic-monitor.log 2>&1 &
```

---

## 停止程序

### 前台运行
按 `Ctrl+C` 停止

### 后台运行
```bash
# 查找进程
ps aux | grep traffic-monitor

# 停止进程
sudo kill <PID>

# 或使用 pkill
sudo pkill traffic-monitor
```

---

## 下一步

- 📖 阅读 [README.md](README.md) 了解详细功能
- 🔧 查看 [DEPLOYMENT_CHECKLIST.md](DEPLOYMENT_CHECKLIST.md) 进行生产部署
- 📊 访问 Web 界面开始监控流量

---

**需要帮助？** 查看 README.md 的故障排查章节
