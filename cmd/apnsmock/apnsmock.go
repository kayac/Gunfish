package main

import (
	"flag"
	"github.com/kayac/Gunfish"
)

func main() {
	var (
		confFile string
	)
	flag.StringVar(&confFile, "c", "./test/gunfish_test.toml", "config file")
	flag.Parse()

	config, err := gunfish.LoadConfig(confFile)
	if err != nil {
		return
	}
	gunfish.StartAPNSMockServer(config)
}
