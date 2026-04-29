# 安装指南

## 快速安装（推荐）

### 一键安装

使用以下命令一键安装 Linux 流量监控系统：

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh | sudo bash
```

或者先下载脚本检查后再执行：

```bash
wget https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

### 安装脚本功能

安装脚本会自动完成以下操作：

1. **检测系统环境**
   - 自动识别 Linux 发行版（Ubuntu/Debian/CentOS/RHEL/Fedora/Arch）
   - 自动识别系统架构（x86_64/ARM64/ARM）

2. **安装依赖**
   - 安装 libpcap 运行时库
   - 安装 curl、wget、tar 等工具

3. **获取程序**
   - 优先从 GitHub Releases 下载预编译的二进制文件（速度快）
   - 如果预编译版本不可用，自动切换到源码编译模式

4. **配置服务**
   - 创建 systemd 服务
   - 配置开机自启动选项

5. **启动服务**
   - 可选择立即启动监控服务

### 支持的平台

**架构支持：**
- x86_64 (amd64)
- ARM64 (aarch64)
- ARM (armv7l)

**发行版支持：**
- Ubuntu / Debian
- CentOS / RHEL / Fedora
- Arch Linux / Manjaro

## 手动安装

### 方式一：下载预编译二进制文件

根据您的系统架构下载对应的预编译版本：

#### x86_64 (amd64) 系统

```bash
# 下载最新版本
wget https://github.com/xiaoxinmm/linux-traffic-monitor/releases/latest/download/traffic-monitor-linux-amd64.tar.gz

# 解压
tar -xzf traffic-monitor-linux-amd64.tar.gz

# 移动到系统路径
sudo mv traffic-monitor /usr/local/bin/

# 添加执行权限
sudo chmod +x /usr/local/bin/traffic-monitor
```

#### ARM64 系统

```bash
wget https://github.com/xiaoxinmm/linux-traffic-monitor/releases/latest/download/traffic-monitor-linux-arm64.tar.gz
tar -xzf traffic-monitor-linux-arm64.tar.gz
sudo mv traffic-monitor /usr/local/bin/
sudo chmod +x /usr/local/bin/traffic-monitor
```

#### ARM (32位) 系统

```bash
wget https://github.com/xiaoxinmm/linux-traffic-monitor/releases/latest/download/traffic-monitor-linux-arm.tar.gz
tar -xzf traffic-monitor-linux-arm.tar.gz
sudo mv traffic-monitor /usr/local/bin/
sudo chmod +x /usr/local/bin/traffic-monitor
```

#### 安装运行时依赖

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

#### 运行程序

```bash
sudo traffic-monitor
```

### 方式二：从源码编译

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

## 配置 Systemd 服务

如果您手动安装，可以手动创建 systemd 服务：

### 创建服务文件

```bash
sudo nano /etc/systemd/system/traffic-monitor.service
```

添加以下内容：

```ini
[Unit]
Description=Linux Traffic Monitor
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/linux-traffic-monitor
ExecStart=/usr/local/bin/traffic-monitor
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

### 启用和启动服务

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start traffic-monitor

# 设置开机自启
sudo systemctl enable traffic-monitor

# 查看服务状态
sudo systemctl status traffic-monitor
```

## 使用说明

### 服务管理

```bash
# 启动服务
sudo systemctl start traffic-monitor

# 停止服务
sudo systemctl stop traffic-monitor

# 重启服务
sudo systemctl restart traffic-monitor

# 查看状态
sudo systemctl status traffic-monitor

# 查看日志
sudo journalctl -u traffic-monitor -f

# 开机自启
sudo systemctl enable traffic-monitor

# 禁用开机自启
sudo systemctl disable traffic-monitor
```

### 访问 Web 界面

在浏览器中打开：

```
http://服务器IP:8080
```

例如：
- 本地访问：`http://localhost:8080`
- 远程访问：`http://192.168.1.100:8080`

### 功能说明

**监控模式：**
- **端口监控**：按监听端口统计流量
- **主机监控**：按本地 IP 地址统计流量（支持多网卡）

**时间范围：**
- 15分钟、30分钟、1小时（分钟级精度）
- 1天、3天、7天（小时级精度）
- 30天（天级精度）

**流量统计：**
- 入站流量和出站流量分离
- 字节数和数据包数统计
- 实时速率计算
- 峰值速率追踪

## 常见问题

### 权限错误

监控程序需要 root 权限来捕获网络数据包：

```bash
sudo ./traffic-monitor
```

或者授予 CAP_NET_RAW 能力：

```bash
sudo setcap cap_net_raw+ep ./traffic-monitor
./traffic-monitor
```

### 端口被占用

如果 8080 端口已被占用，可以修改 `main.go` 中的端口配置后重新编译。

### 没有捕获到流量

1. 检查监控的端口是否真的有流量
2. 验证网络接口是否启动：`ip link show`
3. 检查防火墙规则：`sudo iptables -L`
4. 确认以 root 权限运行

### 数据库锁定

如果看到 "database is locked" 错误：

```bash
# 停止所有运行的实例
sudo systemctl stop traffic-monitor

# 删除锁文件
sudo rm /opt/linux-traffic-monitor/traffic.db-wal
sudo rm /opt/linux-traffic-monitor/traffic.db-shm

# 重启服务
sudo systemctl start traffic-monitor
```

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

## 性能指标

- **内存使用**：约 50-100MB（取决于流量大小）
- **CPU 使用**：中等流量下约 5-10%
- **磁盘 I/O**：最小化（WAL 模式 + 批量写入）
- **测试负载**：持续 10K 包/秒

## 安全建议

- 程序仅捕获数据包头部（不包含载荷内容）
- Web 界面无身份验证（建议使用防火墙或反向代理）
- 数据库包含 IP 地址（注意隐私法规）
- 需要 root 权限运行（建议隔离运行环境）

## 技术支持

- 问题反馈：https://github.com/xiaoxinmm/linux-traffic-monitor/issues
- 讨论区：https://github.com/xiaoxinmm/linux-traffic-monitor/discussions
