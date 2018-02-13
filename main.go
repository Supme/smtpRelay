package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/supme/smtpRelay/model"
	"github.com/supme/smtpRelay/sender"
	"github.com/supme/smtpRelay/server"
	"os"
)

const version = "v0.2.6"

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

	fmt.Println("Start SMTP sender")
	sender.Run()

	fmt.Println("Starting SMTP server...")
	server.Run()
}
