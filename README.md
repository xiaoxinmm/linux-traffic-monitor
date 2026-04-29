# 🚀 Linux Port Traffic Monitor

一个高性能的 Linux 端口流量监控 Web 程序，使用 Go 语言开发，支持实时抓包、数据聚合、历史查询和可视化展示。

## ✨ 核心特性

- **自动端口发现**：自动识别所有处于 LISTEN 状态的 TCP/UDP 端口
- **实时流量监控**：基于 `gopacket` 的高性能数据包捕获
- **多维度统计**：按 `[来源IP + 本地端口]` 维度统计流量
- **智能降采样**：三级粒度数据存储（分钟/小时/天），自动清理过期数据
- **Web 可视化**：基于 ECharts 的实时图表展示
- **单文件部署**：前端资源通过 `go:embed` 打包，无需额外文件

## 📊 监控指标

- 实时速率 (Bytes/s)
- 总流量 (Total Bytes)
- 数据包总数 (Packets)
- 峰值速率 (Peak Rate)
- 平均速率 (Average Rate)
- 入站/出站流量分离统计

## 🕐 时间跨度支持

| 时间范围 | 数据粒度 | 保留时长 |
|---------|---------|---------|
| 15m / 30m / 60m | 1 分钟 | 6 小时 |
| 1d / 3d / 7d | 1 小时 | 7 天 |
| 30d | 1 天 | 30 天 |

## 🛠️ 技术栈

- **网络捕获**：`google/gopacket` + `libpcap`
- **数据库**：`modernc.org/sqlite`（纯 Go 实现，无 CGO 依赖）
- **Web 框架**：Go 标准库 `net/http`
- **前端可视化**：HTML5 + ECharts 5.x

## 📦 安装依赖

### 1. 安装 libpcap 开发库

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y libpcap-dev

# CentOS/RHEL
sudo yum install -y libpcap-devel

# Arch Linux
sudo pacman -S libpcap
```

### 2. 下载 Go 依赖

```bash
go mod download
```

## 🚀 编译与运行

### 方式一：使用启动脚本（推荐）

**使用 sudo 运行（自动处理权限）：**
```bash
sudo ./start.sh
```

**或者先编译，再运行：**
```bash
./build.sh              # 编译（不需要 sudo）
sudo ./traffic-monitor  # 运行（需要 sudo）
```

### 方式二：手动编译

```bash
go build -o traffic-monitor main.go
```

### 方式三：使用 Makefile

```bash
make build              # 编译
sudo make run           # 编译并运行
```

### 运行方式

**选项 1：使用 sudo（推荐）**
```bash
sudo ./traffic-monitor
```

**选项 2：赋予 CAP_NET_RAW 能力（避免 root 运行）**
```bash
sudo setcap cap_net_raw+ep ./traffic-monitor
./traffic-monitor
```

> **注意**：数据包捕获需要 root 权限或 CAP_NET_RAW 能力。

## 🌐 访问 Web 界面

启动后访问：

```
http://localhost:8080
```

## 📂 项目结构

```
.
├── main.go          # 主程序（包含所有模块）
├── index.html       # 前端页面（通过 go:embed 嵌入）
├── go.mod           # Go 模块依赖
├── README.md        # 使用文档
└── traffic_monitor.db  # SQLite 数据库（运行时自动创建）
```

## 🏗️ 系统架构

### 数据流向

```
网卡数据包 → Pcap 捕获 → 内存 Map 聚合 → 每分钟快照 → SQLite 批量写入 → 降采样处理
                ↓
            实时 API 查询
```

### 核心模块

#### 1️⃣ 内存聚合器 (Memory Aggregator)
- 使用 `sync.Pool` 管理数据包 buffer，减少 GC 压力
- 并发安全的 Map 存储实时流量统计
- 每分钟生成快照并重置计数器

#### 2️⃣ Pcap 抓包引擎
- BPF 过滤器：只捕获 TCP/UDP 流量
- 支持 IPv4/IPv6
- 自动识别入站/出站流量

#### 3️⃣ SQLite 持久化层
- WAL 模式提升并发性能
- 事务批量写入
- 自动降采样与数据清理

#### 4️⃣ Web API
- `GET /api/ports/active`：实时活跃端口列表
- `GET /api/ports/stats?port={port}&range={range}`：历史流量查询

## 🔧 配置说明

### 修改监听端口

编辑 `main.go` 第 596 行：

```go
Addr: ":8080",  // 修改为其他端口
```

### 修改数据库路径

编辑 `main.go` 第 520 行：

```go
database, err = NewDatabase("traffic_monitor.db")  // 修改路径
```

### 调整降采样策略

编辑 `DownsampleAndCleanup()` 函数中的时间参数：

```go
sixHoursAgo := now.Add(-6 * time.Hour).Unix()    // 粒度 0 保留时长
sevenDaysAgo := now.Add(-7 * 24 * time.Hour).Unix()  // 粒度 1 保留时长
thirtyDaysAgo := now.Add(-30 * 24 * time.Hour).Unix()  // 粒度 2 保留时长
```

## 📈 性能优化

### 1. GC 优化
- 使用 `sync.Pool` 复用 packet buffer
- 避免频繁的内存分配

### 2. 数据库优化
- WAL 模式：读写并发
- 批量事务写入
- 索引优化：`(port, timestamp, granularity)`

### 3. 并发控制
- `sync.RWMutex` 保护共享数据
- 读多写少场景优化

## 🐛 故障排查

### 问题 1：权限不足

```
Error: failed to open interface eth0: Operation not permitted
```

**解决方案**：使用 `sudo` 运行或赋予 CAP_NET_RAW 能力。

### 问题 2：找不到网络接口

```
Error: no suitable network interface found
```

**解决方案**：检查网络接口状态 `ip link show`，确保至少有一个非 loopback 接口处于 UP 状态。

### 问题 3：libpcap 未安装

```
Error: pcap.h: No such file or directory
```

**解决方案**：安装 libpcap 开发库（见安装依赖章节）。

### 问题 4：端口已被占用

```
Error: listen tcp :8080: bind: address already in use
```

**解决方案**：修改监听端口或停止占用 8080 端口的进程。

## 📊 数据库表结构

```sql
CREATE TABLE traffic_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,           -- Unix 时间戳
    port INTEGER NOT NULL,                -- 端口号
    source_ip TEXT NOT NULL,              -- 来源 IP
    direction TEXT NOT NULL,              -- inbound/outbound
    bytes INTEGER NOT NULL,               -- 字节数
    packets INTEGER NOT NULL,             -- 数据包数
    peak_rate REAL NOT NULL,              -- 峰值速率
    granularity INTEGER NOT NULL DEFAULT 0,  -- 粒度 (0/1/2)
    created_at INTEGER NOT NULL           -- 创建时间
);
```

## 🔒 安全建议

1. **生产环境部署**：建议使用反向代理（Nginx/Caddy）并启用 HTTPS
2. **访问控制**：添加身份认证机制（Basic Auth / JWT）
3. **防火墙规则**：限制 Web 端口的访问来源
4. **日志审计**：记录所有 API 访问日志

## 📝 TODO

- [ ] 支持多网卡监控
- [ ] 添加告警功能（流量阈值）
- [ ] 支持导出数据（CSV/JSON）
- [ ] 添加用户认证
- [ ] 支持 Prometheus metrics 导出
- [ ] Docker 容器化部署

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 👨‍💻 作者

由 AI 助手生成，基于高性能 Go 网络编程最佳实践。
