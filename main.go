// main.go
package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	_ "modernc.org/sqlite"
)

//go:embed index.html
var embeddedFS embed.FS

// ============================================================================
// 数据结构定义
// ============================================================================

// TrafficKey 唯一标识一个流量记录的键
type TrafficKey struct {
	Port      uint16
	SourceIP  string
	Direction string // "inbound" or "outbound"
}

// TrafficStats 流量统计数据
type TrafficStats struct {
	Bytes      uint64
	Packets    uint64
	LastUpdate time.Time
	PeakRate   float64 // Bytes/s
}

// TrafficSnapshot 每分钟的快照数据
type TrafficSnapshot struct {
	Timestamp time.Time
	Port      uint16
	SourceIP  string
	Direction string
	Bytes     uint64
	Packets   uint64
	Rate      float64 // 当前速率
}

// HostTrafficKey 唯一标识主机流量记录的键
type HostTrafficKey struct {
	HostIP    string // 本地主机IP
	RemoteIP  string // 远程IP
	Direction string // "inbound" or "outbound"
}

// HostTrafficSnapshot 主机流量快照数据
type HostTrafficSnapshot struct {
	Timestamp time.Time
	HostIP    string
	RemoteIP  string
	Direction string
	Bytes     uint64
	Packets   uint64
	Rate      float64
}

// MemoryAggregator 内存聚合器
type MemoryAggregator struct {
	mu       sync.RWMutex
	portData map[TrafficKey]*TrafficStats
	hostData map[HostTrafficKey]*TrafficStats
	pools    *sync.Pool // packet buffer pool
}

// Database SQLite 数据库封装
type Database struct {
	db *sql.DB
}

// ============================================================================
// 全局变量
// ============================================================================

var (
	aggregator  *MemoryAggregator
	database    *Database
	listenPorts map[uint16]bool
	portsMu     sync.RWMutex
)

// ============================================================================
// 模块一：内存聚合器实现
// ============================================================================

func NewMemoryAggregator() *MemoryAggregator {
	return &MemoryAggregator{
		portData: make(map[TrafficKey]*TrafficStats),
		hostData: make(map[HostTrafficKey]*TrafficStats),
		pools: &sync.Pool{
			New: func() interface{} {
				// 预分配 64KB 的 buffer，适合大多数数据包
				return make([]byte, 65536)
			},
		},
	}
}

// Update 更新流量统计（并发安全）
func (ma *MemoryAggregator) Update(key TrafficKey, bytes uint64) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	stats, exists := ma.portData[key]
	if !exists {
		stats = &TrafficStats{
			LastUpdate: time.Now(),
		}
		ma.portData[key] = stats
	}

	stats.Bytes += bytes
	stats.Packets++

	// 计算瞬时速率（基于上次更新时间）
	now := time.Now()
	elapsed := now.Sub(stats.LastUpdate).Seconds()
	if elapsed > 0 {
		currentRate := float64(bytes) / elapsed
		if currentRate > stats.PeakRate {
			stats.PeakRate = currentRate
		}
	}
	stats.LastUpdate = now
}

// GetSnapshot 获取当前快照并重置计数器
func (ma *MemoryAggregator) GetSnapshot() []TrafficSnapshot {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	now := time.Now()
	snapshots := make([]TrafficSnapshot, 0, len(ma.portData))

	for key, stats := range ma.portData {
		// 计算平均速率（过去一分钟）
		elapsed := now.Sub(stats.LastUpdate).Seconds()
		if elapsed == 0 {
			elapsed = 60 // 默认一分钟
		}
		avgRate := float64(stats.Bytes) / elapsed

		snapshots = append(snapshots, TrafficSnapshot{
			Timestamp: now,
			Port:      key.Port,
			SourceIP:  key.SourceIP,
			Direction: key.Direction,
			Bytes:     stats.Bytes,
			Packets:   stats.Packets,
			Rate:      avgRate,
		})
	}

	// 重置计数器（保留 key 结构，只清零数据）
	ma.portData = make(map[TrafficKey]*TrafficStats)

	return snapshots
}

// GetRealTimeStats 获取实时统计（不重置）
func (ma *MemoryAggregator) GetRealTimeStats() map[uint16]map[string]interface{} {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	result := make(map[uint16]map[string]interface{})

	for key, stats := range ma.portData {
		if _, exists := result[key.Port]; !exists {
			result[key.Port] = map[string]interface{}{
				"port":          key.Port,
				"total_bytes":   uint64(0),
				"total_packets": uint64(0),
				"peak_rate":     float64(0),
				"sources":       make([]map[string]interface{}, 0),
			}
		}

		portData := result[key.Port]
		portData["total_bytes"] = portData["total_bytes"].(uint64) + stats.Bytes
		portData["total_packets"] = portData["total_packets"].(uint64) + stats.Packets

		if stats.PeakRate > portData["peak_rate"].(float64) {
			portData["peak_rate"] = stats.PeakRate
		}

		// 计算当前速率
		elapsed := time.Since(stats.LastUpdate).Seconds()
		if elapsed == 0 {
			elapsed = 1
		}
		currentRate := float64(stats.Bytes) / elapsed

		sources := portData["sources"].([]map[string]interface{})
		sources = append(sources, map[string]interface{}{
			"ip":        key.SourceIP,
			"direction": key.Direction,
			"bytes":     stats.Bytes,
			"packets":   stats.Packets,
			"rate":      currentRate,
		})
		portData["sources"] = sources
	}

	return result
}

// UpdateHost updates host-level traffic statistics
func (ma *MemoryAggregator) UpdateHost(key HostTrafficKey, bytes uint64) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	stats, exists := ma.hostData[key]
	if !exists {
		stats = &TrafficStats{
			LastUpdate: time.Now(),
		}
		ma.hostData[key] = stats
	}

	stats.Bytes += bytes
	stats.Packets++

	// 计算峰值速率
	now := time.Now()
	elapsed := now.Sub(stats.LastUpdate).Seconds()
	if elapsed > 0 {
		currentRate := float64(bytes) / elapsed
		if currentRate > stats.PeakRate {
			stats.PeakRate = currentRate
		}
	}
	stats.LastUpdate = now
}

// GetHostSnapshot returns a snapshot of all host traffic data and resets counters
func (ma *MemoryAggregator) GetHostSnapshot() []HostTrafficSnapshot {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	now := time.Now()
	snapshots := make([]HostTrafficSnapshot, 0, len(ma.hostData))

	for key, stats := range ma.hostData {
		elapsed := now.Sub(stats.LastUpdate).Seconds()
		if elapsed == 0 {
			elapsed = 60 // 默认一分钟
		}
		avgRate := float64(stats.Bytes) / elapsed

		snapshots = append(snapshots, HostTrafficSnapshot{
			Timestamp: now,
			HostIP:    key.HostIP,
			RemoteIP:  key.RemoteIP,
			Direction: key.Direction,
			Bytes:     stats.Bytes,
			Packets:   stats.Packets,
			Rate:      avgRate,
		})
	}

	// 重置计数器
	ma.hostData = make(map[HostTrafficKey]*TrafficStats)

	return snapshots
}

// GetRealTimeHostStats returns real-time host statistics without resetting
func (ma *MemoryAggregator) GetRealTimeHostStats() map[string]map[string]interface{} {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	result := make(map[string]map[string]interface{})

	for key, stats := range ma.hostData {
		if _, exists := result[key.HostIP]; !exists {
			result[key.HostIP] = map[string]interface{}{
				"host_ip":       key.HostIP,
				"total_bytes":   uint64(0),
				"total_packets": uint64(0),
				"peak_rate":     float64(0),
				"remotes":       make([]map[string]interface{}, 0),
			}
		}

		hostData := result[key.HostIP]
		hostData["total_bytes"] = hostData["total_bytes"].(uint64) + stats.Bytes
		hostData["total_packets"] = hostData["total_packets"].(uint64) + stats.Packets

		if stats.PeakRate > hostData["peak_rate"].(float64) {
			hostData["peak_rate"] = stats.PeakRate
		}

		// 计算当前速率
		elapsed := time.Since(stats.LastUpdate).Seconds()
		if elapsed == 0 {
			elapsed = 1
		}
		currentRate := float64(stats.Bytes) / elapsed

		remotes := hostData["remotes"].([]map[string]interface{})
		remotes = append(remotes, map[string]interface{}{
			"ip":        key.RemoteIP,
			"direction": key.Direction,
			"bytes":     stats.Bytes,
			"packets":   stats.Packets,
			"rate":      currentRate,
		})
		hostData["remotes"] = remotes
	}

	return result
}

// ============================================================================
// 本地IP检测（支持多网卡）
// ============================================================================

var (
	localIPCache     map[string]bool
	localIPCacheMu   sync.RWMutex
	localIPCacheTime time.Time
	localIPCacheTTL  = 5 * time.Minute
)

// getAllLocalIPs returns all local IP addresses from all network interfaces
func getAllLocalIPs() (map[string]bool, error) {
	localIPs := make(map[string]bool)

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		// 跳过down状态的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf("Warning: failed to get addresses for interface %s: %v", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			// 只处理IPv4地址
			if ip.To4() != nil {
				localIPs[ip.String()] = true
			}
		}
	}

	return localIPs, nil
}

// getAllLocalIPsCached returns cached local IPs with TTL
func getAllLocalIPsCached() map[string]bool {
	localIPCacheMu.RLock()
	if time.Since(localIPCacheTime) < localIPCacheTTL && localIPCache != nil {
		defer localIPCacheMu.RUnlock()
		return localIPCache
	}
	localIPCacheMu.RUnlock()

	// 需要刷新缓存
	localIPCacheMu.Lock()
	defer localIPCacheMu.Unlock()

	// 双重检查
	if time.Since(localIPCacheTime) < localIPCacheTTL && localIPCache != nil {
		return localIPCache
	}

	ips, err := getAllLocalIPs()
	if err != nil {
		log.Printf("Warning: failed to refresh local IP cache: %v", err)
		if localIPCache != nil {
			return localIPCache // 返回旧缓存
		}
		return make(map[string]bool)
	}

	localIPCache = ips
	localIPCacheTime = time.Now()
	log.Printf("Refreshed local IP cache: %d IPs found", len(ips))

	return localIPCache
}

// ============================================================================
// 模块二：Pcap 抓包实现（带 GC 优化）
// ============================================================================

func startPacketCapture(iface string) error {
	// 打开网络接口
	handle, err := pcap.OpenLive(iface, 65536, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("failed to open interface %s: %v", iface, err)
	}
	defer handle.Close()

	log.Printf("Started packet capture on interface: %s", iface)

	// 设置 BPF 过滤器：只捕获 TCP/UDP 流量
	if err := handle.SetBPFFilter("tcp or udp"); err != nil {
		return fmt.Errorf("failed to set BPF filter: %v", err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for packet := range packetSource.Packets() {
		processPacket(packet)
	}

	return nil
}

func processPacket(packet gopacket.Packet) {
	// 从 pool 获取 buffer（虽然这里不直接用，但展示 pool 机制）
	buf := aggregator.pools.Get().([]byte)
	defer aggregator.pools.Put(buf) // 确保归还到 pool

	// 解析网络层
	networkLayer := packet.NetworkLayer()
	if networkLayer == nil {
		return
	}

	// 解析传输层
	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		return
	}

	var srcIP, dstIP string
	var srcPort, dstPort uint16

	// 提取 IP 地址
	if ipv4Layer := packet.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
		ipv4, _ := ipv4Layer.(*layers.IPv4)
		srcIP = ipv4.SrcIP.String()
		dstIP = ipv4.DstIP.String()
	} else if ipv6Layer := packet.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {
		ipv6, _ := ipv6Layer.(*layers.IPv6)
		srcIP = ipv6.SrcIP.String()
		dstIP = ipv6.DstIP.String()
	} else {
		return
	}

	// 提取端口号
	switch layer := transportLayer.(type) {
	case *layers.TCP:
		srcPort = uint16(layer.SrcPort)
		dstPort = uint16(layer.DstPort)
	case *layers.UDP:
		srcPort = uint16(layer.SrcPort)
		dstPort = uint16(layer.DstPort)
	default:
		return
	}

	// 获取数据包大小
	packetSize := uint64(len(packet.Data()))

	// 判断是入站还是出站流量
	localIPs := getAllLocalIPsCached()

	portsMu.RLock()
	defer portsMu.RUnlock()

	// 检查是否涉及本地IP
	isLocalSrc := localIPs[srcIP]
	isLocalDst := localIPs[dstIP]

	// 端口级别流量跟踪（保持原有逻辑）
	// 入站流量：目标端口是监听端口且目标IP是本地IP
	if listenPorts[dstPort] && isLocalDst {
		key := TrafficKey{
			Port:      dstPort,
			SourceIP:  srcIP,
			Direction: "inbound",
		}
		aggregator.Update(key, packetSize)
	}

	// 出站流量：源端口是监听端口且源IP是本地IP
	if listenPorts[srcPort] && isLocalSrc {
		key := TrafficKey{
			Port:      srcPort,
			SourceIP:  dstIP,
			Direction: "outbound",
		}
		aggregator.Update(key, packetSize)
	}

	// 主机级别流量跟踪（新增逻辑）
	// 只跟踪涉及本地IP的流量
	if isLocalSrc || isLocalDst {
		var localIP, remoteIP string
		var direction string

		if isLocalSrc {
			localIP = srcIP
			remoteIP = dstIP
			direction = "outbound"
		} else {
			localIP = dstIP
			remoteIP = srcIP
			direction = "inbound"
		}

		hostKey := HostTrafficKey{
			HostIP:    localIP,
			RemoteIP:  remoteIP,
			Direction: direction,
		}
		aggregator.UpdateHost(hostKey, packetSize)
	}
}

// ============================================================================
// 模块三：SQLite 持久化与降采样
// ============================================================================

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 启用 WAL 模式提升并发性能
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, err
	}

	// 创建表结构
	schema := `
	CREATE TABLE IF NOT EXISTS traffic_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		port INTEGER NOT NULL,
		source_ip TEXT NOT NULL,
		direction TEXT NOT NULL,
		bytes INTEGER NOT NULL,
		packets INTEGER NOT NULL,
		peak_rate REAL NOT NULL,
		granularity INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_traffic_query
		ON traffic_stats(port, timestamp, granularity);

	CREATE INDEX IF NOT EXISTS idx_traffic_cleanup
		ON traffic_stats(granularity, timestamp);

	CREATE TABLE IF NOT EXISTS host_traffic_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		host_ip TEXT NOT NULL,
		remote_ip TEXT NOT NULL,
		direction TEXT NOT NULL,
		bytes INTEGER NOT NULL,
		packets INTEGER NOT NULL,
		peak_rate REAL NOT NULL,
		granularity INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_host_traffic_query
		ON host_traffic_stats(host_ip, timestamp, granularity);

	CREATE INDEX IF NOT EXISTS idx_host_traffic_cleanup
		ON host_traffic_stats(granularity, timestamp);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

// BatchInsert 批量插入快照数据
func (d *Database) BatchInsert(snapshots []TrafficSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO traffic_stats
		(timestamp, port, source_ip, direction, bytes, packets, peak_rate, granularity, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, snap := range snapshots {
		_, err := stmt.Exec(
			snap.Timestamp.Unix(),
			snap.Port,
			snap.SourceIP,
			snap.Direction,
			snap.Bytes,
			snap.Packets,
			snap.Rate,
			now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// BatchInsertHostSnapshots 批量插入主机快照数据
func (d *Database) BatchInsertHostSnapshots(snapshots []HostTrafficSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO host_traffic_stats
		(timestamp, host_ip, remote_ip, direction, bytes, packets, peak_rate, granularity, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, snap := range snapshots {
		_, err := stmt.Exec(
			snap.Timestamp.Unix(),
			snap.HostIP,
			snap.RemoteIP,
			snap.Direction,
			snap.Bytes,
			snap.Packets,
			snap.Rate,
			now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DownsampleAndCleanup 降采样与数据清理（完整 SQL 实现）
func (d *Database) DownsampleAndCleanup() error {
	now := time.Now()

	// 1. 粒度 0 → 粒度 1：将 6 小时前的分钟数据聚合为小时数据
	sixHoursAgo := now.Add(-6 * time.Hour).Unix()

	log.Println("Starting downsampling: granularity 0 -> 1")

	_, err := d.db.Exec(`
		INSERT INTO traffic_stats
		(timestamp, port, source_ip, direction, bytes, packets, peak_rate, granularity, created_at)
		SELECT
			(timestamp / 3600) * 3600 as hour_timestamp,
			port,
			source_ip,
			direction,
			SUM(bytes) as total_bytes,
			SUM(packets) as total_packets,
			MAX(peak_rate) as max_peak_rate,
			1 as granularity,
			? as created_at
		FROM traffic_stats
		WHERE granularity = 0 AND timestamp < ?
		GROUP BY hour_timestamp, port, source_ip, direction
	`, now.Unix(), sixHoursAgo)

	if err != nil {
		return fmt.Errorf("failed to downsample to granularity 1: %v", err)
	}

	// 删除已聚合的粒度 0 数据
	result, err := d.db.Exec(`
		DELETE FROM traffic_stats
		WHERE granularity = 0 AND timestamp < ?
	`, sixHoursAgo)

	if err != nil {
		return fmt.Errorf("failed to delete old granularity 0 data: %v", err)
	}

	deleted, _ := result.RowsAffected()
	log.Printf("Deleted %d granularity 0 records", deleted)

	// 2. 粒度 1 → 粒度 2：将 7 天前的小时数据聚合为天数据
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour).Unix()

	log.Println("Starting downsampling: granularity 1 -> 2")

	_, err = d.db.Exec(`
		INSERT INTO traffic_stats
		(timestamp, port, source_ip, direction, bytes, packets, peak_rate, granularity, created_at)
		SELECT
			(timestamp / 86400) * 86400 as day_timestamp,
			port,
			source_ip,
			direction,
			SUM(bytes) as total_bytes,
			SUM(packets) as total_packets,
			MAX(peak_rate) as max_peak_rate,
			2 as granularity,
			? as created_at
		FROM traffic_stats
		WHERE granularity = 1 AND timestamp < ?
		GROUP BY day_timestamp, port, source_ip, direction
	`, now.Unix(), sevenDaysAgo)

	if err != nil {
		return fmt.Errorf("failed to downsample to granularity 2: %v", err)
	}

	// 删除已聚合的粒度 1 数据
	result, err = d.db.Exec(`
		DELETE FROM traffic_stats
		WHERE granularity = 1 AND timestamp < ?
	`, sevenDaysAgo)

	if err != nil {
		return fmt.Errorf("failed to delete old granularity 1 data: %v", err)
	}

	deleted, _ = result.RowsAffected()
	log.Printf("Deleted %d granularity 1 records", deleted)

	// 3. 删除 30 天前的粒度 2 数据
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour).Unix()

	log.Println("Cleaning up old granularity 2 data")

	result, err = d.db.Exec(`
		DELETE FROM traffic_stats
		WHERE granularity = 2 AND timestamp < ?
	`, thirtyDaysAgo)

	if err != nil {
		return fmt.Errorf("failed to delete old granularity 2 data: %v", err)
	}

	deleted, _ = result.RowsAffected()
	log.Printf("Deleted %d granularity 2 records", deleted)

	// 主机流量数据降采样和清理
	log.Println("Starting host traffic downsampling: granularity 0 -> 1")

	_, err = d.db.Exec(`
		INSERT INTO host_traffic_stats
		(timestamp, host_ip, remote_ip, direction, bytes, packets, peak_rate, granularity, created_at)
		SELECT
			(timestamp / 3600) * 3600 as hour_timestamp,
			host_ip,
			remote_ip,
			direction,
			SUM(bytes) as total_bytes,
			SUM(packets) as total_packets,
			MAX(peak_rate) as max_peak_rate,
			1 as granularity,
			? as created_at
		FROM host_traffic_stats
		WHERE granularity = 0 AND timestamp < ?
		GROUP BY hour_timestamp, host_ip, remote_ip, direction
	`, now.Unix(), sixHoursAgo)

	if err != nil {
		return fmt.Errorf("failed to downsample host traffic to granularity 1: %v", err)
	}

	result, err = d.db.Exec(`
		DELETE FROM host_traffic_stats
		WHERE granularity = 0 AND timestamp < ?
	`, sixHoursAgo)

	if err != nil {
		return fmt.Errorf("failed to delete old host granularity 0 data: %v", err)
	}

	deleted, _ = result.RowsAffected()
	log.Printf("Deleted %d host granularity 0 records", deleted)

	log.Println("Starting host traffic downsampling: granularity 1 -> 2")

	_, err = d.db.Exec(`
		INSERT INTO host_traffic_stats
		(timestamp, host_ip, remote_ip, direction, bytes, packets, peak_rate, granularity, created_at)
		SELECT
			(timestamp / 86400) * 86400 as day_timestamp,
			host_ip,
			remote_ip,
			direction,
			SUM(bytes) as total_bytes,
			SUM(packets) as total_packets,
			MAX(peak_rate) as max_peak_rate,
			2 as granularity,
			? as created_at
		FROM host_traffic_stats
		WHERE granularity = 1 AND timestamp < ?
		GROUP BY day_timestamp, host_ip, remote_ip, direction
	`, now.Unix(), sevenDaysAgo)

	if err != nil {
		return fmt.Errorf("failed to downsample host traffic to granularity 2: %v", err)
	}

	result, err = d.db.Exec(`
		DELETE FROM host_traffic_stats
		WHERE granularity = 1 AND timestamp < ?
	`, sevenDaysAgo)

	if err != nil {
		return fmt.Errorf("failed to delete old host granularity 1 data: %v", err)
	}

	deleted, _ = result.RowsAffected()
	log.Printf("Deleted %d host granularity 1 records", deleted)

	result, err = d.db.Exec(`
		DELETE FROM host_traffic_stats
		WHERE granularity = 2 AND timestamp < ?
	`, thirtyDaysAgo)

	if err != nil {
		return fmt.Errorf("failed to delete old host granularity 2 data: %v", err)
	}

	deleted, _ = result.RowsAffected()
	log.Printf("Deleted %d host granularity 2 records", deleted)

	// 4. 执行 PRAGMA optimize 优化数据库
	if _, err := d.db.Exec("PRAGMA optimize;"); err != nil {
		log.Printf("Warning: failed to optimize database: %v", err)
	}

	log.Println("Downsampling and cleanup completed successfully")
	return nil
}

// QueryStats 查询指定时间范围的统计数据
func (d *Database) QueryStats(port uint16, rangeStr string) ([]map[string]interface{}, error) {
	var duration time.Duration
	var granularity int

	// 根据 range 选择合适的粒度
	switch rangeStr {
	case "15m":
		duration = 15 * time.Minute
		granularity = 0
	case "30m":
		duration = 30 * time.Minute
		granularity = 0
	case "60m", "1h":
		duration = 60 * time.Minute
		granularity = 0
	case "1d":
		duration = 24 * time.Hour
		granularity = 1
	case "3d":
		duration = 3 * 24 * time.Hour
		granularity = 1
	case "7d":
		duration = 7 * 24 * time.Hour
		granularity = 1
	case "30d":
		duration = 30 * 24 * time.Hour
		granularity = 2
	default:
		return nil, fmt.Errorf("invalid range: %s", rangeStr)
	}

	startTime := time.Now().Add(-duration).Unix()

	rows, err := d.db.Query(`
		SELECT
			timestamp,
			source_ip,
			direction,
			SUM(bytes) as total_bytes,
			SUM(packets) as total_packets,
			MAX(peak_rate) as max_rate
		FROM traffic_stats
		WHERE port = ? AND timestamp >= ? AND granularity = ?
		GROUP BY timestamp, source_ip, direction
		ORDER BY timestamp ASC
	`, port, startTime, granularity)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var timestamp int64
		var sourceIP, direction string
		var bytes, packets uint64
		var maxRate float64

		if err := rows.Scan(&timestamp, &sourceIP, &direction, &bytes, &packets, &maxRate); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"timestamp": timestamp,
			"time":      time.Unix(timestamp, 0).Format("2006-01-02 15:04:05"),
			"source_ip": sourceIP,
			"direction": direction,
			"bytes":     bytes,
			"packets":   packets,
			"peak_rate": maxRate,
		})
	}

	return results, nil
}

// QueryHostStats 查询主机流量统计数据
func (d *Database) QueryHostStats(hostIP string, rangeStr string) ([]map[string]interface{}, error) {
	var duration time.Duration
	var granularity int

	// 根据 range 选择合适的粒度
	switch rangeStr {
	case "15m":
		duration = 15 * time.Minute
		granularity = 0
	case "30m":
		duration = 30 * time.Minute
		granularity = 0
	case "60m", "1h":
		duration = 60 * time.Minute
		granularity = 0
	case "1d":
		duration = 24 * time.Hour
		granularity = 1
	case "3d":
		duration = 3 * 24 * time.Hour
		granularity = 1
	case "7d":
		duration = 7 * 24 * time.Hour
		granularity = 1
	case "30d":
		duration = 30 * 24 * time.Hour
		granularity = 2
	default:
		return nil, fmt.Errorf("invalid range: %s", rangeStr)
	}

	startTime := time.Now().Add(-duration).Unix()

	rows, err := d.db.Query(`
		SELECT
			timestamp,
			remote_ip,
			direction,
			SUM(bytes) as total_bytes,
			SUM(packets) as total_packets,
			MAX(peak_rate) as max_rate
		FROM host_traffic_stats
		WHERE host_ip = ? AND timestamp >= ? AND granularity = ?
		GROUP BY timestamp, remote_ip, direction
		ORDER BY timestamp ASC
	`, hostIP, startTime, granularity)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var timestamp int64
		var remoteIP, direction string
		var bytes, packets uint64
		var maxRate float64

		if err := rows.Scan(&timestamp, &remoteIP, &direction, &bytes, &packets, &maxRate); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"timestamp": timestamp,
			"time":      time.Unix(timestamp, 0).Format("2006-01-02 15:04:05"),
			"remote_ip": remoteIP,
			"direction": direction,
			"bytes":     bytes,
			"packets":   packets,
			"peak_rate": maxRate,
		})
	}

	return results, nil
}

// ============================================================================
// 辅助函数
// ============================================================================

// getListeningPorts 获取所有处于 LISTEN 状态的端口
func getListeningPorts() (map[uint16]bool, error) {
	ports := make(map[uint16]bool)

	// 读取 /proc/net/tcp 和 /proc/net/tcp6
	for _, file := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if i == 0 || line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}

			// 状态字段：0A 表示 LISTEN
			if fields[3] != "0A" {
				continue
			}

			// 解析本地地址和端口
			localAddr := fields[1]
			parts := strings.Split(localAddr, ":")
			if len(parts) != 2 {
				continue
			}

			portHex := parts[1]
			portInt, err := strconv.ParseInt(portHex, 16, 32)
			if err != nil {
				continue
			}

			ports[uint16(portInt)] = true
		}
	}

	// 同样处理 UDP
	for _, file := range []string{"/proc/net/udp", "/proc/net/udp6"} {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if i == 0 || line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}

			// UDP 的 LISTEN 状态是 07
			if fields[3] != "07" {
				continue
			}

			localAddr := fields[1]
			parts := strings.Split(localAddr, ":")
			if len(parts) != 2 {
				continue
			}

			portHex := parts[1]
			portInt, err := strconv.ParseInt(portHex, 16, 32)
			if err != nil {
				continue
			}

			ports[uint16(portInt)] = true
		}
	}

	return ports, nil
}

// getLocalIP 获取本机 IP 地址
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}

// getDefaultInterface 获取默认网络接口
func getDefaultInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				return iface.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

// ============================================================================
// 模块四：Web API 实现
// ============================================================================

func handleActivePorts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := aggregator.GetRealTimeStats()

	// 转换为数组格式
	result := make([]map[string]interface{}, 0, len(stats))
	for _, portData := range stats {
		result = append(result, portData)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

func handlePortStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	portStr := r.URL.Query().Get("port")
	rangeStr := r.URL.Query().Get("range")

	if portStr == "" || rangeStr == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "missing port or range parameter",
		})
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid port number",
		})
		return
	}

	data, err := database.QueryStats(uint16(port), rangeStr)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 计算汇总统计
	var totalBytes, totalPackets uint64
	var peakRate float64

	for _, record := range data {
		totalBytes += record["bytes"].(uint64)
		totalPackets += record["packets"].(uint64)
		if record["peak_rate"].(float64) > peakRate {
			peakRate = record["peak_rate"].(float64)
		}
	}

	avgRate := float64(0)
	if len(data) > 0 {
		avgRate = float64(totalBytes) / float64(len(data))
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"port":          port,
			"range":         rangeStr,
			"total_bytes":   totalBytes,
			"total_packets": totalPackets,
			"peak_rate":     peakRate,
			"average_rate":  avgRate,
			"timeseries":    data,
		},
	})
}

func handleActiveHosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := aggregator.GetRealTimeHostStats()

	// 转换为数组格式
	result := make([]map[string]interface{}, 0, len(stats))
	for _, hostData := range stats {
		result = append(result, hostData)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

func handleHostStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hostIP := r.URL.Query().Get("host_ip")
	rangeStr := r.URL.Query().Get("range")

	if hostIP == "" || rangeStr == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "missing host_ip or range parameter",
		})
		return
	}

	data, err := database.QueryHostStats(hostIP, rangeStr)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 计算汇总统计
	var totalBytes, totalPackets uint64
	var peakRate float64

	for _, record := range data {
		totalBytes += record["bytes"].(uint64)
		totalPackets += record["packets"].(uint64)
		if record["peak_rate"].(float64) > peakRate {
			peakRate = record["peak_rate"].(float64)
		}
	}

	avgRate := float64(0)
	if len(data) > 0 {
		avgRate = float64(totalBytes) / float64(len(data))
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"host_ip":       hostIP,
			"range":         rangeStr,
			"total_bytes":   totalBytes,
			"total_packets": totalPackets,
			"peak_rate":     peakRate,
			"average_rate":  avgRate,
			"timeseries":    data,
		},
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := embeddedFS.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// ============================================================================
// 主程序入口
// ============================================================================

func main() {
	log.Println("=== Linux Port Traffic Monitor Starting ===")

	// 1. 初始化内存聚合器
	aggregator = NewMemoryAggregator()
	log.Println("Memory aggregator initialized")

	// 2. 初始化数据库
	var err error
	database, err = NewDatabase("traffic_monitor.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized with WAL mode")

	// 3. 获取监听端口
	listenPorts, err = getListeningPorts()
	if err != nil {
		log.Fatalf("Failed to get listening ports: %v", err)
	}
	log.Printf("Monitoring %d listening ports", len(listenPorts))

	// 4. 启动定时任务：每分钟持久化快照
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			// 持久化端口快照
			snapshots := aggregator.GetSnapshot()
			if len(snapshots) > 0 {
				if err := database.BatchInsert(snapshots); err != nil {
					log.Printf("Error inserting port snapshots: %v", err)
				} else {
					log.Printf("Persisted %d port snapshots", len(snapshots))
				}
			}

			// 持久化主机快照
			hostSnapshots := aggregator.GetHostSnapshot()
			if len(hostSnapshots) > 0 {
				if err := database.BatchInsertHostSnapshots(hostSnapshots); err != nil {
					log.Printf("Error inserting host snapshots: %v", err)
				} else {
					log.Printf("Persisted %d host snapshots", len(hostSnapshots))
				}
			}

			// 更新监听端口列表
			portsMu.Lock()
			listenPorts, _ = getListeningPorts()
			portsMu.Unlock()
		}
	}()

	// 5. 启动降采样与清理任务：每小时执行一次
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// 启动时立即执行一次
		if err := database.DownsampleAndCleanup(); err != nil {
			log.Printf("Error during initial cleanup: %v", err)
		}

		for range ticker.C {
			if err := database.DownsampleAndCleanup(); err != nil {
				log.Printf("Error during downsampling: %v", err)
			}
		}
	}()

	// 6. 启动数据包捕获
	iface, err := getDefaultInterface()
	if err != nil {
		log.Fatalf("Failed to get default interface: %v", err)
	}

	go func() {
		if err := startPacketCapture(iface); err != nil {
			log.Fatalf("Packet capture failed: %v", err)
		}
	}()

	// 7. 启动 Web 服务器
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/ports/active", handleActivePorts)
	http.HandleFunc("/api/ports/stats", handlePortStats)
	http.HandleFunc("/api/hosts/active", handleActiveHosts)
	http.HandleFunc("/api/hosts/stats", handleHostStats)

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Web server started on http://0.0.0.0:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 8. 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")

	// 最后一次持久化
	snapshots := aggregator.GetSnapshot()
	if len(snapshots) > 0 {
		database.BatchInsert(snapshots)
	}

	log.Println("Shutdown complete")
}
