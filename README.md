# Linux Traffic Monitor

English | [简体中文](README_CN.md)

Real-time network traffic monitoring system for Linux with dual monitoring modes: port-level and host-level traffic analysis.

## Features

### Dual Monitoring Modes
- **Port Monitoring**: Track traffic for specific listening ports
- **Host Monitoring**: Monitor all traffic by local IP addresses (multi-NIC support)

### Real-time Visualization
- Web-based dashboard with interactive charts (ECharts)
- Auto-refresh capability
- Toggle between Port and Host views
- Dark theme with professional styling

### Time Range Analysis
- Multiple time ranges: 15m, 30m, 1h, 1d, 3d, 7d, 30d
- Historical data query with automatic downsampling
- Peak rate tracking

### Data Management
- SQLite database with WAL mode for high concurrency
- 3-tier automatic downsampling (minute → hour → day)
- Efficient in-memory aggregation
- Automatic data cleanup

### Traffic Tracking
- Inbound and outbound traffic separation
- Bytes and packets counting
- Per-source/remote IP tracking
- Real-time rate calculation

## Quick Start

### One-Line Installation (Recommended)

**For AMD64 (x86_64) systems:**
```bash
curl -fsSL https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh | sudo bash
```

**For ARM64/ARM systems:**
```bash
# The script will automatically build from source
curl -fsSL https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh | sudo bash
```

Or download and inspect the script first:

```bash
wget https://raw.githubusercontent.com/xiaoxinmm/linux-traffic-monitor/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

### What the Installation Script Does

The installation script will:
1. **Detect your system**
   - Automatically identify Linux distribution (Ubuntu/Debian/CentOS/RHEL/Fedora/Arch)
   - Detect system architecture (x86_64/ARM64/ARM)

2. **Install dependencies**
   - Install libpcap runtime library
   - Install curl, wget, tar utilities

3. **Get the program**
   - **AMD64**: Download precompiled binary from GitHub Releases (fast!)
   - **ARM/ARM64**: Automatically build from source (requires Go and build tools)

4. **Configure service**
   - Create systemd service
   - Configure auto-start option

5. **Start service**
   - Optionally start the monitoring service immediately

### Supported Platforms

**Architectures:**
- ✅ x86_64 (amd64) - **Precompiled binary available**
- ✅ ARM64 (aarch64) - Built from source
- ✅ ARM (armv7l) - Built from source

**Distributions:**
- Ubuntu / Debian
- CentOS / RHEL / Fedora
- Arch Linux / Manjaro

### Manual Installation

#### Option 1: Download Precompiled Binary (AMD64 Only)

Download the latest release for x86_64 systems:

```bash
# Download latest AMD64 binary
wget https://github.com/xiaoxinmm/linux-traffic-monitor/releases/latest/download/traffic-monitor-v1.1.0-linux-amd64.tar.gz

# Extract
tar -xzf traffic-monitor-v1.1.0-linux-amd64.tar.gz

# Move to system path
sudo mv traffic-monitor /usr/local/bin/

# Add execute permission
sudo chmod +x /usr/local/bin/traffic-monitor
```

Install libpcap dependency:

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

Then run:
```bash
sudo traffic-monitor
```

#### Option 2: Build from Source (All Architectures)

##### Requirements

- Linux system
- Go 1.21 or higher
- libpcap-dev
- Root privileges (for packet capture)

##### Install Build Dependencies

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

##### Build and Run

```bash
# Clone the repository
git clone https://github.com/xiaoxinmm/linux-traffic-monitor.git
cd linux-traffic-monitor

# Build
go build -o traffic-monitor main.go

# Run (requires root for packet capture)
sudo ./traffic-monitor
```

The web interface will be available at `http://localhost:8080`

## Usage

### Using Systemd Service (Recommended)

If you used the installation script, the monitor is installed as a systemd service:

```bash
# Start the monitor
sudo systemctl start traffic-monitor

# Enable auto-start on boot
sudo systemctl enable traffic-monitor

# Check status
sudo systemctl status traffic-monitor

# View logs
sudo journalctl -u traffic-monitor -f

# Stop the monitor
sudo systemctl stop traffic-monitor

# Restart the monitor
sudo systemctl restart traffic-monitor
```

### Running Manually

```bash
sudo ./traffic-monitor
```

By default, the monitor:
- Listens on all network interfaces
- Captures traffic on ports: 22, 80, 443, 3306, 6379, 8080, 9090
- Serves web UI on port 8080
- Stores data in `traffic.db`

### Accessing the Dashboard

Open your browser and navigate to:
```
http://your-server-ip:8080
```

### Switching Views

Use the toggle buttons at the top of the dashboard:
- **Ports**: View traffic by listening port
- **Hosts**: View traffic by local IP address

### Querying Historical Data

1. Select a port or host from the dropdown
2. Choose a time range (15m to 30d)
3. Click "Query" to view the traffic chart

## API Endpoints

### Port Monitoring

- `GET /api/ports/active` - Get all active ports with real-time stats
- `GET /api/ports/stats?port=<port>&range=<range>` - Query historical port traffic

### Host Monitoring

- `GET /api/hosts/active` - Get all active hosts with real-time stats
- `GET /api/hosts/stats?host=<ip>&range=<range>` - Query historical host traffic

### Parameters

- `port`: Port number (e.g., 80, 443)
- `host`: Local IP address (e.g., 192.168.1.100)
- `range`: Time range
  - `15m`, `30m`, `1h` - Recent data (minute granularity)
  - `1d`, `3d`, `7d` - Daily data (hour granularity)
  - `30d` - Monthly data (day granularity)

### Response Format

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

## Architecture

### Data Flow

```
Network Packets → Pcap Capture → Packet Processing
                                        ↓
                                  Memory Aggregator
                                   (Real-time)
                                        ↓
                              ┌─────────┴─────────┐
                              ↓                   ↓
                        Port Stats          Host Stats
                              ↓                   ↓
                        SQLite Database (WAL mode)
                              ↓
                    Automatic Downsampling
                    (minute → hour → day)
```

### Downsampling Strategy

- **Granularity 0 (Minute)**: Raw data, kept for 2 hours
- **Granularity 1 (Hour)**: Aggregated hourly, kept for 8 days
- **Granularity 2 (Day)**: Aggregated daily, kept for 31 days

### Database Schema

**port_traffic_stats**
- Tracks traffic per port, source IP, and direction
- Indexed on (port, timestamp, granularity)

**host_traffic_stats**
- Tracks traffic per local IP, remote IP, and direction
- Indexed on (host_ip, timestamp, granularity)

## Configuration

Edit `main.go` to customize monitored ports:

```go
// Monitored ports
var listenPorts = map[int]bool{
    22:   true,  // SSH
    80:   true,  // HTTP
    443:  true,  // HTTPS
    3306: true,  // MySQL
    6379: true,  // Redis
    8080: true,  // Custom
    9090: true,  // Custom
}
```

After editing, rebuild with `./build.sh`

## Troubleshooting

### Permission Denied

The monitor requires root privileges for packet capture:
```bash
sudo ./traffic-monitor
```

Or grant CAP_NET_RAW capability:
```bash
sudo setcap cap_net_raw+ep ./traffic-monitor
./traffic-monitor
```

### Port Already in Use

If port 8080 is already in use, modify the web server port in `main.go` and rebuild.

### No Traffic Captured

1. Check if the monitored ports are actually receiving traffic
2. Verify network interface is up: `ip link show`
3. Check firewall rules: `sudo iptables -L`

### Database Locked

If you see "database is locked" errors:
1. Stop all running instances
2. Remove the lock: `rm traffic.db-wal traffic.db-shm`
3. Restart the monitor

## Performance

- Memory usage: ~50-100MB (depends on traffic volume)
- CPU usage: ~5-10% on moderate traffic
- Disk I/O: Minimal (WAL mode + periodic batch writes)
- Tested with: 10K packets/sec sustained traffic

## Security Considerations

- The monitor captures packet headers only (no payload)
- Web interface has no authentication (use firewall or reverse proxy)
- Database contains IP addresses (consider privacy regulations)
- Runs as root (required for pcap, isolate if possible)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

- Issues: https://github.com/xiaoxinmm/linux-traffic-monitor/issues
- Discussions: https://github.com/xiaoxinmm/linux-traffic-monitor/discussions
