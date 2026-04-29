# Linux 流量监控系统

[English](README.md) | 简体中文

Linux 实时网络流量监控系统，支持双监控模式：端口级和主机级流量分析。

## 功能特性

### 双监控模式
- **端口监控**：追踪特定监听端口的流量
- **主机监控**：按本地 IP 地址监控所有流量（支持多网卡）

### 实时可视化
- 基于 Web 的仪表板，交互式图表（ECharts）
- 自动刷新功能
- 端口和主机视图切换
- 深色主题，专业样式

### 时间范围分析
- 多种时间范围：15分钟、30分钟、1小时、1天、3天、7天、30天
- 历史数据查询，自动降采样
- 峰值速率追踪

### 数据管理
- SQLite 数据库，WAL 模式支持高并发
- 3 层自动降采样（分钟 → 小时 → 天）
- 高效的内存聚合
- 自动数据清理

### 流量追踪
- 入站和出站流量分离
- 字节数和数据包计数
- 按源/远程 IP 追踪
- 实时速率计算

## 快速开始

### 一键安装（推荐）

**AMD64 (x86_64) 系统：**
```bash
curl -fsSL https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh | sudo bash
```

**ARM64/ARM 系统：**
```bash
# 脚本会自动从源码编译
curl -fsSL https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh | sudo bash
```

或者先下载脚本检查后再执行：

```bash
wget https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

### 安装脚本功能

安装脚本会自动完成：

1. **检测系统**
   - 自动识别 Linux 发行版（Ubuntu/Debian/CentOS/RHEL/Fedora/Arch）
   - 检测系统架构（x86_64/ARM64/ARM）

2. **安装依赖**
   - 安装 libpcap 运行时库
   - 安装 curl、wget、tar 工具

3. **获取程序**
   - **AMD64**：从 GitHub Releases 下载预编译二进制文件（快速！）
   - **ARM/ARM64**：自动从源码编译（需要 Go 和编译工具）

4. **配置服务**
   - 创建 systemd 服务
   - 配置开机自启动选项

5. **启动服务**
   - 可选择立即启动监控服务

### 支持的平台

**架构：**
- ✅ x86_64 (amd64) - **提供预编译二进制文件**
- ✅ ARM64 (aarch64) - 从源码编译
- ✅ ARM (armv7l) - 从源码编译

**发行版：**
- Ubuntu / Debian
- CentOS / RHEL / Fedora
- Arch Linux / Manjaro

## 手动安装

### 方式一：下载预编译二进制文件（仅 AMD64）

下载最新的 x86_64 版本：

```bash
# 下载最新 AMD64 二进制文件
wget https://github.com/xiaoxinmm/linux-traffic-monitor/releases/latest/download/traffic-monitor-v1.1.0-linux-amd64.tar.gz

# 解压
tar -xzf traffic-monitor-v1.1.0-linux-amd64.tar.gz

# 移动到系统路径
sudo mv traffic-monitor /usr/local/bin/

# 添加执行权限
sudo chmod +x /usr/local/bin/traffic-monitor
```

安装 libpcap 依赖：

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y libpcap0.8
```

**CentOS/RHEL:**
```bash
sudo yum install -y libpcap
```

**Arch Linux:**
```bash
sudo pacman -S libpcap
```

然后运行：
```bash
sudo traffic-monitor
```

### 方式二：从源码编译（所有架构）

#### 系统要求

- Linux 操作系统
- Go 1.21 或更高版本
- libpcap 开发库
- Root 权限（用于抓包）

#### 安装编译依赖

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y libpcap-dev golang-go git
```

**CentOS/RHEL:**
```bash
sudo yum install -y libpcap-devel golang git
```

**Arch Linux:**
```bash
sudo pacman -S libpcap go git
```

#### 编译和运行

```bash
# 克隆仓库
git clone https://github.com/xiaoxinmm/linux-traffic-monitor.git
cd linux-traffic-monitor

# 编译
go build -o traffic-monitor main.go

# 运行（需要 root 权限）
sudo ./traffic-monitor
```

程序启动后，Web 界面将在 `http://localhost:8080` 可用。

## 使用说明

### 使用 Systemd 服务（推荐）

如果使用安装脚本，监控程序已安装为 systemd 服务：

```bash
# 启动监控
sudo systemctl start traffic-monitor

# 设置开机自启
sudo systemctl enable traffic-monitor

# 查看状态
sudo systemctl status traffic-monitor

# 查看日志
sudo journalctl -u traffic-monitor -f

# 停止监控
sudo systemctl stop traffic-monitor

# 重启监控
sudo systemctl restart traffic-monitor
```

### 手动运行

```bash
sudo ./traffic-monitor
```

默认情况下，监控程序：
- 监听所有网络接口
- 捕获以下端口的流量：22, 80, 443, 3306, 6379, 8080, 9090
- Web UI 在 8080 端口提供服务
- 数据存储在 `traffic.db`

### 访问仪表板

在浏览器中打开：
```
http://服务器IP:8080
```

例如：
- 本地访问：`http://localhost:8080`
- 远程访问：`http://192.168.1.100:8080`

### 切换视图

使用仪表板顶部的切换按钮：
- **端口**：按监听端口查看流量
- **主机**：按本地 IP 地址查看流量

### 查询历史数据

1. 从下拉列表中选择端口或主机
2. 选择时间范围（15分钟到30天）
3. 点击"查询"查看流量图表

## API 端点

### 端口监控

- `GET /api/ports/active` - 获取所有活动端口及实时统计
- `GET /api/ports/stats?port=<port>&range=<range>` - 查询历史端口流量

### 主机监控

- `GET /api/hosts/active` - 获取所有活动主机及实时统计
- `GET /api/hosts/stats?host=<ip>&range=<range>` - 查询历史主机流量

### 参数

- `port`: 端口号（例如：80, 443）
- `host`: 本地 IP 地址（例如：192.168.1.100）
- `range`: 时间范围
  - `15m`, `30m`, `1h` - 最近数据（分钟粒度）
  - `1d`, `3d`, `7d` - 每日数据（小时粒度）
  - `30d` - 月度数据（天粒度）

### 响应格式

```json
{
  "success": true,
  "data": [
    {
      "timestamp": "2026-04-29T13:00:00Z",
      "bytes": 1048576,
      "packets": 1024,
      "rate": 17476.27,
      "direction": "inbound",
      "source_ip": "192.168.1.100"
    }
  ]
}
```

## 架构

### 数据流

```
网络数据包 → Pcap 捕获 → 数据包处理
                              ↓
                        内存聚合器
                        （实时）
                              ↓
                    ┌─────────┴─────────┐
                    ↓                   ↓
              端口统计              主机统计
                    ↓                   ↓
              SQLite 数据库（WAL 模式）
                    ↓
              自动降采样
          （分钟 → 小时 → 天）
```

### 降采样策略

- **粒度 0（分钟）**：原始数据，保留 2 小时
- **粒度 1（小时）**：按小时聚合，保留 8 天
- **粒度 2（天）**：按天聚合，保留 31 天

### 数据库架构

**port_traffic_stats**
- 按端口、源 IP 和方向追踪流量
- 在 (port, timestamp, granularity) 上建立索引

**host_traffic_stats**
- 按本地 IP、远程 IP 和方向追踪流量
- 在 (host_ip, timestamp, granularity) 上建立索引

## 配置

编辑 `main.go` 自定义监控端口：

```go
// 监控的端口
var listenPorts = map[int]bool{
    22:   true,  // SSH
    80:   true,  // HTTP
    443:  true,  // HTTPS
    3306: true,  // MySQL
    6379: true,  // Redis
    8080: true,  // 自定义
    9090: true,  // 自定义
}
```

编辑后，使用 `go build -o traffic-monitor main.go` 重新编译。

## 故障排除

### 权限被拒绝

监控程序需要 root 权限来捕获数据包：
```bash
sudo ./traffic-monitor
```

或授予 CAP_NET_RAW 能力：
```bash
sudo setcap cap_net_raw+ep ./traffic-monitor
./traffic-monitor
```

### 端口已被占用

如果 8080 端口已被占用，修改 `main.go` 中的 Web 服务器端口并重新编译。

### 没有捕获到流量

1. 检查监控的端口是否真的有流量
2. 验证网络接口是否启动：`ip link show`
3. 检查防火墙规则：`sudo iptables -L`

### 数据库锁定

如果看到 "database is locked" 错误：
1. 停止所有运行的实例
2. 删除锁：`rm traffic.db-wal traffic.db-shm`
3. 重启监控程序

### 卸载程序

```bash
# 停止并禁用服务
sudo systemctl stop traffic-monitor
sudo systemctl disable traffic-monitor

# 删除服务文件
sudo rm /etc/systemd/system/traffic-monitor.service

# 删除程序文件
sudo rm -rf /opt/linux-traffic-monitor
sudo rm /usr/local/bin/traffic-monitor

# 重新加载 systemd
sudo systemctl daemon-reload
```

## 性能

- 内存使用：约 50-100MB（取决于流量大小）
- CPU 使用：中等流量下约 5-10%
- 磁盘 I/O：最小化（WAL 模式 + 批量写入）
- 测试负载：持续 10K 包/秒

## 安全考虑

- 程序仅捕获数据包头部（不包含载荷）
- Web 界面无身份验证（建议使用防火墙或反向代理）
- 数据库包含 IP 地址（注意隐私法规）
- 以 root 身份运行（抓包需要，建议隔离）

## 许可证

MIT License

## 贡献

欢迎贡献！请随时提交 Pull Request。

## 支持

- 问题反馈：https://github.com/xiaoxinmm/linux-traffic-monitor/issues
- 讨论区：https://github.com/xiaoxinmm/linux-traffic-monitor/discussions
