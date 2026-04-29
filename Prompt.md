# 角色与背景 (Role & Context)
你是一位精通 Golang 底层网络编程与高性能并发设计的专家。
我需要你使用 Go 开发一个单文件的中量级 Linux 端口流量监控 Web 程序。该程序需要包含底层的网络抓包逻辑、内存数据聚合、SQLite 存储以及提供前端图表展示所需的 API。

# 核心功能需求 (Core Requirements)
1.  **监控范围**：自动识别并监控服务器上所有“处于 LISTEN 状态（已开启）”的端口。
2.  **数据维度**：以 `[来源 IP] + [本地端口]` 作为联合键（Composite Key）。
3.  **核心指标**：实时速率 (Bytes/s)、总流量 (Total Bytes)、数据包总数 (Packets)、峰值速率 (Peak Rate)、平均速率 (Average Rate)。
4.  **时间跨度支持**：
    * 实时级：Now (实时速率，无需查库)。
    * 短期级：15m (分钟), 30m, 60m。
    * 长期级：1d (天), 3d, 7d, 30d。

# 技术栈选型 (Tech Stack)
* **网络捕获**：使用 `google/gopacket` 与 `pcap`。
* **Web 与 API**：使用 Go 标准库 `net/http` 或轻量级路由，结合 `embed` 打包单页前端 (HTML + Echarts)。
* **持久化**：使用纯 Go 实现的 `modernc.org/sqlite`（无 CGO 依赖）。

# 系统架构与模块设计 (Architecture Design)

请按以下模块化思路思考，并输出核心代码：

### 模块一：高性能数据抓取 (Capture & Memory Aggregation)
1.  实现一个 Pcap 监听器，捕获进出本机网卡的数据包。
2.  使用 `sync.Pool` 管理数据包对象，避免频繁分配内存导致 GC 压力过大。
3.  **内存状态管理**：维护一个并发安全的 Map，用于存储当前分钟的实时数据（计算 Now 速率）。
4.  设计一个 Ticker，每分钟将内存中的聚合数据“快照”传递给持久化层，并重置当前分钟的计数器。

### 模块二：基于 SQLite 的持久化与降采样 (Persistence & Downsampling)
1.  **初始化配置**：启动时执行 `PRAGMA journal_mode=WAL;` 提升读写性能。
2.  **表结构设计**：包含时间戳、IP、端口、入站/出站流量和包数、峰值速率，以及一个 `granularity` (粒度) 字段。
3.  **批量写入**：将模块一传来的“每分钟快照”通过事务批量 (Batch Insert) 写入数据库。
4.  **降采样与清理任务 (Retention Task)**：
    * 启动一个后台协程，定期执行 SQL。
    * **粒度 0 (1 分钟)**：保留最近 6 小时。超过时间的记录，聚合成粒度 1。
    * **粒度 1 (1 小时)**：保留最近 7 天。超过时间的记录，聚合成粒度 2。
    * **粒度 2 (1 天)**：保留最近 30 天。超期的直接清理 (DELETE)。

### 模块三：Web API 设计 (API Endpoints)
实现以下 JSON API：
1.  `GET /api/ports/active`：返回当前所有开启监控的端口及其**实时内存中**的速率 (Now)。
2.  `GET /api/ports/stats?port={port}&range={range}`：
    * `range` 可选值为: `15m`, `30m`, `60m`, `1d`, `3d`, `7d`, `30d`。
    * 根据 `range` 智能选择查询 SQLite 中的哪个 `granularity`，返回用于图表绘制的时序数据，并包含该时段内的流量峰值和均值。

# 交付要求 (Deliverables)
请先简要确认你对**数据流向（Pcap -> Map聚合 -> SQLite批量写）**以及**降采样策略**的理解。确认无误后，请输出完整的核心模块代码，特别是结构体设计、Pcap 抓包的 GC 优化部分，以及 SQLite 的数据清理逻辑。
# Deliverables
请先向我确认你对系统架构的理解，并简述你打算采用的**数据捕获方案（pcap vs eBPF）**和**降采样策略**。确认无误后，再按模块逐步输出 Go 源码。