package sender

import (
	"encoding/base64"
	"github.com/supme/directEmail"
	"github.com/supme/smtpRelay/model"
	"sync"
	"time"
	"os"
	"log"
)

// Run start sending queue emails
func Run() {
	if model.Config.Hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			log.Print(err)
		}
		model.Config.Hostname = h
	}
	go sendQueue()
	go resendQueue()
}

func sendQueue() {
	for {
		emails := model.GetNewQueue(model.Config.SendStream)
		if len(emails) == 0 {
			time.Sleep(10 * time.Second)
		} else {
			send(emails)
		}
	}
}

func resendQueue() {
	for {
		emails := model.GetRepeatQueue(model.Config.ResendStream)
		if len(emails) == 0 {
			time.Sleep(10 * time.Second)
		} else {
			send(emails)
		}
	}
}

func stringToByte(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func send(emails []model.Queue) {
	var wg sync.WaitGroup
	for i := range emails {
		wg.Add(1)
		go func(wg *sync.WaitGroup, email model.Queue) {
			defer wg.Done()
			dataByte, err := stringToByte(email.Data)
			if err != nil {
				email.LaterStatus = "550 " + err.Error()
				return
			}
			sender := directEmail.New()
			sender.Host = model.Config.Hostname
			sender.FromEmail = email.From
			sender.ToEmail = email.Rcpt
			sender.SetRawMessageBytes(dataByte)
			err = sender.Send()
			if err != nil {
				email.LaterStatus = err.Error()
			} else {
				email.LaterStatus = "250 2.0.0 Ok"
			}
			model.SetStatus(&email)
		}(&wg, emails[i])
	}
	wg.Wait()
}
