package main

import (
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/supme/smtpRelay/model"
	"github.com/supme/smtpRelay/sender"
	"github.com/supme/smtpRelay/server"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "c", "./config.ini", "Config file")
	flag.Parse()
}

func main() {
	if _, err := toml.DecodeFile(configFile, &model.Config); err != nil {
		panic(err)
	}

	err := model.OpenQueueDb()
	if err != nil {
		panic(err)
	}
	defer model.QueueDb.Close()

	err = model.OpenStatusDb()
	if err != nil {
		panic(err)
	}
	defer model.QueueDb.Close()

	sender.Run()

	server.Run()
}
