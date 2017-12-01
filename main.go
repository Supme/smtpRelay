package main

import (
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/supme/smtpRelay/model"
	"github.com/supme/smtpRelay/sender"
	"github.com/supme/smtpRelay/server"
)

var configFile string

var config struct {
	QueueDbDialect        string   `toml:"QueueDbDialect"`
	QueueDbConnect        string   `toml:"QueueDbConnect"`
	StatusDbDialect       string   `toml:"StatusDbDialect"`
	StatusDbConnect       string   `toml:"StatusDbConnect"`
	Hostname              string   `toml:"Hostname"`
	AllowIP               []string `toml:"AllowIP"`
	AllowDomains          []string `toml:"AllowDomains"`
	SmtpListenAddr        string   `toml:"SmtpListenAddr"`
	MaxRepeatSend         uint     `toml:"MaxRepeatSend"`
	RepeatIntervalMinutes uint     `toml:"RepeatIntervalMinutes"`
}

func init() {
	flag.StringVar(&configFile, "c", "./config.ini", "Config file")
	flag.Parse()
}

func main() {
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		panic(err)
	}

	err := model.OpenQueueDb(config.QueueDbDialect, config.QueueDbConnect)
	if err != nil {
		panic(err)
	}
	defer model.QueueDb.Close()

	err = model.OpenStatusDb(config.StatusDbDialect, config.StatusDbConnect)
	if err != nil {
		panic(err)
	}
	defer model.QueueDb.Close()

	model.Hostname = config.Hostname

	model.SetAllowIP(config.AllowIP...)
	model.SetAllowDomains(config.AllowDomains...)

	model.MaxRepeatSend = config.MaxRepeatSend
	model.RepeatIntervalMinutes = config.RepeatIntervalMinutes

	sender.Run()

	server.Run(config.SmtpListenAddr)
}
