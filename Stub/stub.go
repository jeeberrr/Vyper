package main

import (
	_ "embed"
	"vyper/Stub/config"
)

//go:embed config.ini
var configfile string

func main() {
	vyper := config.Parse(configfile)
	vyper.Run()
}
