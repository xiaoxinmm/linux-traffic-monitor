# 项目开发记录

## 项目状态

**当前版本**: v1.1.0  
**最后更新**: 2026-04-29  
**状态**: 功能完整，暂停开发

---

## 已完成功能

### 核心功能
- ✅ **双监控模式**
  - 端口级流量监控（按监听端口统计）
  - 主机级流量监控（按本地 IP 地址统计，支持多网卡）
  
- ✅ **实时数据采集**
  - 基于 libpcap 的高性能数据包捕获
  - 入站/出站流量分离统计
  - 字节数和数据包数计数
  - 实时速率计算（Bytes/s）
  - 峰值速率追踪

- ✅ **数据存储与管理**
  - SQLite 数据库（WAL 模式，支持高并发）
  - 3 层自动降采样（分钟 → 小时 → 天）
  - 高效内存聚合
  - 自动数据清理（保留策略：2小时/8天/31天）

- ✅ **Web 可视化界面**
  - 基于 ECharts 的交互式图表
  - 端口/主机视图切换
  - 多时间范围查询（15m, 30m, 1h, 1d, 3d, 7d, 30d）
  - 自动刷新功能
  - 深色主题

### 部署与安装
- ✅ **一键安装脚本** (`install.sh`)
  - 自动检测系统架构（x86_64/ARM64/ARM）
  - 自动检测 Linux 发行版（Ubuntu/Debian/CentOS/RHEL/Fedora/Arch）
  - AMD64 架构：自动下载预编译二进制文件
  - ARM 架构：自动从源码编译
  - glibc 兼容性检测（不兼容时自动回退到源码编译）
  - systemd 服务配置
  - 开机自启动选项

- ✅ **GitHub Actions CI/CD**
  - 自动构建 AMD64 预编译二进制文件
  - 使用 Ubuntu 20.04 编译（兼容更多系统）
  - 自动发布到 GitHub Releases
  - 版本标签触发自动构建

- ✅ **双语文档**
  - 英文文档：`README.md`
  - 中文文档：`README_CN.md`
  - 详细的安装、使用、配置说明

### 支持的平台
- ✅ **架构支持**
  - x86_64 (amd64) - 预编译二进制
  - ARM64 (aarch64) - 源码编译
  - ARM (armv7l) - 源码编译

- ✅ **发行版支持**
  - Ubuntu / Debian
  - CentOS / RHEL / Fedora
  - Arch Linux / Manjaro

---

## 技术架构

### 技术栈
- **语言**: Go 1.21+
- **数据包捕获**: libpcap + gopacket
- **数据库**: SQLite (modernc.org/sqlite)
- **前端**: 原生 HTML/CSS/JavaScript + ECharts
- **部署**: systemd service

### 数据流
```
网络数据包 → Pcap 捕获 → 数据包处理
                              ↓
                        内存聚合器（实时）
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

### 关键文件
- `main.go` - 主程序（约 1500 行）
- `index.html` - Web 界面（嵌入到二进制文件）
- `install.sh` - 一键安装脚本
- `.github/workflows/release.yml` - CI/CD 配置
- `traffic_monitor.db` - 数据库文件（运行时生成）

---

## 已知问题与限制

### 当前限制
1. **单网卡抓包**
   - 程序只在默认网卡上抓包
   - 但能正确识别和统计所有网卡的流量（通过 IP 判断）

2. **监听端口配置**
   - 默认监听端口硬编码在 `main.go` 中
   - 修改需要重新编译（未来可改为配置文件）

3. **Web 界面无认证**
   - 建议使用防火墙或反向代理保护
   - 或仅在内网使用

4. **需要 root 权限**
   - 数据包捕获需要 root 权限
   - 可使用 `setcap cap_net_raw+ep` 授权

### 已解决的问题
- ✅ glibc 版本兼容性（通过 Ubuntu 20.04 编译 + 兼容性检测）
- ✅ ARM 架构交叉编译（改为在目标机器上编译）
- ✅ 多网卡流量统计（自动识别所有本地 IP）

---

## 未来开发建议

### 优先级 P0（核心功能增强）
1. **配置文件支持**
   - 将监听端口、Web 端口等配置外部化
   - 支持 YAML/JSON 配置文件
   - 文件位置：`/etc/traffic-monitor/config.yaml`

2. **Web 界面认证**
   - 添加基本的用户名/密码认证
   - 或集成 OAuth2/LDAP
   - Session 管理

3. **告警功能**
   - 流量阈值告警
   - 支持 Webhook/邮件/Slack 通知
   - 异常流量检测

### 优先级 P1（易用性改进）
4. **命令行参数支持**
   ```bash
   traffic-monitor --config /path/to/config.yaml
   traffic-monitor --port 9090
   traffic-monitor --interface eth1
   ```

5. **多网卡抓包支持**
   - 同时在多个网卡上抓包
   - 适用于复杂网络环境

6. **数据导出功能**
   - 导出为 CSV/JSON
   - Prometheus metrics 接口
   - 与 Grafana 集成

### 优先级 P2（高级功能）
7. **协议分析**
   - HTTP/HTTPS 流量统计
   - DNS 查询统计
   - 应用层协议识别

8. **地理位置信息**
   - 集成 GeoIP 数据库
   - 显示远程 IP 的地理位置
   - 国家/城市级别统计

9. **性能优化**
   - 支持更高的数据包速率（当前测试 10K pps）
   - 优化内存使用
   - 数据库查询优化

10. **Docker 支持**
    - 提供官方 Docker 镜像
    - docker-compose 一键部署
    - 支持容器网络监控

---

## 开发环境设置

### 本地开发
```bash
# 克隆仓库
git clone https://github.com/xiaoxinmm/linux-traffic-monitor.git
cd linux-traffic-monitor

# 安装依赖
sudo apt-get install -y libpcap-dev golang-go

# 编译
go build -o traffic-monitor main.go

# 运行
sudo ./traffic-monitor
```

### 修改监听端口
编辑 `main.go`，找到以下代码段：
```go
// 监控的端口（约在第 320 行）
var listenPorts = map[int]bool{
    22:   true,  // SSH
    80:   true,  // HTTP
    443:  true,  // HTTPS
    3306: true,  // MySQL
    6379: true,  // Redis
    8080: true,  // Custom
    9090: true,  // Custom
}

// Web 服务器端口（约在第 1451 行）
server := &http.Server{
    Addr: ":8080",  // 修改这里
    ...
}
```

### 发布新版本
```bash
# 1. 更新版本号（在 README 和 install.sh 中）
# 2. 提交代码
git add .
git commit -m "Release v1.2.0"
git push

# 3. 创建版本标签
git tag v1.2.0
git push origin v1.2.0

# 4. GitHub Actions 会自动构建并发布
```

---

## 性能指标

### 测试环境
- CPU: 2 核
- 内存: 4GB
- 网络: 1Gbps

### 性能数据
- **内存使用**: 50-100MB（取决于流量大小）
- **CPU 使用**: 5-10%（中等流量）
- **磁盘 I/O**: 最小化（WAL 模式 + 批量写入）
- **测试负载**: 持续 10K 包/秒

---

## 安全考虑

1. **数据隐私**
   - 程序仅捕获数据包头部（不包含载荷）
   - 数据库包含 IP 地址（注意隐私法规）

2. **访问控制**
   - Web 界面无身份验证（需要额外保护）
   - 建议使用防火墙限制访问
   - 或使用 Nginx 反向代理 + 认证

3. **权限管理**
   - 需要 root 权限运行（抓包需要）
   - 建议隔离运行环境
   - 可使用 `setcap` 降低权限需求

---

## 相关资源

### 仓库信息
- **GitHub**: https://github.com/xiaoxinmm/linux-traffic-monitor
- **Issues**: https://github.com/xiaoxinmm/linux-traffic-monitor/issues
- **Discussions**: https://github.com/xiaoxinmm/linux-traffic-monitor/discussions

### 文档
- 英文文档: `README.md`
- 中文文档: `README_CN.md`
- 安装指南: `INSTALL_CN.md`

### 依赖库
- gopacket: https://github.com/google/gopacket
- modernc.org/sqlite: https://gitlab.com/cznic/sqlite
- ECharts: https://echarts.apache.org/

---

## 许可证

MIT License

---

## 联系方式

如需继续开发或有问题，请通过以下方式联系：
- GitHub Issues: https://github.com/xiaoxinmm/linux-traffic-monitor/issues
- GitHub Discussions: https://github.com/xiaoxinmm/linux-traffic-monitor/discussions

---

**最后更新**: 2026-04-29  
**维护者**: xiaoxinmm
