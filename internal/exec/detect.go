package exec

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SystemInfo holds detected system information.
type SystemInfo struct {
	IPAddress    string
	Hostname     string
	OS           string
	Architecture string
	IsRaspberry  bool
	HasApache    bool
	HasMySQL     bool
	HasPHP       bool
}

// DetectSystem gathers system information.
func DetectSystem() SystemInfo {
	info := SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}

	// Detect IP address
	info.IPAddress = detectIP()

	// Detect hostname
	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	// Check if running on Raspberry Pi
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		content := string(data)
		info.IsRaspberry = strings.Contains(content, "Raspberry") || strings.Contains(content, "BCM")
	}
	// Also check device-tree model
	if data, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		if strings.Contains(string(data), "Raspberry") {
			info.IsRaspberry = true
		}
	}

	// Check installed services
	info.HasApache = commandExists("apache2")
	info.HasMySQL = commandExists("mysql") || commandExists("mariadb")
	info.HasPHP = commandExists("php")

	return info
}

// detectIP finds the primary local IP address.
func detectIP() string {
	// First try hostname -I (common on Linux)
	if out, err := exec.Command("hostname", "-I").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Fallback: connect to a remote address to determine local IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		addr := conn.LocalAddr().(*net.UDPAddr)
		return addr.IP.String()
	}

	// Final fallback
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
	_, err := exec.LookPath(name)
	return err == nil
}
