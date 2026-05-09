//go:build windows

package exfiltration

func getSysDescription(data *DataStruct) string {
	var description string
	description = "OS VERSION: " + data.SysInfo.OSVersion + "\nIP ADDRESS: " + data.SysInfo.ExposedIP + "\nLOCATION:   " + data.SysInfo.Location
	return description
}
