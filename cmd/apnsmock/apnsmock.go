package main

import (
	"flag"

	"github.com/kayac/Gunfish/config"
	"github.com/kayac/Gunfish/apns"
)

func main() {
	var (
		confFile string
	)
	flag.StringVar(&confFile, "c", "./test/gunfish_test.toml", "config file")
	flag.Parse()

	config, err := config.LoadConfig(confFile)
	if err != nil {
		return
	}
	apns.StartAPNSMockServer(config.Apns.CertFile, config.Apns.KeyFile)
}
