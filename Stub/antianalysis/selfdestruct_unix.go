//go:build linux || darwin

package antianalysis

import (
	"os"
	"os/exec"
	"syscall"
)

func SelfDestruct() {
	path, _ := os.Executable()

	// Spawns a detached shell that waits 3 seconds and deletes the executable
	cmd := exec.Command("sh", "-c", "sleep 3 && rm -f \""+path+"\"")

	// Setsid creates a new session so the child process survives the parent's exit
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	cmd.Start()
	os.Exit(0)
}
