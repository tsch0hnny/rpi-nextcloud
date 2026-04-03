package exec

import (
	"net"
	"os"
	osexec "os/exec"
	"runtime"
	"strings"
	"syscall"
)

// SystemInfo holds detected system information.
type SystemInfo struct {
	IPAddress    string
	Hostname     string
	OS           string
	Architecture string
	IsRaspberry  bool
	IsDebian     bool
	HasApt       bool
	HasApache    bool
	HasMySQL     bool
	HasPHP       bool
	HasSystemd   bool
	DiskFreeGB   float64
}

// DetectSystem gathers system information.
func DetectSystem() SystemInfo {
	info := SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}

	info.IPAddress = detectIP()

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	// Check if running on Raspberry Pi
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		content := string(data)
		info.IsRaspberry = strings.Contains(content, "Raspberry") || strings.Contains(content, "BCM")
	}
	if data, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		if strings.Contains(string(data), "Raspberry") {
			info.IsRaspberry = true
		}
	}

	// Check Debian-based
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := strings.ToLower(string(data))
		info.IsDebian = strings.Contains(content, "debian") ||
			strings.Contains(content, "ubuntu") ||
			strings.Contains(content, "raspbian")
	}

	// Check installed tools
	info.HasApt = commandExists("apt")
	info.HasApache = commandExists("apache2")
	info.HasMySQL = commandExists("mysql") || commandExists("mariadb")
	info.HasPHP = commandExists("php")
	info.HasSystemd = commandExists("systemctl")

	// Check disk space on /var
	info.DiskFreeGB = diskFreeGB("/var")

	return info
}

func detectIP() string {
	if out, err := osexec.Command("hostname", "-I").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) > 0 {
			return parts[0]
		}
	}

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		addr := conn.LocalAddr().(*net.UDPAddr)
		return addr.IP.String()
	}

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}

func commandExists(name string) bool {
	_, err := osexec.LookPath(name)
	return err == nil
}

func diskFreeGB(path string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return -1
	}
	return float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
}
