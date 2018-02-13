package main

import (
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/supme/smtpRelay/model"
	"github.com/supme/smtpRelay/sender"
	"github.com/supme/smtpRelay/server"
	"fmt"
	"os"
)

const version = "v0.2.5"

var configFile string

func init() {
	ver := flag.Bool("v", false, "print current version")
	flag.StringVar(&configFile, "c", "./config.ini", "Config file")
	flag.Parse()
	if *ver {
		fmt.Println(version)
		os.Exit(0)
	}
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
	defer model.StatusDb.Close()

	sender.Run()

	server.Run()
}
